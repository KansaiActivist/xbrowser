//go:build windows

package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func themeMode() string {
	switch strings.ToLower(strings.TrimSpace(cfg.Theme.Preset)) {
	case "light":
		return "light"
	case "system":
		return "system"
	}
	return "dark"
}

func systemIsLight() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err == nil && v != 0
}

func effectivePreset() string {
	switch themeMode() {
	case "light":
		return "light"
	case "system":
		if systemIsLight() {
			return "light"
		}
	}
	return "dark"
}

// Xの night_mode クッキー: 0=ライト 1=ダーク(Dim) 2=ダーク(Lights out)
func webNightMode() string {
	if effectivePreset() == "light" {
		return "0"
	}
	return "2"
}

func setNightModeJS(mode string) string {
	return `(function(){try{var h=location.hostname;if(h!=='x.com'&&!/\.x\.com$/.test(h)&&h!=='twitter.com'&&!/\.twitter\.com$/.test(h))return;` +
		`var d=h.indexOf('twitter')>=0?'.twitter.com':'.x.com';` +
		`document.cookie='night_mode=` + mode + `; domain='+d+'; path=/; max-age=31536000; secure';location.reload();}catch(e){}})()`
}

// ページから報告された現在のクッキー値が希望とずれていれば、1回だけ書き換えて再読み込みする
func (t *Tab) syncWebTheme(cur string) {
	if cfg.ThemeSync == "off" || t.kind != kindWeb || !t.ready {
		return
	}
	want := webNightMode()
	if cur == want || (cur == "" && want == "0") || t.themeTried == want {
		return
	}
	t.themeTried = want
	t.Eval(setNightModeJS(want))
}

func applyWebThemeAll() {
	for _, t := range app.tabs.tabs {
		t.themeTried = ""
		if t.kind == kindWeb && t.ready && cfg.ThemeSync != "off" {
			t.themeTried = webNightMode()
			t.Eval(setNightModeJS(webNightMode()))
		}
	}
}

func themeApplied() {
	applyTheme(true)
	app.tabs.pushStateAll()
	applyWebThemeAll()
}

func toggleTheme() {
	next := "light"
	if effectivePreset() == "light" {
		next = "dark"
	}
	cfg.Theme = Theme{Preset: next, TabRadius: cfg.Theme.TabRadius}
	themeApplied()
}

func systemThemeChanged() {
	if themeMode() != "system" {
		return
	}
	prev := palette.bar
	resolveTheme()
	if palette.bar == prev {
		return
	}
	invalidateBar()
	app.tabs.pushStateAll()
	applyWebThemeAll()
}
