package main

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	jsoniter "github.com/json-iterator/go"
	"os"
	"path/filepath"
)

func getConfigPath() string {
	appDataDir := os.Getenv("APPDATA")
	if appDataDir == "" {
		appDataDir = os.TempDir()
	}
	return filepath.Join(appDataDir, "chrome_updater", "config.json")
}

// 初始化数据
func initData() *SettingsData {
	configFilePath := getConfigPath()
	configFileExist := fileExist(configFilePath)
	var config Config
	sd := createSettings()
	if configFileExist {
		file, err := os.Open(configFilePath)
		if err != nil {
			fmt.Println("无法打开文件:", err)
		}
		decoder := jsoniter.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			fmt.Println("解析 JSON 失败:", err)
		}
		sd.installPath.Set(config.InstallPath)
		sd.branch.Set(config.VersionBranch)
		sd.urlKey.Set(config.DownloadChannel)
		sd.remainInstallFileSettings.Set(config.RemainInstallFile)
		sd.remainHistoryFileSettings.Set(config.RemainHistoryFile)
		sd.oldPlusVer.Set(config.OldPlusVer)
		sd.chromePlus.Set(config.ChromePlus)
		sd.themeSettings.Set(config.Theme)
		sd.langSettings.Set(config.Lang)
		sd.ghProxy.Set(config.GhProxy)
	}
	return sd
}

func saveConfig(data *SettingsData) {
	config := Config{
		InstallPath:       getString(data.installPath),
		VersionBranch:     getString(data.branch),
		DownloadChannel:   getString(data.urlKey),
		RemainInstallFile: getBool(data.remainInstallFileSettings),
		RemainHistoryFile: getBool(data.remainHistoryFileSettings),
		OldPlusVer:        getString(data.oldPlusVer),
		ChromePlus:        getString(data.chromePlus),
		Theme:             getString(data.themeSettings),
		Lang:              getString(data.langSettings),
		GhProxy:           getString(data.ghProxy),
	}
	jsonData, _ := jsoniter.Marshal(config)
	configFilePath := getConfigPath()
	os.Remove(configFilePath)
	configFileExist := fileExist(configFilePath)
	if !configFileExist {
		dir := filepath.Dir(configFilePath)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Println("无法创建目录:", err)
			return
		}
		// 创建文件
		file, err := os.Create(configFilePath)
		if err != nil {
			fmt.Println("无法创建文件:", err)
			return
		}
		defer file.Close()
		_, err = file.Write(jsonData)
		if err != nil {
			fmt.Println("无法写入文件:", err)
			return
		}
	}
}

// 创建配置数据
func createSettings() *SettingsData {
	installPath := binding.NewString()
	branch := binding.NewString()
	oldVer := binding.NewString()
	oldVer.Set("-")
	curVer := binding.NewString()
	curVer.Set("-")
	fileSize := binding.NewString()
	fileSize.Set("-")
	SHA1 := binding.NewString()
	SHA1.Set("-")
	SHA256 := binding.NewString()
	SHA256.Set("-")
	urlList := binding.NewStringList()
	_ = installPath.Set("请配置Chrome安装目录(APP)")
	_ = branch.Set("stable")
	downBtnStatus := binding.NewBool()
	downBtnStatus.Set(true) // 初始下载按钮状态
	checkBtnStatus := binding.NewBool()
	checkBtnStatus.Set(true) // 初始检查按钮状态
	folderEntryStatus := binding.NewBool()
	folderEntryStatus.Set(false) //初始化Chrome安装目录状态
	urlKey := binding.NewString()
	urlKey.Set("edgedl.me.gvt1") //设置默认下载通道
	processStatus := binding.NewBool()
	processStatus.Set(false) //初始化下载安装进度的进度条状态
	remainInstallFileSettings := binding.NewBool()
	remainInstallFileSettings.Set(false) //保留安装文件
	remainHistoryFileSettings := binding.NewBool()
	remainHistoryFileSettings.Set(false) //保留历史文件
	themeSettings := binding.NewString()
	themeSettings.Set(LoadString("SystemOption"))
	langSettings := binding.NewString()
	langSettings.Set(LoadString("SystemOption"))
	oldPlusVer := binding.NewString()
	curPlusVer := binding.NewString()
	curPlusVer.Set("-")
	oldPlusVer.Set("-")
	chromePlus := binding.NewString()
	chromePlus.Set("Bush2021")
	plusDownloadUrl := binding.NewString()
	plusBtnStatus := binding.NewBool()
	plusBtnStatus.Set(true)
	plusProcessStatus := binding.NewBool()
	plusProcessStatus.Set(false)
	ghProxy := binding.NewString()
	ghProxy.Set("")
	return &SettingsData{
		installPath:               installPath,
		oldVer:                    oldVer,
		branch:                    branch,
		curVer:                    curVer,
		fileSize:                  fileSize,
		SHA1:                      SHA1,
		SHA256:                    SHA256,
		urlList:                   urlList,
		downBtnStatus:             downBtnStatus,
		checkBtnStatus:            checkBtnStatus,
		folderEntryStatus:         folderEntryStatus,
		urlKey:                    urlKey,
		processStatus:             processStatus,
		oldPlusVer:                oldPlusVer,
		curPlusVer:                curPlusVer,
		chromePlus:                chromePlus,
		plusDownloadUrl:           plusDownloadUrl,
		plusBtnStatus:             plusBtnStatus,
		plusProcessStatus:         plusProcessStatus,
		remainInstallFileSettings: remainInstallFileSettings,
		remainHistoryFileSettings: remainHistoryFileSettings,
		themeSettings:             themeSettings,
		langSettings:              langSettings,
		ghProxy:                   ghProxy,
	}
}
