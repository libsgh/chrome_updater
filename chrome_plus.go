package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

func chromePlusScreen(win fyne.Window, data *SettingsData) fyne.CanvasObject {
	var githubReleaseMap map[string]GithubRelease
	var versionList []string
	chromePlusRadio := widget.NewRadioGroup([]string{
		"Bush2021"}, func(value string) {
		data.chromePlus.Set(value)
	})
	versionSelect := widget.NewSelect([]string{}, func(ver string) {
		setPlusVer(data, ver, githubReleaseMap)
	})
	versionSelect.PlaceHolder = LoadString("VersionSelectPlaceHolder")
	versionSelect.Disable()
	chromePlusRadio.Selected = getString(data.chromePlus)
	chromePlusRadio.Disable()
	downBtn := widget.NewButtonWithIcon(LoadString("InstallBtnLabel"), theme.DownloadIcon(), func() {
		ov, _ := data.oldPlusVer.Get()
		cv, _ := data.curPlusVer.Get()
		if cv == ov && fileExist(path.Join(getString(data.installPath), "version.dll")) {
			alertInfo(LoadString("NoNeedUpdateMsg"), win)
		} else {
			chromeInUse := isProcessExist("chrome.exe")
			if chromeInUse {
				alertInfo(LoadString("ChromeRunningMsg"), win)
			} else {
				data.plusBtnStatus.Set(true)
				installPlus(data, win)
			}
		}
	})
	checkBtn := widget.NewButtonWithIcon(LoadString("CheckBtnLabel"), theme.SearchIcon(), func() {
		githubReleaseMap, versionList = getChromePlusInfo(getString(data.ghProxy))
		if githubReleaseMap != nil && len(versionList) > 0 {
			versionSelect.SetOptions(versionList)
			versionSelect.SetSelected(versionList[0])
			setPlusVer(data, versionList[0], githubReleaseMap)
			versionSelect.Enable()
			downBtn.Enable()
		} else {
			alertInfo(LoadString("CheckErrMsg"), win)
		}
	})
	data.plusBtnStatus.AddListener(binding.NewDataListener(func() {
		if getBool(data.plusBtnStatus) {
			downBtn.Disable()
		} else {
			downBtn.Enable()
		}
	}))
	curVerLabel := widget.NewLabelWithData(data.curPlusVer)
	curVerLabel.TextStyle.Bold = true
	form := widget.NewForm(
		&widget.FormItem{Text: LoadString("NowVerLabel"), Widget: widget.NewLabelWithData(data.oldPlusVer)},
		&widget.FormItem{Text: LoadString("LatestVerLabel"), Widget: versionSelect},
		&widget.FormItem{Text: LoadString("BranchLabel"), Widget: chromePlusRadio},
	)
	rich := widget.NewRichTextFromMarkdown(LoadString("MarkdownMsg"))
	rich.Wrapping = fyne.TextWrapWord
	infoCard := widget.NewCard("", "", rich)
	plusDownloadProgress = widget.NewProgressBar()
	plusDownloadProgress.TextFormatter = func() string {
		if plusDownloadProgress.Max*0.9 == plusDownloadProgress.Value {
			return fmt.Sprintf(LoadString("PlusDownloadedMsg"))
		} else if plusDownloadProgress.Max == plusDownloadProgress.Value {
			return "安装完成"
		} else if plusDownloadProgress.Value == -1 {
			return "下载失败，请稍后重试"
		}
		return fmt.Sprintf(LoadString("PlusDownloadingMsg"))
	}
	data.plusProcessStatus.AddListener(binding.NewDataListener(func() {
		if getBool(data.plusProcessStatus) {
			plusDownloadProgress.Show()
		} else {
			plusDownloadProgress.Hide()
		}
	}))
	return container.New(&buttonLayout{}, container.NewVBox(form,
		infoCard,
	), container.NewVBox(plusDownloadProgress, container.NewGridWithColumns(2, checkBtn, downBtn)))
}

func setPlusVer(data *SettingsData, ver string, releaseMap map[string]GithubRelease) {
	plusInfo := releaseMap[ver]
	data.curPlusVer.Set(plusInfo.TagName)
	data.plusDownloadUrl.Set(pathJoin(getString(data.ghProxy), plusInfo.Assets[0].BrowserDownloadURL))
}

var (
	plusDownloadProgress *widget.ProgressBar
)

func installPlus(data *SettingsData, win fyne.Window) {
	url := getString(data.plusDownloadUrl)
	plusDownloadProgress.SetValue(0)
	data.plusProcessStatus.Set(true)
	sysInfo := getInfo()
	parentPath, _ := data.installPath.Get()
	downloadProgress.SetValue(0)
	fileName := getFileName(url)
	fileName = filepath.Join(parentPath, fileName)
	fileSize, _ := getFileSize(url)
	var wg = &sync.WaitGroup{}
	GoroutineDownload(url, fileName, 4, 5*1024*1024, 30, fileSize, plusDownloadProgress, wg)
	UnCompress7z(fileName, parentPath)
	os.Rename(filepath.Join(parentPath, sysInfo.goarch, "App", "version.dll"), path.Join(parentPath, "version.dll"))
	os.Rename(filepath.Join(parentPath, sysInfo.goarch, "App", "chrome++.ini"), path.Join(parentPath, "chrome++.ini"))
	//clean tmp dir
	os.Remove(fileName)
	os.RemoveAll(filepath.Join(parentPath, "x86"))
	os.RemoveAll(filepath.Join(parentPath, "x64"))
	plusDownloadProgress.SetValue(1)
	defer data.oldPlusVer.Set(getString(data.curPlusVer))
	defer data.plusBtnStatus.Set(false)
	alertInfo(LoadString("InstalledMsg"), win)
}

func getChromePlusInfo(ghProxy string) (map[string]GithubRelease, []string) {
	response, err := http.Get(pathJoin(ghProxy, "https://raw.githubusercontent.com/libsgh/ghapi-json-generator/output/v2/repos/Bush2021/chrome_plus/releases%3Fper_page%3D10/data.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	var githubReleases []GithubRelease
	jsoniter.UnmarshalFromString(string(data), &githubReleases)
	if err != nil {
		fmt.Println(err)
	}
	result := make(map[string]GithubRelease)
	versionList := make([]string, 0)

	for _, item := range githubReleases {
		result[item.TagName] = item
		versionList = append(versionList, item.TagName)
	}
	return result, versionList
}
