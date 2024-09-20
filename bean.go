package main

import (
	"encoding/xml"
	"fyne.io/fyne/v2/data/binding"
	"time"
)

// ChromeInfo chrome信息
type ChromeInfo struct {
	Sha1         string      `json:"sha1"`
	Sha256       string      `json:"sha256"`
	Chromewithgc interface{} `json:"chromewithgc"`
	Version      string      `json:"version"`
	Size         int64       `json:"size"`
	Time         int64       `json:"time"`
	Urls         []string    `json:"urls"`
}

// ChromePlusInfo chrome ++
type ChromePlusInfo struct {
	Version     string `json:"version"`
	DownloadUrl string `json:"downloadurl"`
}

// SysInfo 系统信息
type SysInfo struct {
	goarch, goos string
}

// 配置信息
type SettingsData struct {
	installPath               binding.String     //安装目录
	oldVer                    binding.String     //旧版本号
	branch                    binding.String     //版本分支
	curVer                    binding.String     //最新版本号
	fileSize                  binding.String     //文件大小
	SHA1                      binding.String     //文件SHA1
	SHA256                    binding.String     //文件SHA256
	urlList                   binding.StringList //下载地址
	downBtnStatus             binding.Bool       //下载按钮状态
	checkBtnStatus            binding.Bool       //检查按钮状态
	folderEntryStatus         binding.Bool       //安装目录修改状态
	urlKey                    binding.String     //下载通道
	chromePlus                binding.String     //chrome_plus
	oldPlusVer                binding.String     //已安装chrome_plus版本
	curPlusVer                binding.String     //最新chrome_plus版本
	plusDownloadUrl           binding.String     //最新chrome_plus下载地址
	plusBtnStatus             binding.Bool       //plus下载安装状态
	plusProcessStatus         binding.Bool       //plus下载安装进度的进度条状态
	processStatus             binding.Bool       //下载安装进度的进度条状态
	remainInstallFileSettings binding.Bool       //是否保留安装文件
	remainHistoryFileSettings binding.Bool       //是否保留历史文件
	themeSettings             binding.String     //主题设置
	langSettings              binding.String     //语言设置
	ghProxy                   binding.String     //Github代理
	proxyType                 binding.String     //代理类型
	autoUpdate                binding.Bool       //自动更新
}

// 配置选项
type Config struct {
	InstallPath       string `json:"install_path"`        //安装目录
	VersionBranch     string `json:"version_branch"`      //版本分支
	DownloadChannel   string `json:"download_channel"`    //下载通道
	OldPlusVer        string `json:"old_plus_ver"`        //已安装chrome_plus版本
	ChromePlus        string `json:"chrome_plus"`         //chrome_plus
	RemainInstallFile bool   `json:"remain_install_file"` //是否保留安装文件
	RemainHistoryFile bool   `json:"remain_history_file"` //是否保留历史文件
	Theme             string `json:"theme"`               //主题设置
	Lang              string `json:"lang"`                //语言设置
	GhProxy           string `json:"gh_proxy"`            //Github代理加速
	ProxyType         string `json:"proxy_type"`          //代理类型
	AutoUpdate        bool   `json:"auto_update"`         //自动更新
}

type GithubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		URL                string      `json:"url"`
		ID                 int         `json:"id"`
		NodeID             string      `json:"node_id"`
		Name               string      `json:"name"`
		Label              interface{} `json:"label"`
		ContentType        string      `json:"content_type"`
		State              string      `json:"state"`
		Size               int         `json:"size"`
		DownloadCount      int         `json:"download_count"`
		CreatedAt          time.Time   `json:"created_at"`
		UpdatedAt          time.Time   `json:"updated_at"`
		BrowserDownloadURL string      `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"`
}

type TestText struct {
	Label string
}

// Chrome更新查询
type Response struct {
	XMLName  xml.Name `xml:"response"`
	Text     string   `xml:",chardata"`
	Protocol string   `xml:"protocol,attr"`
	Server   string   `xml:"server,attr"`
	Daystart struct {
		Text           string `xml:",chardata"`
		ElapsedDays    string `xml:"elapsed_days,attr"`
		ElapsedSeconds string `xml:"elapsed_seconds,attr"`
	} `xml:"daystart"`
	App struct {
		Text        string `xml:",chardata"`
		Appid       string `xml:"appid,attr"`
		Cohort      string `xml:"cohort,attr"`
		Cohortname  string `xml:"cohortname,attr"`
		Status      string `xml:"status,attr"`
		Updatecheck struct {
			Text   string `xml:",chardata"`
			Status string `xml:"status,attr"`
			Urls   struct {
				Text string `xml:",chardata"`
				URL  []struct {
					Text     string `xml:",chardata"`
					Codebase string `xml:"codebase,attr"`
				} `xml:"url"`
			} `xml:"urls"`
			Manifest struct {
				Text    string `xml:",chardata"`
				Version string `xml:"version,attr"`
				Actions struct {
					Text   string `xml:",chardata"`
					Action []struct {
						Text      string `xml:",chardata"`
						Arguments string `xml:"arguments,attr"`
						Event     string `xml:"event,attr"`
						Run       string `xml:"run,attr"`
						Version   string `xml:"Version,attr"`
						E         string `xml:"e,attr"`
						Vent      string `xml:"vent,attr"`
						Onsuccess string `xml:"onsuccess,attr"`
					} `xml:"action"`
				} `xml:"actions"`
				Packages struct {
					Text    string `xml:",chardata"`
					Package struct {
						Text       string `xml:",chardata"`
						Fp         string `xml:"fp,attr"`
						Hash       string `xml:"hash,attr"`
						HashSha256 string `xml:"hash_sha256,attr"`
						Name       string `xml:"name,attr"`
						Required   string `xml:"required,attr"`
						Size       string `xml:"size,attr"`
					} `xml:"package"`
				} `xml:"packages"`
			} `xml:"manifest"`
		} `xml:"updatecheck"`
		Data struct {
			Text   string `xml:",chardata"`
			Index  string `xml:"index,attr"`
			Name   string `xml:"name,attr"`
			Status string `xml:"status,attr"`
		} `xml:"data"`
	} `xml:"app"`
}
