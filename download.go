package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
)

const (
	MinChunkSize   = 256 * 1024
	MaxChunkSize   = 4 * 1024 * 1024
	DefaultThreads = 16
	BufferSize     = 32 * 1024
	MaxRetries     = 3
	ProgressExt    = ".gdl"
)

type chunkTask struct {
	start   int64
	attempt int
}

type Downloader struct {
	URL       string
	FilePath  string
	FileSize  int64
	Threads   int
	ChunkSize int64
	UseProxy  bool

	ctx    context.Context
	cancel context.CancelFunc

	client      *http.Client
	progressBar *widget.ProgressBar
	sd          *SettingsData

	downloaded   atomic.Int64
	progressFile string
	startTime    time.Time

	mu        sync.Mutex
	completed map[int64]bool

	Done chan error
}

func NewDownloader(sd *SettingsData, url, savePath string, threads int, pb *widget.ProgressBar) *Downloader {
	if threads <= 0 {
		threads = DefaultThreads
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Downloader{
		URL:          url,
		FilePath:     savePath,
		Threads:      threads,
		UseProxy:     true,
		ctx:          ctx,
		cancel:       cancel,
		progressBar:  pb,
		sd:           sd,
		completed:    make(map[int64]bool),
		progressFile: savePath + ProgressExt,
		Done:         make(chan error, 1),
	}
}

func (d *Downloader) Start() {
	go d.run()
}

func (d *Downloader) Cancel() {
	d.cancel()
}

func (d *Downloader) Progress() float64 {
	if d.FileSize <= 0 {
		return 0
	}
	return float64(d.downloaded.Load()) / float64(d.FileSize) * 0.9
}

func (d *Downloader) run() {
	defer close(d.Done)
	defer d.cleanup()

	d.startTime = time.Now()
	d.client = d.newHTTPClient()

	logger.Debugf("下载开始: URL=%s, UseProxy=%v, FileSize=%d", d.URL, d.UseProxy, d.FileSize)

	if d.FileSize <= 0 {
		logger.Infof("FileSize未知, 尝试HEAD请求获取文件大小...")
		size, err := d.getFileSize()
		if err != nil {
			logger.Errorf("getFileSize失败: %v", err)
			d.Done <- fmt.Errorf("get file size: %w", err)
			return
		}
		d.FileSize = size
		logger.Debugf("HEAD获取到文件大小: %s", humanize.Bytes(uint64(d.FileSize)))
	}

	d.ChunkSize = d.calcChunkSize()

	logger.Debugf("下载开始: %s (大小: %s, 线程: %d, 分块: %s)",
		filepath.Base(d.FilePath), humanize.Bytes(uint64(d.FileSize)),
		d.Threads, humanize.Bytes(uint64(d.ChunkSize)))

	d.loadProgress()

	if err := os.MkdirAll(filepath.Dir(d.FilePath), 0755); err != nil {
		d.Done <- err
		return
	}
	d.preAllocate()

	tasks := make(chan chunkTask, d.Threads*2)
	var pending atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < d.Threads; i++ {
		wg.Add(1)
		go d.worker(tasks, &wg, &pending)
	}

	stopProgress := make(chan struct{})
	go d.progressUpdater(stopProgress)
	defer close(stopProgress)

	for start := int64(0); start < d.FileSize; start += d.ChunkSize {
		if d.ctx.Err() != nil {
			break
		}
		d.mu.Lock()
		done := d.completed[start]
		d.mu.Unlock()
		if done {
			d.downloaded.Add(d.realChunkSize(start))
			continue
		}
		pending.Add(1)
		tasks <- chunkTask{start: start}
	}

	for pending.Load() > 0 && d.ctx.Err() == nil {
		time.Sleep(100 * time.Millisecond)
	}

	close(tasks)
	wg.Wait()

	if d.ctx.Err() != nil {
		d.Done <- d.ctx.Err()
		return
	}

	elapsed := time.Since(d.startTime).Round(time.Second)
	avgSpeed := float64(d.FileSize) / elapsed.Seconds()
	logger.Debugf("下载完成! 用时: %v, 平均速度: %s/s", elapsed, humanize.Bytes(uint64(avgSpeed)))

	if d.downloaded.Load() >= d.FileSize {
		d.Done <- nil
	} else {
		d.Done <- fmt.Errorf("下载不完整: %d/%d", d.downloaded.Load(), d.FileSize)
	}
}

func (d *Downloader) worker(tasks chan chunkTask, wg *sync.WaitGroup, pending *atomic.Int64) {
	defer wg.Done()
	for task := range tasks {
		if d.ctx.Err() != nil {
			pending.Add(-1)
			continue
		}
		err := d.downloadChunk(task.start)
		if err != nil {
			logger.Debugf("分块 %d 请求失败: %v (attempt=%d)", task.start, err, task.attempt)
			if task.attempt < MaxRetries && d.ctx.Err() == nil {
				backoff := time.Duration(1<<task.attempt) * 500 * time.Millisecond
				logger.Debugf("分块 %d 重试(%d/%d): %v", task.start, task.attempt+1, MaxRetries, err)
				time.Sleep(backoff)
				task.attempt++
				tasks <- task
				continue
			}
			logger.Errorf("分块 %d 下载失败: %v", task.start, err)
		}
		pending.Add(-1)
	}
}

func (d *Downloader) progressUpdater(stop <-chan struct{}) {
	if d.progressBar == nil {
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			fyne.DoAndWait(func() {
				d.progressBar.SetValue(d.Progress())
			})
		}
	}
}

