package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	MAC_BETA = `<?xml version='1.0' encoding='UTF-8'?>
	<request protocol='3.0' version='1.3.23.9' shell_version='1.3.21.103' ismachine='0'
			 sessionid='{3597644B-2952-4F92-AE55-D315F45F80A5}' installsource='ondemandcheckforupdate'
			 requestid='{CD7523AD-A40D-49F4-AEEF-8C114B804658}' dedup='cr'>
	<hw sse='1' sse2='1' sse3='1' ssse3='1' sse41='1' sse42='1' avx='1' physmemory='12582912' />
	<os platform='mac' version='46.0.2490.86' arch='x64'/>
	<app appid='com.google.Chrome' ap='betachannel' version='' nextversion='' lang='' brand='GGLS' client=''>
		<updatecheck/>
	</app>
	</request>`
	MAC_CANARY = `<?xml version='1.0' encoding='UTF-8'?>
	<request protocol='3.0' version='1.3.23.9' shell_version='1.3.21.103' ismachine='0'
			 sessionid='{3597644B-2952-4F92-AE55-D315F45F80A5}' installsource='ondemandcheckforupdate'
			 requestid='{CD7523AD-A40D-49F4-AEEF-8C114B804658}' dedup='cr'>
	<hw sse='1' sse2='1' sse3='1' ssse3='1' sse41='1' sse42='1' avx='1' physmemory='12582912' />
	<os platform='mac' version='46.0.2490.86' arch='x64'/>
	<app appid='com.google.Chrome.Canary' ap='' version='' nextversion='' lang='' brand='GGLS' client=''>
		<updatecheck/>
	</app>
	</request>`
	MAC_DEV = `<?xml version='1.0' encoding='UTF-8'?>
	<request protocol='3.0' version='1.3.23.9' shell_version='1.3.21.103' ismachine='0'
			 sessionid='{3597644B-2952-4F92-AE55-D315F45F80A5}' installsource='ondemandcheckforupdate'
			 requestid='{CD7523AD-A40D-49F4-AEEF-8C114B804658}' dedup='cr'>
	<hw sse='1' sse2='1' sse3='1' ssse3='1' sse41='1' sse42='1' avx='1' physmemory='12582912' />
	<os platform='mac' version='46.0.2490.86' arch='x64'/>
	<app appid='com.google.Chrome' ap='devchannel' version='' nextversion='' lang='' brand='GGLS' client=''>
		<updatecheck/>
	</app>
	</request>`
	MAC_STABLE = `<?xml version='1.0' encoding='UTF-8'?>
	<request protocol='3.0' version='1.3.23.9' shell_version='1.3.21.103' ismachine='0'
			 sessionid='{3597644B-2952-4F92-AE55-D315F45F80A5}' installsource='ondemandcheckforupdate'
			 requestid='{CD7523AD-A40D-49F4-AEEF-8C114B804658}' dedup='cr'>
	<hw sse='1' sse2='1' sse3='1' ssse3='1' sse41='1' sse42='1' avx='1' physmemory='12582912' />
	<os platform='mac' version='46.0.2490.86' arch='x64'/>
	<app appid='com.google.Chrome' ap='' version='' nextversion='' lang='' brand='GGLS' client=''>
		<updatecheck/>
	</app>
	</request>`
	WIN_BETA_X64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x64"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="x64-beta-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_BETA_ARM64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="arm64"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="1.1-beta-arch_arm64-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_BETA_X86 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x86"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="x86-beta-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_CANARY_X64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x64"/>
	<app appid="{4EA16AC7-FD5A-47C3-875B-DBF4A2008C20}" version="" nextversion="" ap="x64-canary-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_CANARY_ARM64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="arm64"/>
	<app appid="{4EA16AC7-FD5A-47C3-875B-DBF4A2008C20}" version="" nextversion="" ap="arm64-canary-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_CANARY_X86 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x86"/>
	<app appid="{4EA16AC7-FD5A-47C3-875B-DBF4A2008C20}" version="" nextversion="" ap="-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_DEV_X64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x64"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="x64-dev-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_DEV_AMR64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="arm64"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="2.0-dev-arch_arm64-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_DEV_X86 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x86"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="x86-dev-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_STABLE_X64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x64"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="x64-stable-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_STABLE_ARM64 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="arm64"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="arm64-stable-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
	WIN_STABLE_X86 = `<?xml version="1.0" encoding="UTF-8"?>
	<request protocol="3.0" updater="Omaha" updaterversion="1.3.36.152" shell_version="1.3.36.151" ismachine="0" sessionid="{11111111-1111-1111-1111-111111111111}" installsource="taggedmi" requestid="{11111111-1111-1111-1111-111111111111}" dedup="cr" domainjoined="0">
	<hw physmemory="16" sse="1" sse2="1" sse3="1" ssse3="1" sse41="1" sse42="1" avx="1"/>
	<os platform="win" version="10.0.22621.1028" sp="" arch="x86"/>
	<app appid="{8A69D345-D564-463C-AFF1-A69D9E530F96}" version="" nextversion="" ap="x86-stable-statsdef_1" lang="de" brand="" client="" installage="-1" installdate="-1" iid="{11111111-1111-1111-1111-111111111111}">
		<updatecheck/>
		<data name="install" index="empty"/>
	</app>
	</request>`
)

