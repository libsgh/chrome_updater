package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"github.com/bodgit/sevenzip"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	versionDLL                 = syscall.NewLazyDLL("version.dll")
	procGetFileVersionInfo     = versionDLL.NewProc("GetFileVersionInfoW")
	procGetFileVersionInfoSize = versionDLL.NewProc("GetFileVersionInfoSizeW")
	procVerQueryValue          = versionDLL.NewProc("VerQueryValueW")
)

// url转换
func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		fyne.LogError("Could not parse URL", err)
	}

	return link
}

// 路径是否合法
func isValidPath(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	cleanAbsPath := filepath.Clean(absPath)
	cleanPath := filepath.Clean(path)
	return cleanAbsPath == cleanPath && dirExist(path)
}

// 字符串是否是数字
func isNumeric(str string) bool {
	_, err := strconv.ParseFloat(str, 64)
	return err == nil
}

// 格式化文件大小
func formatFileSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d Bytes", size)
	}
}

// 获取文件大小
func getFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("服务器返回错误： %v", resp.Status)
	}

	return resp.ContentLength, nil
}

// 下载文件块
func downloadChunk(url string, start, end int64) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("服务器不支持分块下载：%v", resp.Status)
	}

	// 创建一个临时文件用于保存下载的文件块
	chunkPath := fmt.Sprintf("chunk_%d_%d.tmp", start, end)
	chunkFile, err := os.Create(chunkPath)
	if err != nil {
		return "", err
	}
	defer chunkFile.Close()

	_, err = io.Copy(chunkFile, resp.Body)
	if err != nil {
		return "", err
	}

	return chunkPath, nil
}

// 合并文件块
func mergeChunk(chunkPath string, output *os.File) error {
	chunkFile, err := os.Open(chunkPath)
	if err != nil {
		return err
	}
	defer chunkFile.Close()

	_, err = io.Copy(output, chunkFile)
	if err != nil {
		return err
	}

	return nil
}

// 获取文件名
func getFileName(fileURL string) string {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		logger.Error("Failed to parse URL:", err)
		return ""
	}
	filename := path.Base(parsedURL.Path)
	return filename
}

func unzip(zipFile string, filterNames ...string) error {
	parentPath := filepath.Dir(zipFile)
	// 打开 ZIP 文件
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		logger.Errorln("无法打开 ZIP 文件:", err)
		return err
	}
	defer r.Close()
	// 遍历 ZIP 文件中的文件并解压指定文件名的文件
	for _, file := range r.File {
		for _, targetFileName := range filterNames {
			if file.Name == targetFileName {
				rc, err := file.Open()
				if err != nil {
					logger.Error("无法打开 ZIP 文件中的文件:", err)
					return err
				}
				defer rc.Close()

				// 创建解压后的文件
				newFile, err := os.Create(filepath.Join(parentPath, file.Name))
				if err != nil {
					logger.Error("无法创建解压后的文件:", err)
					return err
				}
				defer newFile.Close()

				// 将 ZIP 文件中的内容解压到新文件中
				_, err = io.Copy(newFile, rc)
				if err != nil {
					logger.Error("无法解压 ZIP 文件中的内容:", err)
					return err
				}
				logger.Error("解压文件:", file.Name)
			}
		}
	}
	logger.Debug("解压完成.")
	return nil
}

// 7z解压缩
func UnCompress7z(filePath, targetDir string) {
	r, err := sevenzip.OpenReader(filePath)
	if err != nil {
		logger.Panic(err)
	}
	defer r.Close()
	for _, file := range r.File {
		rc, err := file.Open()
		if err != nil {
			logger.Panic(err)
		}
		defer rc.Close()
		fp := path.Join(targetDir, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(fp, os.ModePerm)
		} else {
			outputFile, err := os.Create(fp)
			if err != nil {
				logger.Panic(err)
			}
			defer outputFile.Close()
			buf := make([]byte, 1*1024*1024)
			_, err = io.CopyBuffer(outputFile, rc, buf)
			//_, err = io.Copy(outputFile, rc)
		}
	}
}

func UnCompress7zFilter(filePath, targetDir, filterName string) error {
	r, err := sevenzip.OpenReader(filePath)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer r.Close()
	for _, file := range r.File {
		if strings.HasPrefix(file.Name, filterName) {
			rc, err := file.Open()
			if err != nil {
				logger.Panic(err)
			}
			defer rc.Close()
			fp := path.Join(targetDir, file.Name)
			if file.FileInfo().IsDir() {
				_ = os.MkdirAll(fp, os.ModePerm)
			} else {
				outputFile, err := os.Create(fp)
				if err != nil {
					logger.Error(err)
					return err
				}
				defer outputFile.Close()
				_, err = io.Copy(outputFile, rc)
			}
		}
	}
	return nil
}

