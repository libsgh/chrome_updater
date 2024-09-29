package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"github.com/bodgit/sevenzip"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	versionDLL                 = syscall.NewLazyDLL("version.dll")
	procGetFileVersionInfo     = versionDLL.NewProc("GetFileVersionInfoW")
	procGetFileVersionInfoSize = versionDLL.NewProc("GetFileVersionInfoSizeW")
	procVerQueryValue          = versionDLL.NewProc("VerQueryValueW")
)

// url转换
func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		fyne.LogError("Could not parse URL", err)
	}

	return link
}

// 路径是否合法
func isValidPath(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	cleanAbsPath := filepath.Clean(absPath)
	cleanPath := filepath.Clean(path)
	return cleanAbsPath == cleanPath && dirExist(path)
}

// 字符串是否是数字
func isNumeric(str string) bool {
	_, err := strconv.ParseFloat(str, 64)
	return err == nil
}

// 格式化文件大小
func formatFileSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d Bytes", size)
	}
}

// 获取文件大小
func getFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("服务器返回错误： %v", resp.Status)
	}

	return resp.ContentLength, nil
}

// 下载文件块
func downloadChunk(url string, start, end int64) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("服务器不支持分块下载：%v", resp.Status)
	}

	// 创建一个临时文件用于保存下载的文件块
	chunkPath := fmt.Sprintf("chunk_%d_%d.tmp", start, end)
	chunkFile, err := os.Create(chunkPath)
	if err != nil {
		return "", err
	}
	defer chunkFile.Close()

	_, err = io.Copy(chunkFile, resp.Body)
	if err != nil {
		return "", err
	}

	return chunkPath, nil
}

// 合并文件块
func mergeChunk(chunkPath string, output *os.File) error {
	chunkFile, err := os.Open(chunkPath)
	if err != nil {
		return err
	}
	defer chunkFile.Close()

	_, err = io.Copy(output, chunkFile)
	if err != nil {
		return err
	}

	return nil
}

// 获取文件名
func getFileName(fileURL string) string {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		fmt.Println("Failed to parse URL:", err)
		return ""
	}
	filename := path.Base(parsedURL.Path)
	return filename
}

func unzip(zipFile string, filterNames ...string) {
	parentPath := filepath.Dir(zipFile)
	// 打开 ZIP 文件
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		logger.Errorln("无法打开 ZIP 文件:", err)
		return
	}
	defer r.Close()
	// 遍历 ZIP 文件中的文件并解压指定文件名的文件
	for _, file := range r.File {
		for _, targetFileName := range filterNames {
			if file.Name == targetFileName {
				rc, err := file.Open()
				if err != nil {
					logger.Errorln("无法打开 ZIP 文件中的文件:", err)
					return
				}
				defer rc.Close()

				// 创建解压后的文件
				newFile, err := os.Create(filepath.Join(parentPath, file.Name))
				if err != nil {
					logger.Errorln("无法创建解压后的文件:", err)
					return
				}
				defer newFile.Close()

				// 将 ZIP 文件中的内容解压到新文件中
				_, err = io.Copy(newFile, rc)
				if err != nil {
					fmt.Println("无法解压 ZIP 文件中的内容:", err)
					return
				}
				logger.Errorln("解压文件:", file.Name)
			}
		}
	}
	logger.Debug("解压完成.")
}

// 7z解压缩
func UnCompress7z(filePath, targetDir string) {
	r, err := sevenzip.OpenReader(filePath)
	if err != nil {
		logger.Panic(err)
	}
	defer r.Close()
	for _, file := range r.File {
		rc, err := file.Open()
		if err != nil {
			logger.Panic(err)
		}
		defer rc.Close()
		fp := path.Join(targetDir, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(fp, os.ModePerm)
		} else {
			outputFile, err := os.Create(fp)
			if err != nil {
				logger.Panic(err)
			}
			defer outputFile.Close()
			buf := make([]byte, 1*1024*1024)
			_, err = io.CopyBuffer(outputFile, rc, buf)
			//_, err = io.Copy(outputFile, rc)
		}
	}
}

func UnCompress7zFilter(filePath, targetDir, filterName string) {
	r, err := sevenzip.OpenReader(filePath)
	if err != nil {
		logger.Panic(err)
	}
	defer r.Close()
	for _, file := range r.File {
		if strings.HasPrefix(file.Name, filterName) {
			rc, err := file.Open()
			if err != nil {
				logger.Panic(err)
			}
			defer rc.Close()
			fp := path.Join(targetDir, file.Name)
			if file.FileInfo().IsDir() {
				os.MkdirAll(fp, os.ModePerm)
			} else {
				outputFile, err := os.Create(fp)
				if err != nil {
					logger.Panic(err)
				}
				defer outputFile.Close()
				_, err = io.Copy(outputFile, rc)
			}
		}
	}
}

// 计算文件SHA1
func sumFileSHA1(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return ""
	}
	defer file.Close()
	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		fmt.Println("读取文件错误:", err)
		return ""
	}
	hashValue := hash.Sum(nil)
	hashString := hex.EncodeToString(hashValue)
	return strings.ToUpper(hashString)
}