var chromeMap = map[string]string{
	"win_stable_x86":   WIN_STABLE_X86,
	"win_stable_x64":   WIN_STABLE_X64,
	"win_stable_arm64": WIN_STABLE_ARM64,
	"win_dev_x86":      WIN_DEV_X86,
	"win_dev_x64":      WIN_DEV_X64,
	"win_dev_arm64":    WIN_DEV_AMR64,
	"win_canary_x86":   WIN_CANARY_X86,
	"win_canary_x64":   WIN_CANARY_X64,
	"win_canary_arm64": WIN_CANARY_ARM64,
	"win_beta_x86":     WIN_BETA_X86,
	"win_beta_x64":     WIN_BETA_X64,
	"win_beta_arm64":   WIN_BETA_ARM64,
	"mac_stable":       MAC_STABLE,
	"mac_dev":          MAC_DEV,
	"mac_beta":         MAC_BETA,
	"mac_canary":       MAC_CANARY}

// 本地获取Chrome版本信息
func getLocalChromeInfo(key string, data *SettingsData) (ChromeInfo, error) {
	updateUrl := "https://tools.google.com/service/update2"
	body := []byte(chromeMap[key])
	req, err := http.NewRequest("POST", updateUrl, bytes.NewBuffer(body))
	if err != nil {
		logger.Error(err)
		return ChromeInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Google Update/1.3.36.152;winhttp")

	// First try without proxy
	info, err := tryRequest(req, nil)
	if err == nil {
		return info, nil
	}
	logger.Warnf("Direct request failed: %v, retrying with proxy", err)

	// If direct request failed, try with proxy
	proxyClient := getHttpProxyClient(data)
	if proxyClient.Transport != nil { // Only if proxy is configured
		return tryRequest(req, proxyClient)
	}

	return ChromeInfo{}, fmt.Errorf("all request attempts failed (last error: %v)", err)
}

func tryRequest(req *http.Request, client *http.Client) (ChromeInfo, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return ChromeInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ChromeInfo{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChromeInfo{}, err
	}

	d := &Response{}
	if err := xml.Unmarshal(responseBody, d); err != nil {
		return ChromeInfo{}, err
	}

	info := ChromeInfo{}
	name := d.App.Updatecheck.Manifest.Packages.Package.Name
	var urls []string
	for _, s := range d.App.Updatecheck.Urls.URL {
		urls = append(urls, s.Codebase+name)
	}
	info.Urls = urls
	info.Version = d.App.Updatecheck.Manifest.Version
	info.Sha256 = strings.ToUpper(d.App.Updatecheck.Manifest.Packages.Package.HashSha256)

	bts, err := base64.StdEncoding.DecodeString(d.App.Updatecheck.Manifest.Packages.Package.Hash)
	if err != nil {
		logger.Errorf("Base64 decoding error: %v", err)
		return ChromeInfo{}, err
	}

	hexString := hex.EncodeToString(bts)
	info.Sha1 = strings.ToUpper(hexString)

	fileSize, err := strconv.ParseInt(d.App.Updatecheck.Manifest.Packages.Package.Size, 10, 64)
	if err != nil {
		logger.Errorf("Size parsing error: %v", err)
		return ChromeInfo{}, err
	}
	info.Size = fileSize
	info.Time = time.Now().UnixMilli()

	return info, nil
}

func getHttpProxyClient(sd *SettingsData) *http.Client {
	ghProxy := getString(sd.ghProxy)
	if ghProxy == "" {
		return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
			Proxy: GetProxyURL(),
		}}
	}

	// Ensure proper proxy URL prefix
	proxyType := getString(sd.proxyType)
	if proxyType == "HTTP(S)" && !strings.HasPrefix(ghProxy, "http") {
		ghProxy = "http://" + ghProxy
	} else if proxyType == "SOCKS5" && !strings.HasPrefix(ghProxy, "socks5") {
		ghProxy = "socks5://" + ghProxy
	}

	urlproxy, err := url.Parse(ghProxy)
	if err != nil {
		logger.Errorf("Invalid proxy URL: %v", err)
		return &http.Client{Timeout: 5 * time.Second}
	}

	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(urlproxy),
		},
	}
}

func GetSystemProxy() (enabled bool, proxyServer string, err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return false, "", err
	}
	defer key.Close()
	enable, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		return false, "", err
	}

	server, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return false, "", err
	}

	return enable == 1, server, nil
}

func GetProxyURL() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		// 1. 检测系统代理
		enabled, proxyServer, err := GetSystemProxy()
		if err == nil && enabled && proxyServer != "" {
			return url.Parse("http://" + proxyServer) // 假设是 HTTP 代理
		}

		// 2. 回退到环境变量
		return http.ProxyFromEnvironment(req)
	}
}
