package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"strings"
)

type MyTheme struct {
	Theme binding.String
	Lang  binding.String
}

var _ fyne.Theme = (*MyTheme)(nil)

// Font return bundled font resource
func (mt MyTheme) Font(s fyne.TextStyle) fyne.Resource {
	lang := getString(mt.Lang)
	if lang == "zh-CN" || ((lang == "System" || lang == "" || lang == "zh-TW") && DefaultLocaleName() == "zh-CN") {
		if s.Bold {
			return resourceAssetsFontAlibabaPuHuiTi385BoldTtf
		}
		return resourceAssetsFontAlibabaPuHuiTi355RegularTtf
	}
	return theme.DefaultTheme().Font(s)
}
func (mt *MyTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	t, _ := mt.Theme.Get()
	if strings.Contains(t, LoadString("ThemeLightOption")) {
		v = theme.VariantLight
	} else if strings.Contains(t, LoadString("ThemeDarkOption")) {
		v = theme.VariantDark
	} else {
		v = 2
	}
	return theme.DefaultTheme().Color(n, v)
}

func (*MyTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (*MyTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}