// 检查文件是否存在
func fileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			fmt.Println("发生错误:", err)
		}
		return false
	}
	return true
}
func getString(v binding.String) string {
	gv, _ := v.Get()
	return gv
}
func getBool(v binding.Bool) bool {
	gv, _ := v.Get()
	return gv
}
func getStringList(v binding.StringList) []string {
	gv, _ := v.Get()
	return gv
}
func isProcessExist(appName string) bool {
	appary := make(map[string]int)
	cmd := exec.Command("cmd", "/C", "tasklist")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	output, _ := cmd.Output()
	n := strings.Index(string(output), "System")
	if n == -1 {
		fmt.Println("no find")
		os.Exit(1)
	}
	data := string(output)[n:]
	fields := strings.Fields(data)
	for k, v := range fields {
		if v == appName {
			appary[appName], _ = strconv.Atoi(fields[k+1])

			return true
		}
	}
	return false
}
func alertInfo(message string, win fyne.Window) {
	cnf := dialog.NewInformation(LoadString("DialogTooltipTitle"), message, win)
	cnf.SetDismissText(LoadString("DialogCloseLabel"))
	cnf.Show()
}
func alertConfirm(message string, callback func(bool), win fyne.Window) {
	cnf := dialog.NewConfirm(LoadString("DialogConfirmLabel"), message, func(b bool) {
		callback(b)
	}, win)
	cnf.SetDismissText(LoadString("DialogCloseLabel"))
	cnf.SetConfirmText(LoadString("DialogConfirmLabel"))
	cnf.Show()
}
func dirExist(dir string) bool {
	// 打开当前目录
	dirHandle, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer dirHandle.Close()
	return true
}

// 获取系统信息
func getInfo() SysInfo {
	goarch := "x64"
	goos := "win"
	if runtime.GOARCH == "amd64" {
		goarch = "x64"
	} else if runtime.GOARCH == "386" {
		goarch = "x86"
	} else if runtime.GOARCH == "arm64" {
		goarch = "arm64"
	}
	if runtime.GOOS == "darwin" {
		goos = "mac"
	} else if runtime.GOOS == "windows" {
		goos = "win"
	}
	return SysInfo{goarch, goos}
}
func getMapKeys(m map[string]GithubRelease) []string {
	keys := make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	return keys
}
func pathJoin(baseURL, subPath string) string {
	if baseURL == "" {
		return subPath
	}
	return strings.TrimSuffix(baseURL, "/") + "/" + subPath
}
func restartApp(a fyne.App) {
	ex, err := os.Executable()
	if err != nil {
		logger.Errorln(err)
	}
	exeName := filepath.Base(ex)
	parentPath := filepath.Dir(ex)
	updaterPath := filepath.Join(parentPath, exeName)
	cmd := exec.Command("cmd.exe", "/C", updaterPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	a.Quit()
	_ = cmd.Start()
}
func clearOldUpdater() {
	ex, err := os.Executable()
	if err != nil {
		logger.Errorln(err)
	}
	exeName := filepath.Base(ex)
	parentPath := filepath.Dir(ex)
	_ = os.Remove(filepath.Join(parentPath, exeName+"_old"))
}
func handlerErr(err error, message string, win fyne.Window) {
	if err != nil {
		alertInfo(message, win)
	}
}

func GetFileVersion(path string) (string, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	// Get the size of the version info
	size, _, _ := procGetFileVersionInfoSize.Call(uintptr(unsafe.Pointer(p)), 0)
	if size == 0 {
		return "", fmt.Errorf("failed to get version info size")
	}

	// Allocate a buffer to hold the version info
	buffer := make([]byte, size)

	// Get the version info
	ret, _, err := procGetFileVersionInfo.Call(
		uintptr(unsafe.Pointer(p)),
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	if ret == 0 {
		return "", fmt.Errorf("failed to get version info: %v", err)
	}

	// Query the VarFileInfo to find the language and code page
	var block unsafe.Pointer
	var blockLen uint32
	ret, _, err = procVerQueryValue.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("\\VarFileInfo\\Translation"))),
		uintptr(unsafe.Pointer(&block)),
		uintptr(unsafe.Pointer(&blockLen)),
	)
	if ret == 0 {
		return "", fmt.Errorf("failed to query translation value: %v", err)
	}

	translations := (*[1 << 20][2]uint16)(block)[:blockLen/4]
	if len(translations) == 0 {
		return "", fmt.Errorf("no translations found")
	}

	// Use the first translation found
	langCodePage := fmt.Sprintf("%04x%04x", translations[0][0], translations[0][1])
	subBlock := fmt.Sprintf("\\StringFileInfo\\%s\\FileVersion", langCodePage)

	// Query the version value
	ret, _, err = procVerQueryValue.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(subBlock))),
		uintptr(unsafe.Pointer(&block)),
		uintptr(unsafe.Pointer(&blockLen)),
	)
	if ret == 0 {
		return "", fmt.Errorf("failed to query version value: %v", err)
	}

	version := syscall.UTF16ToString((*[1 << 20]uint16)(block)[:blockLen])
	return version, nil
}
func GetVersion(sd *SettingsData, fileName string) string {
	exePath := filepath.Join(getString(sd.installPath), fileName)
	if fileExist(exePath) {
		ver, err := GetFileVersion(exePath)
		if err == nil {
			return ver
		} else {
			logger.Errorln(ver)
		}
	}
	return "-"
}
