package main

import (
	"flag"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"runtime/debug"
)

func main() {
	var level string
	flag.StringVar(&level, "debug", "0", "调试级别")
	flag.Parse()
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("%s", debug.Stack())
			logger.Errorf("This is error message: %v", r)
		}
	}()
	ap := app.New()
	//icon, _ := fyne.LoadResourceFromPath("./assets/img/chrome.ico")
	ap.SetIcon(resourceAssetsImgChromePng)
	//t.SetFonts("./assets/font/MiSans-Regular.ttf", "")
	InitLogger(level)
	//初始化绑定数据
	data := initData()
	logger.Debug("Init data success:")
	initBundle(*data)
	logger.Debug("Set lang success.")
	ap.Settings().SetTheme(&MyTheme{data.themeSettings, data.langSettings})
	logger.Debug("Set theme success.")
	meta := ap.Metadata()
	win := ap.NewWindow(LoadString("TitleLabel") + " v" + meta.Version + " by Libs")
	chromeAutoUpdate(ap, win, data)
	tabs := container.NewAppTabs(
		container.NewTabItem(LoadString("TabMainLabel"), baseScreen(win, data)),
		container.NewTabItem("Chrome++", chromePlusScreen(win, data)),
		container.NewTabItem(LoadString("TabSettingLabel"), settingsScreen(ap, win, data)),
	)
	tabs.Refresh()
	tabs.OnSelected = func(t *container.TabItem) {
		fyne.CurrentApp().Settings().SetTheme(fyne.CurrentApp().Settings().Theme())
	}
	win.SetContent(
		tabs,
	)
	//win.SetMainMenu(makeMenu(ap, win))
	win.CenterOnScreen()
	win.Resize(fyne.NewSize(500, 400))
	//win.SetFixedSize(true)
	win.SetOnClosed(func() {
		//保存配置数据
		clearOldUpdater()
		err := saveConfig(data)
		handlerErr(err, "配置保存失败，请检查目录权限", win)
	})
	ap.Lifecycle().SetOnStopped(func() {
		clearOldUpdater()
		err := saveConfig(data)
		handlerErr(err, "配置保存失败，请检查目录权限", win)
	})
	win.SetCloseIntercept(func() {
		if !getBool(data.autoUpdate) {
			ap.Quit()
		} else {
			win.Hide()
		}
	})
	win.ShowAndRun()
}

func initBundle(data SettingsData) {
	lang := getString(data.langSettings)
	if lang == "System" || lang == "" {
		DelayInitializeLocale()
	} else {
		SetLocale(lang)
	}
}
