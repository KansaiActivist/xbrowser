//go:build windows

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/lxn/win"
)

//go:embed pages/settings.html
var settingsHTML string

//go:embed pages/start.html
var startHTML string

//go:embed pages/i18n.js
var i18nJS string

const (
	startURL    = "xbrowser://start"
	settingsURL = "xbrowser://settings"

	kindWeb      = "web"
	kindStart    = "start"
	kindSettings = "settings"

	wmAppAction     = win.WM_APP + 1
	actOpenSettings = 1
	actOpenProfile  = 2
	actPickImage    = 3
	actFocusAddress = 4
	actToggleTheme  = 5
)

var pendingProfile string

func isLocalURL(u string) bool { return u == startURL || u == settingsURL }

func kindOfURL(u string) string {
	switch u {
	case startURL:
		return kindStart
	case settingsURL:
		return kindSettings
	}
	return kindWeb
}

func newTabURL() string {
	if cfg.StartPage == "simple" {
		return startURL
	}
	return homeURL
}

var allowedHostSuffixes = []string{"x.com", "twitter.com", "twimg.com", "twimg.net"}

func isXURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, s := range allowedHostSuffixes {
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}
	return false
}

func colorHex(c win.COLORREF) string {
	return fmt.Sprintf("#%02X%02X%02X", uint32(c)&0xff, (uint32(c)>>8)&0xff, (uint32(c)>>16)&0xff)
}

type themeField struct {
	key   string
	field *string
	cur   *win.COLORREF
}

func themeFields() []themeField {
	t := &cfg.Theme
	return []themeField{
		{"barBackground", &t.BarBackground, &palette.bar},
		{"tabActive", &t.TabActive, &palette.tabActive},
		{"tabInactive", &t.TabInactive, &palette.tabInactive},
		{"tabHover", &t.TabHover, &palette.tabHover},
		{"tabText", &t.TabText, &palette.tabText},
		{"tabTextInactive", &t.TabTextInactive, &palette.tabTextInactive},
		{"icon", &t.Icon, &palette.icon},
		{"buttonHover", &t.ButtonHover, &palette.buttonHover},
		{"closeHover", &t.CloseHover, &palette.closeHover},
		{"closeHoverIcon", &t.CloseHoverIcon, &palette.closeHoverIcon},
	}
}

func presetName() string { return effectivePreset() }

func settingsState() map[string]any {
	colors := map[string]string{}
	for _, f := range themeFields() {
		colors[f.key] = colorHex(*f.cur)
	}
	return map[string]any{
		"startPage":     cfg.StartPage,
		"notifications": cfg.Notifications,
		"zoom":          cfg.Zoom,
		"preset":        presetName(),
		"presetMode":    themeMode(),
		"themeSync":     cfg.ThemeSync,
		"radius":        palette.radius,
		"colors":        colors,
		"profiles":      profileStates(),
		"palette":       profilePaletteHex(),
		"configPath":    configPath(),
		"lang":          cfg.Language,
		"startBg":       cfg.StartBg,
		"startImageUrl": startImageURL(),
	}
}

func profileStates() []map[string]any {
	out := []map[string]any{}
	for _, id := range allProfiles() {
		color, custom := profileColorHex(id)
		out = append(out, map[string]any{
			"id":      id,
			"name":    cfg.ProfileNames[id],
			"color":   color,
			"custom":  custom,
			"default": id == "Default",
		})
	}
	return out
}

func (t *Tab) pageState() map[string]any {
	state := settingsState()
	state["profile"] = profileLabel(t.profile)
	state["profileColor"], _ = profileColorHex(t.profile)
	return state
}

func (t *Tab) loadLocal() {
	b, _ := json.Marshal(t.pageState())

	page := settingsHTML
	if t.kind == kindStart {
		page = startHTML
	}
	t.title = t.localTitle()
	page = strings.Replace(page, "/*__I18N__*/", i18nJS, 1)
	page = strings.Replace(page, "/*__STATE__*/null", string(b), 1)
	t.chromium.NavigateToString(page)
}

func (t *Tab) localTitle() string {
	if t.kind == kindStart {
		return T("title_start")
	}
	return T("title_settings")
}

func (t *Tab) navigateWeb(u string) {
	t.kind = kindWeb
	t.url = u
	t.title = "X"
	t.chromium.Navigate(u)
}

func (t *Tab) pushState() {
	if !t.ready || t.kind == kindWeb {
		return
	}
	b, _ := json.Marshal(t.pageState())
	t.Eval("window.__setState&&window.__setState(" + string(b) + ")")
}

func (t *Tab) toast(msg string) {
	b, _ := json.Marshal(msg)
	t.Eval("window.__toast&&window.__toast(" + string(b) + ")")
}

func (m *TabManager) pushStateAll() {
	for _, t := range m.tabs {
		t.pushState()
	}
}

