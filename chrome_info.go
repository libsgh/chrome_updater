package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"io"
	"net/http"
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
func getLocalChromeInfo(key string) ChromeInfo {
	url := "https://tools.google.com/service/update2"
	body := []byte(chromeMap[key])
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Google Update/1.3.36.152;winhttp")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Panic(err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err)
	}
	d := &Response{}
	xml.Unmarshal(responseBody, d)
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
		logger.Errorf("Base64 decoding error: %v\n", err)
	}
	hexString := hex.EncodeToString(bts)
	info.Sha1 = strings.ToUpper(hexString)
	fileSize, _ := strconv.ParseInt(d.App.Updatecheck.Manifest.Packages.Package.Size, 10, 64)
	info.Size = fileSize
	info.Time = time.Now().UnixMilli()
	return info
}
