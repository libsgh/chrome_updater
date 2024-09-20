package main

import (
	"embed"
	"fmt"
	"io"
	"slices"

	"github.com/BurntSushi/toml"
)

//go:embed lang
var langFS embed.FS

var (
	langTable = make(map[string]any)
)

func DelayInitializeLocale() error {
	return SetLocale(defaultLocaleName())
}

func SetLocale(lan string) error {
	if !slices.Contains([]string{"zh-CN", "en-US", "zh-TW"}, lan) {
		lan = "en-US"
	}
	langFile := fmt.Sprint("lang/", lan, ".toml")
	fd, err := langFS.Open(langFile)
	if err != nil {
		return err
	}
	defer fd.Close()
	if _, err := toml.NewDecoder(fd).Decode(&langTable); err != nil {
		return err
	}
	return nil
}

func DefaultLocaleName() string {
	return defaultLocaleName()
}

func LoadString(k string) string {
	if v, ok := langTable[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func LoadStringList(ks ...string) []string {
	msgList := make([]string, 0)
	for _, k := range ks {
		if v, ok := langTable[k]; ok {
			if s, ok := v.(string); ok {
				msgList = append(msgList, s)
			}
		}
	}
	return msgList
}

func Fprintf(w io.Writer, format string, a ...any) (n int, err error) {
	if localeFmt := LoadString(format); len(localeFmt) != 0 {
		format = localeFmt
	}
	return fmt.Fprintf(w, format, a...)
}
