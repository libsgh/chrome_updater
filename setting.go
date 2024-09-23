package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

func settingsScreen(a fyne.App, win fyne.Window, data *SettingsData) fyne.CanvasObject {
	installFileConfig := widget.NewCheckWithData(LoadString("BaseRemainInstallFiles"), data.remainInstallFileSettings)
	historyVersionConfig := widget.NewCheckWithData(LoadString("BaseRemainHistoryFiles"), data.remainHistoryFileSettings)
	proxyType := widget.NewSelect([]string{"GH-PROXY", "HTTP(S)", "SOCKS5"}, func(value string) {
		_ = data.proxyType.Set(value)
	})
	proxyTypeVal := getString(data.proxyType)
	if proxyTypeVal == "" {
		proxyType.Selected = "GH-PROXY"
		_ = data.proxyType.Set("GH-PROXY")
	} else {
		proxyType.Selected = getString(data.proxyType)
	}
	ghProxyEntry := widget.NewEntryWithData(data.ghProxy)
	ghProxyEntry.PlaceHolder = LoadString("BaseGhProxy")
	themeRadio := widget.NewRadioGroup([]string{"System", "Light", "Dark"}, func(value string) {
		_ = data.themeSettings.Set(value)
		fyne.CurrentApp().Settings().SetTheme(&MyTheme{data.themeSettings, data.langSettings})
	})
	langRadio := widget.NewRadioGroup([]string{
		"System",
		"en-US",
		"zh-CN"}, func(value string) {
		if value != "" {
			_ = data.langSettings.Set(value)
			restartApp(a)
		}
	})
	if getString(data.langSettings) == "" {
		_ = data.langSettings.Set(LoadString("SystemOption"))
	}
	langRadio.Selected = getString(data.langSettings)
	langRadio.Horizontal = true
	if getString(data.themeSettings) == "" {
		_ = data.themeSettings.Set(LoadString("SystemOption"))
	}
	themeRadio.Selected = getString(data.themeSettings)
	themeRadio.Horizontal = true
	updateUrl := binding.NewString()
	updateBtnText := binding.NewString()
	updateBtnText.Set(LoadString("UpdaterCheckBtnLabel"))
	newBtn := widget.NewButton(getString(updateBtnText), func() {
		//_ = a.OpenURL(parseURL(url))
		UpdateSelf(a, data, getString(updateUrl), updateBtnText)
	})
	go chromeUpdaterNew(data, getString(data.ghProxy), updateUrl, newBtn)
	updateBtnText.AddListener(binding.NewDataListener(func() {
		newBtn.SetText(getString(updateBtnText))
	}))

	return container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle(LoadString("BaseSettingLabel"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(3, installFileConfig, historyVersionConfig),
		container.NewBorder(nil, nil, proxyType, nil, ghProxyEntry),
		widget.NewSeparator(),
		widget.NewLabelWithStyle(LoadString("ThemeLabel"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(themeRadio),
		widget.NewSeparator(),
		widget.NewLabelWithStyle(LoadString("LangLabel"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(langRadio),
		widget.NewSeparator(),
		widget.NewLabelWithStyle(LoadString("AboutLabel"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(
			widget.NewLabel(LoadString("VersionLabel")+": v"+fyne.CurrentApp().Metadata().Version),
			newBtn,
			widget.NewButton(LoadString("IssuesLabel"), func() {
				_ = a.OpenURL(parseURL("https://github.com/libsgh/chrome_updater/issues"))
			}),
		),
		container.NewHBox(
			widget.NewHyperlink(LoadString("OfflinePkgLabel"), parseURL("https://chrome.noki.eu.org")),
			widget.NewLabel("-"),
			widget.NewHyperlink("GitHub", parseURL("https://github.com/libsgh/chrome_updater")),
			widget.NewLabel("-"),
			widget.NewHyperlink("LICENSE", parseURL("https://github.com/libsgh/chrome_updater/blob/main/LICENSE")),
		),
	))
}

func UpdateSelf(a fyne.App, sd *SettingsData, url string, btnText binding.String) {
	_ = btnText.Set("0.0%")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeName := filepath.Base(ex)
	parentPath := filepath.Dir(ex)
	fileName := getFileName(url)
	fileName = filepath.Join(parentPath, fileName)
	fileSize, _ := getFileSize(url)
	var wg = &sync.WaitGroup{}
	updaterDownloadProgress := widget.NewProgressBar()
	updaterDownloadProgress.TextFormatter = func() string {
		percentageStr := fmt.Sprintf("%.1f%%", updaterDownloadProgress.Value*100.0/0.9)
		_ = btnText.Set(percentageStr)
		return ""
	}
	GoroutineDownload(sd, url, fileName, 4, 50*1024, 500, fileSize, updaterDownloadProgress, wg)
	downloadedBytes = 0
	updaterPath := filepath.Join(parentPath, exeName)
	if fileExist(updaterPath) {
		os.Rename(updaterPath, filepath.Join(parentPath, exeName+"_old"))
	}
	unzip(fileName, exeName)
	_ = os.Remove(fileName)
	cmd := exec.Command("cmd.exe", "/C", updaterPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	a.Quit()
	_ = cmd.Start()
}
func chromeUpdaterNew(sd *SettingsData, proxy string, updateUrl binding.String, newBtn *widget.Button) {
	apiUrl := "https://raw.githubusercontent.com/libsgh/ghapi-json-generator/output/v2/repos/libsgh/chrome_updater/releases%3Fper_page%3D10/data.json"
	client, reqUrl := setProxy(sd, apiUrl)
	response, err := client.Get(reqUrl)
	if err != nil {
		log.Println(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	var githubReleases []GithubRelease
	jsoniter.UnmarshalFromString(string(data), &githubReleases)
	if err != nil {
		log.Println(err)
	}
	if len(githubReleases) != 0 {
		ver := "v" + fyne.CurrentApp().Metadata().Version
		lastedVer := githubReleases[0].TagName
		for _, asset := range githubReleases[0].Assets {
			if strings.Contains(asset.BrowserDownloadURL, fmt.Sprintf("chrome_updater-windows-%s.zip", runtime.GOARCH)) {
				updateUrl.Set(asset.BrowserDownloadURL)
			}
		}
		hasNew := ver != lastedVer
		if hasNew {
			newBtn.Show()
		} else {
			newBtn.Hide()
		}
	}
}
