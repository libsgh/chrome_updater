package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log"
	"net/http"
	"runtime"
	"strings"
)

func settingsScreen(a fyne.App, win fyne.Window, data *SettingsData) fyne.CanvasObject {
	installFileConfig := widget.NewCheckWithData(LoadString("BaseRemainInstallFiles"), data.remainInstallFileSettings)
	historyVersionConfig := widget.NewCheckWithData(LoadString("BaseRemainHistoryFiles"), data.remainHistoryFileSettings)
	ghProxyEntry := widget.NewEntryWithData(data.ghProxy)
	ghProxyEntry.PlaceHolder = LoadString("BaseGhProxy")
	themeRadio := widget.NewRadioGroup([]string{"System", "Light", "Dark"}, func(value string) {
		data.themeSettings.Set(value)
		fyne.CurrentApp().Settings().SetTheme(&MyTheme{data.themeSettings, data.langSettings})
	})
	langRadio := widget.NewRadioGroup([]string{
		"System",
		"en-US",
		"zh-CN"}, func(value string) {
		data.langSettings.Set(value)
		win.Content().Refresh()
	})
	if getString(data.langSettings) == "" {
		data.langSettings.Set(LoadString("SystemOption"))
	}
	langRadio.Selected = getString(data.langSettings)
	langRadio.Horizontal = true
	if getString(data.themeSettings) == "" {
		data.themeSettings.Set(LoadString("SystemOption"))
	}
	themeRadio.Selected = getString(data.themeSettings)
	themeRadio.Horizontal = true
	hasNew, url := chromeUpdaterNew(getString(data.ghProxy))
	newBtn := widget.NewButton(LoadString("UpdaterCheckBtnLabel"), func() {
		_ = a.OpenURL(parseURL(url))
	})
	if hasNew {
		newBtn.Show()
	} else {
		newBtn.Hide()
	}
	return container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle(LoadString("BaseSettingLabel"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, installFileConfig, historyVersionConfig),
		ghProxyEntry,
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
func chromeUpdaterNew(proxy string) (bool, string) {
	response, err := http.Get(pathJoin(proxy, "https://raw.githubusercontent.com/libsgh/ghapi-json-generator/output/v2/repos/libsgh/chrome_updater/releases%3Fper_page%3D10/data.json"))
	if err != nil {
		log.Println(err)
		return false, ""
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	var githubReleases []GithubRelease
	jsoniter.UnmarshalFromString(string(data), &githubReleases)
	if err != nil {
		log.Println(err)
		return false, ""
	}
	if len(githubReleases) == 0 {
		return false, ""
	}
	ver := "v" + fyne.CurrentApp().Metadata().Version
	lastedVer := githubReleases[0].TagName
	for _, asset := range githubReleases[0].Assets {
		if strings.Contains(asset.BrowserDownloadURL, fmt.Sprintf("chrome_updater-windows-%s.zip", runtime.GOARCH)) {
			return ver != lastedVer, pathJoin(proxy, asset.BrowserDownloadURL)
		}
	}
	return ver != lastedVer, pathJoin(proxy, githubReleases[0].Assets[0].BrowserDownloadURL)
}