func (d *Downloader) downloadChunk(start int64) error {
	end := start + d.ChunkSize - 1
	if end >= d.FileSize {
		end = d.FileSize - 1
	}

	reqURL := d.URL
	if d.UseProxy && d.sd != nil {
		if proxy := getString(d.sd.ghProxy); proxy != "" {
			if getString(d.sd.proxyType) == "GH-PROXY" {
				reqURL = pathJoin(proxy, reqURL)
			}
		}
	}

	req, err := http.NewRequestWithContext(d.ctx, "GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	if start == 0 {
		logger.Debugf("分块下载请求: URL=%s, Range=bytes=%d-%d", reqURL, start, end)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.OpenFile(d.FilePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, BufferSize)
	n, err := io.CopyBuffer(io.NewOffsetWriter(f, start), resp.Body, buf)
	if err != nil {
		return fmt.Errorf("读取数据失败: %w", err)
	}
	d.downloaded.Add(n)
	d.markCompleted(start)
	return nil
}

func (d *Downloader) newHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}

	if d.UseProxy && d.sd != nil {
		ghProxy := getString(d.sd.ghProxy)
		proxyType := getString(d.sd.proxyType)
		logger.Infof("HTTP客户端代理配置: proxyType=%s, ghProxy=%s", proxyType, ghProxy)
		if ghProxy != "" && proxyType != "GH-PROXY" {
			if proxyType == "HTTP(S)" && !strings.HasPrefix(ghProxy, "http") {
				ghProxy = "http://" + ghProxy
			} else if proxyType == "SOCKS5" && !strings.HasPrefix(ghProxy, "socks5") {
				ghProxy = "socks5://" + ghProxy
			}
			if u, err := url.Parse(ghProxy); err == nil {
				transport.Proxy = http.ProxyURL(u)
				logger.Infof("已设置代理: %s", ghProxy)
			} else {
				logger.Warnf("代理URL解析失败: %s, err=%v", ghProxy, err)
			}
		} else {
			logger.Infof("未设置代理 (proxyType=%s 或 ghProxy为空)", proxyType)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

func (d *Downloader) getFileSize() (int64, error) {
	reqURL := d.URL
	if d.UseProxy && d.sd != nil {
		if proxy := getString(d.sd.ghProxy); proxy != "" {
			if getString(d.sd.proxyType) == "GH-PROXY" {
				reqURL = pathJoin(proxy, reqURL)
			}
		}
	}
	req, err := http.NewRequestWithContext(d.ctx, "HEAD", reqURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength <= 0 {
		return 0, fmt.Errorf("无法获取文件大小")
	}
	return resp.ContentLength, nil
}

func (d *Downloader) realChunkSize(start int64) int64 {
	if start+d.ChunkSize > d.FileSize {
		return d.FileSize - start
	}
	return d.ChunkSize
}

func (d *Downloader) preAllocate() {
	f, err := os.OpenFile(d.FilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_ = f.Truncate(d.FileSize)
}

func (d *Downloader) markCompleted(start int64) {
	d.mu.Lock()
	d.completed[start] = true
	d.mu.Unlock()
}

func (d *Downloader) loadProgress() {
	data, err := os.ReadFile(d.progressFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		start, err := strconv.ParseInt(line, 10, 64)
		if err != nil || start < 0 || start >= d.FileSize {
			continue
		}
		d.completed[start] = true
	}
	if len(d.completed) > 0 {
		logger.Infof("恢复断点续传: %d 个分块已完成", len(d.completed))
	}
}

func (d *Downloader) cleanup() {
	_ = os.Remove(d.progressFile)
}

func (d *Downloader) calcChunkSize() int64 {
	if d.FileSize <= 0 {
		return MinChunkSize
	}
	ideal := d.FileSize / int64(d.Threads*2)
	if ideal < MinChunkSize {
		return MinChunkSize
	}
	if ideal > MaxChunkSize {
		return MaxChunkSize
	}
	return ideal
}