// 计算文件SHA1
func sumFileSHA1(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("无法打开文件:", err)
		return ""
	}
	defer file.Close()
	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		logger.Error("读取文件错误:", err)
		return ""
	}
	hashValue := hash.Sum(nil)
	hashString := hex.EncodeToString(hashValue)
	return strings.ToUpper(hashString)
}

// 检查文件是否存在
func fileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			logger.Error("发生错误:", err)
		}
		return false
	}
	return true
}
func getString(v binding.String) string {
	gv, _ := v.Get()
	return gv
}
func getBool(v binding.Bool) bool {
	gv, _ := v.Get()
	return gv
}
func getStringList(v binding.StringList) []string {
	gv, _ := v.Get()
	return gv
}
func isProcessExist(appPath string) bool {
	// Normalize the input path
	appPath = filepath.Clean(appPath)

	// Create a snapshot of all processes
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		fmt.Printf("CreateToolhelp32Snapshot failed: %v\n", err)
		return false
	}
	defer windows.CloseHandle(snapshot)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	// Get the first process
	err = windows.Process32First(snapshot, &pe)
	if err != nil {
		fmt.Printf("Process32First failed: %v\n", err)
		return false
	}

	for {
		// Open the process to query its full path
		hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe.ProcessID)
		if err == nil {
			// Get the full executable path
			var path [windows.MAX_PATH]uint16
			pathLen := uint32(len(path))
			err = windows.QueryFullProcessImageName(hProcess, 0, &path[0], &pathLen)
			windows.CloseHandle(hProcess)
			if err == nil {
				// Convert UTF16 path to string and normalize
				exePath := windows.UTF16ToString(path[:pathLen])
				exePath = filepath.Clean(exePath)

				// Compare paths (case-insensitive)
				if strings.EqualFold(appPath, exePath) {
					return true
				}
			}
		}

		// Move to the next process
		err = windows.Process32Next(snapshot, &pe)
		if err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			fmt.Printf("Process32Next failed: %v\n", err)
		}
	}

	return false
}
func alertInfo(message string, win fyne.Window) {
	cnf := dialog.NewInformation(LoadString("DialogTooltipTitle"), message, win)
	cnf.SetDismissText(LoadString("DialogCloseLabel"))
	cnf.Show()
}
func alertConfirm(message string, callback func(bool), win fyne.Window) {
	cnf := dialog.NewConfirm(LoadString("DialogConfirmLabel"), message, func(b bool) {
		callback(b)
	}, win)
	cnf.SetDismissText(LoadString("DialogCloseLabel"))
	cnf.SetConfirmText(LoadString("DialogConfirmLabel"))
	cnf.Show()
}
func dirExist(dir string) bool {
	// 打开当前目录
	dirHandle, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer dirHandle.Close()
	return true
}

