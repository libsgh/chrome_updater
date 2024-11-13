package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	jsoniter "github.com/json-iterator/go"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func baseScreen(win fyne.Window, data *SettingsData) fyne.CanvasObject {
	// 获取系统信息
	sysInfo := getInfo()
	installPathHandle(data)
	folderEntry := widget.NewEntryWithData(data.installPath)
	folderEntry.OnChanged = func(path string) {
		installPathHandle(data)
	}
	showFolderPicker := func() {
		onChosen := func(f fyne.ListableURI, err error) {
			if err != nil {
				fmt.Println(err)
				return
			}
			if f == nil {
				return
			}
			_ = data.installPath.Set(f.Path())
		}
		dialog.ShowFolderOpen(onChosen, win)
	}
	folderBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), showFolderPicker)
	data.folderEntryStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.folderEntryStatus.Get(); b {
			folderEntry.Disable()
			folderBtn.Disable()
		} else {
			folderEntry.Enable()
			folderBtn.Enable()
		}
	}))
	checkBtn := widget.NewButtonWithIcon(LoadString("CheckBtnLabel"), theme.SearchIcon(), func() {
		syncChromeInfo(data, sysInfo)
	})
	downloadBtn = widget.NewButtonWithIcon(LoadString("InstallBtnLabel"), theme.DownloadIcon(), func() {
		ov, _ := data.oldVer.Get()
		cv, _ := data.curVer.Get()
		if cv == ov {
			alertInfo(LoadString("NoNeedUpdateMsg"), win)
		} else {
			chromeInUse := isProcessExist("chrome.exe")
			if chromeInUse {
				alertInfo(LoadString("ChromeRunningMsg"), win)
			} else {
				if runFlag == 1 {
					alertInfo(LoadString("ChromeUpdateRunningMsg"), win)
				} else {
					runFlag = 1
					if getString(data.oldVer) == "-" {
						alertConfirm(LoadString("FirstInstallMsg"), func(b bool) {
							execDownAndUnzip(data, downloadProgress, 0)
						}, win)
					} else {
						execDownAndUnzip(data, downloadProgress, 1)
					}
					runFlag = 0
				}

			}
		}
	})
	data.downBtnStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.downBtnStatus.Get(); b {
			downloadBtn.Disable()
		} else {
			downloadBtn.Enable()
		}
	}))
	data.checkBtnStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.checkBtnStatus.Get(); b {
			checkBtn.Disable()
		} else {
			checkBtn.Enable()
		}
	}))
	versionRadio := widget.NewRadioGroup(LoadStringList("StableVerOption", "BetaVerOption", "DevVerOption", "CanaryVerOption"), func(value string) {
		if value == LoadString("StableVerOption") {
			data.branch.Set("stable")
		} else if value == LoadString("BetaVerOption") {
			data.branch.Set("beta")
		} else if value == LoadString("DevVerOption") {
			data.branch.Set("dev")
		} else if value == LoadString("CanaryVerOption") {
			data.branch.Set("canary")
		} else {
			data.branch.Set("stable")
		}
	})
	versionRadio.Horizontal = true
	if getString(data.branch) == "stable" {
		versionRadio.Selected = LoadString("StableVerOption")
	} else if getString(data.branch) == "beta" {
		versionRadio.Selected = LoadString("BetaVerOption")
	} else if getString(data.branch) == "dev" {
		versionRadio.Selected = LoadString("DevVerOption")
	} else if getString(data.branch) == "canary" {
		versionRadio.Selected = LoadString("CanaryVerOption")
	} else {
		versionRadio.Selected = LoadString("StableVerOption")
	}
	urlsRadio := widget.NewRadioGroup([]string{
		"edgedl.me.gvt1",
		"dl.google.com",
		"www.google.com"}, func(value string) {
		if value == "" {
			data.urlKey.Set("edgedl.me.gvt1")
		} else {
			data.urlKey.Set(value)
		}
	})
	urlsRadio.Selected = getString(data.urlKey)
	urlsRadio.Horizontal = true
	buttons := container.NewHBox(folderBtn)
	bar := container.NewBorder(nil, nil, buttons, nil, folderEntry)
	curVerLabel := widget.NewLabelWithData(data.curVer)
	curVerLabel.TextStyle.Bold = true
	oldVer := GetVersion(data, "chrome.exe")
	logger.Info("chrome version:", oldVer)
	_ = data.oldVer.Set(oldVer)
	form := widget.NewForm(
		&widget.FormItem{Text: LoadString("InstallLabel"), Widget: bar},
		&widget.FormItem{Text: LoadString("BranchLabel"), Widget: versionRadio},
		&widget.FormItem{Text: LoadString("NowVerLabel"), Widget: widget.NewLabelWithData(data.oldVer)},
		&widget.FormItem{Text: LoadString("LatestVerLabel"), Widget: curVerLabel},
		&widget.FormItem{Text: LoadString("FileSizeLabel"), Widget: widget.NewLabelWithData(data.fileSize)},
		&widget.FormItem{Text: "SHA1", Widget: widget.NewLabelWithData(data.SHA1)},
		&widget.FormItem{Text: "SHA256", Widget: widget.NewLabelWithData(data.SHA256)},
		&widget.FormItem{Text: LoadString("DownLoadChannelLabel"), Widget: urlsRadio},
	)
	downloadProgress = widget.NewProgressBar()
	downloadProgress.TextFormatter = func() string {
		fs, _ := data.fileSize.Get()
		if downloadProgress.Max*0.9 == downloadProgress.Value {
			return fmt.Sprintf(LoadString("DownLoadedProcessMsg"), fs)
		} else if downloadProgress.Max == downloadProgress.Value {
			return LoadString("InstalledMsg")
		} else if downloadProgress.Value == -1 {
			return LoadString("DownloadFailedMsg")
		} else if downloadProgress.Value == 0.95 {
			return LoadString("Download95Msg")
		}
		fsFloatStr := strings.Split(fs, " ")[0]
		fsFloat, err := strconv.ParseFloat(fsFloatStr, 64)
		if err != nil {
			return LoadString("DownloadNotStartedMsg")
		}
		return fmt.Sprintf(LoadString("DownloadingMsg"), fsFloat*downloadProgress.Value, fs)
	}
	data.processStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.processStatus.Get(); b {
			downloadProgress.Show()
		} else {
			downloadProgress.Hide()
		}
	}))
	if !getBool(data.autoUpdate) {
		go syncChromeInfo(data, sysInfo)
	}
	logger.Debug("Base tab load success.")
	return container.New(&buttonLayout{}, form, container.NewVBox(downloadProgress, container.NewGridWithColumns(2, checkBtn, downloadBtn)))
}
func syncChromeInfo(data *SettingsData, sysInfo SysInfo) {
	chromeInfo := getLocalChromeInfo(getVk(data.branch, sysInfo))
	data.curVer.Set(chromeInfo.Version)
	data.fileSize.Set(formatFileSize(chromeInfo.Size))
	data.urlList.Set(chromeInfo.Urls)
	data.SHA1.Set(chromeInfo.Sha1)
	data.SHA256.Set(chromeInfo.Sha256)
	data.downBtnStatus.Set(false)
}
func execDownAndUnzip(data *SettingsData, downloadProgress *widget.ProgressBar, installType int) {
	data.checkBtnStatus.Set(true)
	data.folderEntryStatus.Set(true)
	data.processStatus.Set(true)
	if installType == 0 {
		initInstallDirs(data)
	}
	url := getDownloadUrl(data.urlList, data.urlKey)
	parentPath, _ := data.installPath.Get()
	downloadProgress.SetValue(0)
	fileName := getFileName(url)
	fileName = filepath.Join(parentPath, fileName)
	fileSize, _ := getFileSize(url)
	var wg = &sync.WaitGroup{}
	GoroutineDownload(nil, url, fileName, 4, 1*1024*1024, 1000, fileSize, downloadProgress, wg)
	downloadedBytes = 0
	sha1 := sumFileSHA1(fileName)
	if v, _ := data.SHA1.Get(); v != sha1 {
		downloadProgress.SetValue(-1)
	} else {
		downloadProgress.SetValue(0.95)
		//解压覆盖
		//UUnCompress7z(fileName, parentPath)
		UnCompressBy7Zip(fileName, parentPath)
		UnCompress7z(filepath.Join(parentPath, "chrome.7z"), parentPath)
		p := filepath.Join(parentPath, "Chrome-bin")
		targetDir := filepath.Dir(p)
		files, _ := os.ReadDir(p)
		for _, f := range files {
			_ = os.Rename(filepath.Join(p, f.Name()), filepath.Join(targetDir, f.Name()))
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
		downloadProgress.SetValue(1)
		data.oldVer.Set(getString(data.curVer))
		defer data.checkBtnStatus.Set(false)
		defer data.folderEntryStatus.Set(false)
	}

}

// 初始化安装目录
func initInstallDirs(data *SettingsData) {
	//创建App、Cache、Data目录
	os.Mkdir(filepath.Join(getString(data.installPath), "App"), os.ModePerm)
	os.Mkdir(filepath.Join(getString(data.installPath), "Cache"), os.ModePerm)
	os.Mkdir(filepath.Join(getString(data.installPath), "Data"), os.ModePerm)
	//更改安装目录
	data.installPath.Set(filepath.Join(getString(data.installPath), "App"))
}

var (
	downloadProgress *widget.ProgressBar
	downloadBtn      *widget.Button
)

// 获取下载地址
func getDownloadUrl(list binding.StringList, urlKey binding.String) string {
	urls, _ := list.Get()
	uk, _ := urlKey.Get()
	for _, url := range urls {
		if strings.HasPrefix(url, "https://"+uk) {
			return url
		}
	}
	return urls[0]
}

// 获取版本分支KEY
func getVk(branch binding.String, sysInfo SysInfo) string {
	b, _ := branch.Get()
	return sysInfo.goos + "_" + b + "_" + sysInfo.goarch
}

// 获取Chrome版本信息
func getChromeInfo(key string) ChromeInfo {
	// 发送 HTTP 请求获取 JSON 数据
	response, err := http.Get("https://chrome.noki.eu.org/api/c/info")
	if err != nil {
		logger.Panic(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	chromeInfo := ChromeInfo{}
	// 解码 JSON 数据到结构体
	infoStr := jsoniter.Get(data, key).ToString()
	jsoniter.UnmarshalFromString(infoStr, &chromeInfo)
	if err != nil {
		logger.Panic(err)
	}
	return chromeInfo
}

// 处理Chrome安装路径
func installPathHandle(data *SettingsData) {
	//读取当前程序所在目录
	p, _ := data.installPath.Get()
	dir, err := os.Getwd()
	if isValidPath(p) {
		dir = p
	} else {
		data.installPath.Set(dir)
	}
	if err != nil {
		logger.Panic(err)
	}
	// 打开当前目录
	dirHandle, err := os.Open(dir)
	if err != nil {
		logger.Panic(err)
	}
	defer dirHandle.Close()
	fileInfos, err := dirHandle.Readdir(-1)
	if err != nil {
		logger.Panic(err)
	}
	result := false
	v := ""
	for _, fileInfo := range fileInfos {
		name := fileInfo.Name()
		if name == "chrome.exe" {
			result = true
		}
		if fileInfo.IsDir() && isNumeric(strings.ReplaceAll(name, ".", "")) {
			v = fileInfo.Name()
		}
	}
	if result {
		data.installPath.Set(dir)
		data.oldVer.Set(v)
	} else {
		data.oldVer.Set("-")
	}
	if getBool(data.downBtnStatus) {
		data.checkBtnStatus.Set(false)
	}
}
