package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/robfig/cron/v3"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

var cronManager = cron.New(cron.WithSeconds())

func chromeAutoUpdate(a fyne.App, win fyne.Window, data *SettingsData) {
	if desk, ok := a.(desktop.App); ok {
		addUpdateCron(data)
		updateMenu := fyne.NewMenuItem(LoadString("SystemTrayAutoUpdateMenu"), func() {
			_ = data.autoUpdate.Set(!getBool(data.autoUpdate))
		})
		updateMenu.Checked = getBool(data.autoUpdate)
		if getBool(data.autoUpdate) {
			cronManager.Start()
		} else {
			cronManager.Stop()
		}
		m := fyne.NewMenu("",
			updateMenu,
			fyne.NewMenuItem(LoadString("SystemTrayShowMenu"), func() {
				win.Show()
			}),
			fyne.NewMenuItem(LoadString("SystemTrayHideMenu"), func() {
				win.Hide()
			}),
		)
		data.autoUpdate.AddListener(binding.NewDataListener(func() {
			updateMenu.Checked = getBool(data.autoUpdate)
			if getBool(data.autoUpdate) {
				cronManager.Start()
			} else {
				cronManager.Stop()
			}
			m.Refresh()
		}))
		desk.SetSystemTrayMenu(m)
	}
}

var runFlag = 0

func addUpdateCron(data *SettingsData) {
	spec := "* * */1 * * *" // 每隔5s执行一次，cron格式（秒，分，时，天，月，周）
	_, _ = cronManager.AddFunc(spec, func() {
		chromeInUse := isProcessExist("chrome.exe")
		if runFlag == 1 || chromeInUse {
			return
		}
		if getString(data.oldVer) != "-" {
			runFlag = 1
			sysInfo := getInfo()
			chromeInfo := getLocalChromeInfo(getVk(data.branch, sysInfo))
			_ = data.curVer.Set(chromeInfo.Version)
			_ = data.fileSize.Set(formatFileSize(chromeInfo.Size))
			_ = data.urlList.Set(chromeInfo.Urls)
			_ = data.SHA1.Set(chromeInfo.Sha1)
			_ = data.SHA256.Set(chromeInfo.Sha256)
			_ = data.downBtnStatus.Set(false)
			ov, _ := data.oldVer.Get()
			cv, _ := data.curVer.Get()
			if cv != ov {
				autoInstall(data)
			}
			runFlag = 0
			downloadBtn.SetText(LoadString("InstallBtnLabel"))
		}
	})
}
func timeCost(start time.Time) {
	tc := time.Since(start)
	fmt.Printf("time cost = %v\n", tc)
}

func autoInstall(data *SettingsData) {
	url := getDownloadUrl(data.urlList, data.urlKey)
	parentPath, _ := data.installPath.Get()
	fileName := getFileName(url)
	fileName = filepath.Join(parentPath, fileName)
	var sha1 string
	if fileExist(fileName) {
		sha1 = sumFileSHA1(fileName)
		if v, _ := data.SHA1.Get(); v != sha1 {
			sha1 = downloadChrome(url, fileName)
		}
	} else {
		sha1 = downloadChrome(url, fileName)
	}
	if v, _ := data.SHA1.Get(); v == sha1 {
		defer timeCost(time.Now())
		UnCompress7z(fileName, parentPath)
		UnCompress7z(path.Join(parentPath, "chrome.7z"), parentPath)
		p := path.Join(parentPath, "Chrome-bin")
		targetDir := filepath.Dir(p)
		files, _ := os.ReadDir(p)
		chromeInUse := isProcessExist("chrome.exe")
		if !chromeInUse {
			for _, f := range files {
				_ = os.Rename(filepath.Join(p, f.Name()), path.Join(targetDir, f.Name()))
			}
			//清理文件
			_ = os.Remove(p)
			_ = os.Remove(filepath.Join(parentPath, "chrome.7z"))
			if !getBool(data.remainInstallFileSettings) {
				_ = os.Remove(fileName)
			}
			if !getBool(data.remainHistoryFileSettings) {
				_ = os.RemoveAll(filepath.Join(parentPath, getString(data.oldVer)))
			}
			_ = data.oldVer.Set(getString(data.curVer))
		} else {
			_ = os.Remove(p)
			_ = os.Remove(filepath.Join(parentPath, "chrome.7z"))
		}
	}
}
func downloadChrome(url, fileName string) string {
	fileSize, _ := getFileSize(url)
	var wg = &sync.WaitGroup{}
	autoDownloadProgress := widget.NewProgressBar()
	autoDownloadProgress.SetValue(0)
	autoDownloadProgress.TextFormatter = func() string {
		percentageStr := fmt.Sprintf("%.1f%%", autoDownloadProgress.Value*100.0/0.9)
		downloadBtn.SetText(LoadString("AutoUpdateProgress") + percentageStr)
		return ""
	}
	GoroutineDownload(nil, url, fileName, 4, 1*1024*1024, 1000, fileSize, autoDownloadProgress, wg)
	downloadedBytes = 0
	sha1 := sumFileSHA1(fileName)
	return sha1
}