func (m *TabManager) OpenSettings() {
	for i, t := range m.tabs {
		if t.kind == kindSettings {
			m.Switch(i)
			return
		}
	}
	m.AddTab(settingsURL, "")
}

var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func applyTheme(save bool) {
	resolveTheme()
	if save {
		saveConfig()
	}
	if app != nil {
		invalidateBar()
	}
}

type settingsSet struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	Save  bool            `json:"save"`
}

func handleSettingsSet(tab *Tab, data string) {
	var p settingsSet
	if json.Unmarshal([]byte(data), &p) != nil {
		return
	}
	var str string
	_ = json.Unmarshal(p.Value, &str)

	switch p.Key {
	case "language":
		if isLang(str) {
			cfg.Language = str
			saveConfig()
			for _, t := range app.tabs.tabs {
				if t.kind != kindWeb {
					t.title = t.localTitle()
				} else if t.ready {
					t.Eval("window.__xb_lang=" + strconv.Quote(cfg.Language))
				}
			}
			invalidateBar()
		}
	case "startBg":
		if str == "white" || str == "black" || (str == "image" && startImageURL() != "") {
			cfg.StartBg = str
			saveConfig()
		}
	case "startPage":
		if str == "simple" || str == "x" {
			cfg.StartPage = str
			saveConfig()
		}
	case "notifications":
		if str == "on" || str == "off" {
			cfg.Notifications = str
			saveConfig()
			app.tabs.applyNotificationSettingAll()
		}
	case "preset":
		if str == "dark" || str == "light" || str == "system" {

			cfg.Theme = Theme{Preset: str, TabRadius: cfg.Theme.TabRadius}
			themeApplied()
			return
		}
	case "themeSync":
		if str == "on" || str == "off" {
			cfg.ThemeSync = str
			saveConfig()
			if str == "on" {
				applyWebThemeAll()
			}
		}
	case "tabRadius":
		var n int
		if json.Unmarshal(p.Value, &n) == nil {
			if n < 2 {
				n = 2
			}
			if n > 24 {
				n = 24
			}
			cfg.Theme.TabRadius = n
			applyTheme(p.Save)
		}
		return
	default:
		for _, f := range themeFields() {
			if f.key == p.Key && hexColorRe.MatchString(str) {
				*f.field = strings.ToUpper(str)
				applyTheme(p.Save)
				return
			}
		}
		return
	}
	app.tabs.pushStateAll()
}

type settingsAct struct {
	Action string `json:"action"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

func handleSettingsAction(tab *Tab, data string) {
	var p settingsAct
	if json.Unmarshal([]byte(data), &p) != nil {
		return
	}
	name := strings.TrimSpace(p.Name)

	switch p.Action {
	case "reset_colors":
		cfg.Theme = Theme{Preset: cfg.Theme.Preset}
		applyTheme(true)
	case "add_profile":
		if name == "" {
			return
		}
		if key := validateNewProfile(name); key != "" {
			tab.toast(T(key, name))
			return
		}
		addProfile(name)
		tab.toast(T("toast_added", name))
	case "remove_profile":
		if removeProfile(name) {
			tab.toast(T("toast_removed", name))
		}
	case "rename_profile":
		newName := strings.TrimSpace(p.Value)
		switch key := renameProfile(name, newName); key {
		case "":
			tab.toast(T("toast_renamed", newName))
		case "toast_exists":
			tab.toast(T(key, newName))
			return
		default:
			tab.toast(T(key))
			return
		}
		invalidateBar()
	case "set_profile_color":
		if !setProfileColor(name, strings.TrimSpace(p.Value)) {
			return
		}
		invalidateBar()
	case "open_profile":

		if strings.EqualFold(name, "Default") {
			name = ""
		}
		pendingProfile = name
		win.PostMessage(app.hwnd, wmAppAction, actOpenProfile, 0)
		return
	case "pick_image":

		win.PostMessage(app.hwnd, wmAppAction, actPickImage, 0)
		return
	case "clear_image":
		clearBackground()
	case "zoom_in":
		app.tabs.ZoomStep(1)
		return
	case "zoom_out":
		app.tabs.ZoomStep(-1)
		return
	case "zoom_reset":
		app.tabs.ZoomReset()
		return
	case "open_config":
		_ = exec.Command("notepad.exe", configPath()).Start()
		return
	}
	app.tabs.pushStateAll()
}

func pickBackgroundImage() {
	path, ok := pickImageFile(app.hwnd)
	if !ok {
		return
	}
	key := importBackground(path)
	app.tabs.pushStateAll()
	for _, t := range app.tabs.tabs {
		if t.kind != kindSettings {
			continue
		}
		if key == "" {
			key = "toast_img_ok"
		}
		t.toast(T(key))
	}
}