// 获取系统信息
func getInfo() SysInfo {
	goarch := "x64"
	goos := "win"
	if runtime.GOARCH == "amd64" {
		goarch = "x64"
	} else if runtime.GOARCH == "386" {
		goarch = "x86"
	} else if runtime.GOARCH == "arm64" {
		goarch = "arm64"
	}
	if runtime.GOOS == "darwin" {
		goos = "mac"
	} else if runtime.GOOS == "windows" {
		goos = "win"
	}
	return SysInfo{goarch, goos}
}
func getMapKeys(m map[string]GithubRelease) []string {
	keys := make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	return keys
}
func pathJoin(baseURL, subPath string) string {
	if baseURL == "" {
		return subPath
	}
	return strings.TrimSuffix(baseURL, "/") + "/" + subPath
}
func restartApp(a fyne.App) {
	ex, err := os.Executable()
	if err != nil {
		logger.Errorln(err)
	}
	exeName := filepath.Base(ex)
	parentPath := filepath.Dir(ex)
	updaterPath := filepath.Join(parentPath, exeName)
	cmd := exec.Command("cmd.exe", "/C", updaterPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	a.Quit()
	_ = cmd.Start()
}
func clearOldUpdater() {
	ex, err := os.Executable()
	if err != nil {
		logger.Errorln(err)
	}
	exeName := filepath.Base(ex)
	parentPath := filepath.Dir(ex)
	_ = os.Remove(filepath.Join(parentPath, exeName+"_old"))
}
func handlerErr(err error, message string, win fyne.Window) {
	if err != nil {
		alertInfo(message, win)
	}
}

func GetFileVersion(path string) (string, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	// Get the size of the version info
	size, _, _ := procGetFileVersionInfoSize.Call(uintptr(unsafe.Pointer(p)), 0)
	if size == 0 {
		return "", fmt.Errorf("failed to get version info size")
	}

	// Allocate a buffer to hold the version info
	buffer := make([]byte, size)

	// Get the version info
	ret, _, err := procGetFileVersionInfo.Call(
		uintptr(unsafe.Pointer(p)),
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	if ret == 0 {
		return "", fmt.Errorf("failed to get version info: %v", err)
	}

	// Query the VarFileInfo to find the language and code page
	var block unsafe.Pointer
	var blockLen uint32
	ret, _, err = procVerQueryValue.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("\\VarFileInfo\\Translation"))),
		uintptr(unsafe.Pointer(&block)),
		uintptr(unsafe.Pointer(&blockLen)),
	)
	if ret == 0 {
		return "", fmt.Errorf("failed to query translation value: %v", err)
	}

	translations := (*[1 << 20][2]uint16)(block)[:blockLen/4]
	if len(translations) == 0 {
		return "", fmt.Errorf("no translations found")
	}

	// Use the first translation found
	langCodePage := fmt.Sprintf("%04x%04x", translations[0][0], translations[0][1])
	subBlock := fmt.Sprintf("\\StringFileInfo\\%s\\FileVersion", langCodePage)

	// Query the version value
	ret, _, err = procVerQueryValue.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(subBlock))),
		uintptr(unsafe.Pointer(&block)),
		uintptr(unsafe.Pointer(&blockLen)),
	)
	if ret == 0 {
		return "", fmt.Errorf("failed to query version value: %v", err)
	}

	version := syscall.UTF16ToString((*[1 << 20]uint16)(block)[:blockLen])
	return version, nil
}
func GetVersion(sd *SettingsData, fileName string) string {
	exePath := filepath.Join(getString(sd.installPath), fileName)
	if fileExist(exePath) {
		ver, err := GetFileVersion(exePath)
		if err == nil {
			return ver
		} else {
			logger.Errorln(ver)
		}
	}
	return "-"
}
func UnCompressBy7Zip(filePath, targetDir string) {
	var data []byte
	if runtime.GOARCH == "386" {
		data = resource7z7za386Exe.Content()
	} else if runtime.GOARCH == "amd64" {
		data = resource7z7zaamd64Exe.Content()
	} else if runtime.GOARCH == "arm64" {
		data = resource7z7zaarm64Exe.Content()
	}
	configPath := getConfigPath()
	zipDir := filepath.Dir(configPath)
	if !fileExist(zipDir) {
		_ = os.MkdirAll(zipDir, os.ModePerm)
	}
	zipExePath := filepath.Join(zipDir, "7za.exe")
	if !fileExist(zipExePath) {
		err := os.WriteFile(zipExePath, data, 0644)
		if err != nil {
			logger.Error("write file:", err)
			return
		}
	}
	if fileExist(filepath.Join(targetDir, "chrome.7z")) {
		_ = os.RemoveAll(filepath.Join(targetDir, "chrome.7z"))
	}
	cmd := exec.Command(zipExePath, "e", filePath, "-o"+targetDir, "-aoa", "-bb0")
	logger.Debug(cmd.String())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	if err != nil {
		logger.Errorf("unzip err: %v\n", err)
	}
}

func makeLink(src, dst string) error {
	defer ole.CoUninitialize()
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	if err != nil {
		return err
	}
	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()
	wShell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wShell.Release()
	cs, err := oleutil.CallMethod(wShell, "CreateShortcut", dst)
	if err != nil {
		return err
	}
	dispatch := cs.ToIDispatch()
	_, err = oleutil.PutProperty(dispatch, "TargetPath", src)
	if err != nil {
		return err
	}
	_, err = oleutil.CallMethod(dispatch, "Save")
	if err != nil {
		return err
	}
	return nil
}

func getDesktopPathKnownFolder() (string, error) {
	folderID := syscall.GUID{
		Data1: 0xB4BFCC3A,
		Data2: 0xDB2C,
		Data3: 0x424C,
		Data4: [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41},
	}

	var pathPtr uintptr
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shGetKnownFolderPath := shell32.NewProc("SHGetKnownFolderPath")

	ret, _, err := shGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderID)),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&pathPtr)),
	)

	if ret != 0 {
		return "", fmt.Errorf("SHGetKnownFolderPath failed: %v", err)
	}

	ole32 := syscall.NewLazyDLL("ole32.dll")
	coTaskMemFree := ole32.NewProc("CoTaskMemFree")
	defer coTaskMemFree.Call(pathPtr)

	desktopPath := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(pathPtr)))
	return desktopPath, nil
}

func getDesktopPathFallback() (string, error) {
	var p [syscall.MAX_PATH]uint16
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shGetFolderPath := shell32.NewProc("SHGetFolderPathW")

	ret, _, err := shGetFolderPath.Call(
		uintptr(0),
		uintptr(0x00),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&p[0])),
	)

	if ret != 0 {
		return "", fmt.Errorf("SHGetFolderPath failed: %v", err)
	}

	desktopPath := syscall.UTF16ToString(p[:])
	return desktopPath, nil
}

func GetDesktopPath() (string, error) {
	p, err := getDesktopPathKnownFolder()
	if err == nil {
		return p, nil
	}

	p, err = getDesktopPathFallback()
	if err == nil {
		return p, nil
	}

	return "", errors.New("failed to get desktop path")
}
