//go:build windows

package main

import (
	"encoding/json"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/win"
)

const tabContainerClass = "XBrowserTabContainer"

func registerTabContainerClass(hInstance win.HINSTANCE) {
	wc := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(win.DefWindowProc),
		HInstance:     hInstance,
		HCursor:       win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW)),
		LpszClassName: syscall.StringToUTF16Ptr(tabContainerClass),
	}
	win.RegisterClassEx(&wc)
}

type Tab struct {
	hwnd       win.HWND
	chromium   *edge.Chromium
	title      string
	url        string
	profile    string
	ready      bool
	themeTried string
	badge      int
	kind       string
	selection  string

	lastTranslate time.Time
	lastOpenTab   time.Time
}

func (t *Tab) Eval(js string) {
	if t.chromium != nil {
		t.chromium.Eval(js)
	}
}

type TabManager struct {
	parent      win.HWND
	tabs        []*Tab
	active      int
	lastContent win.RECT
}

func newTabManager(parent win.HWND) *TabManager {
	initDebugLog(appBaseDir())
	debugLog("TabManager作成 baseDir=%s", appBaseDir())
	return &TabManager{parent: parent, active: -1}
}

func (m *TabManager) Active() *Tab {
	if m.active < 0 || m.active >= len(m.tabs) {
		return &Tab{}
	}
	return m.tabs[m.active]
}

func (m *TabManager) AddTab(url string, profile string) {
	hInstance := win.GetModuleHandle(nil)
	child := win.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr(tabContainerClass),
		syscall.StringToUTF16Ptr(""),
		win.WS_CHILD|win.WS_VISIBLE|win.WS_CLIPCHILDREN|win.WS_CLIPSIBLINGS,
		0, 0, 10, 10,
		m.parent, 0, hInstance, nil,
	)

	tab := &Tab{hwnd: child, url: url, title: T("title_new"), profile: profile, kind: kindOfURL(url)}
	m.tabs = append(m.tabs, tab)
	newIndex := len(m.tabs) - 1

	chromium := edge.NewChromium()
	chromium.DataPath = profileDataDir(profile)
	debugLog("プロファイル=%s dataDir=%s", profileLabel(profile), chromium.DataPath)
	chromium.MessageCallback = func(msg string) { handleTabMessage(tab, msg) }
	chromium.NavigationCompletedCallback = func(sender *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) {
		debugLog("NavigationCompleted url=%s", url)
		applyZoomTo(chromium)
	}
	tab.chromium = chromium
	chromium.AcceleratorKeyCallback = func(vk uint) bool {
		if win.GetKeyState(win.VK_CONTROL) >= 0 {
			return false
		}
		switch vk {
		case 0xBB, 0x6B:
			m.ZoomStep(1)
			return true
		case 0xBD, 0x6D:
			m.ZoomStep(-1)
			return true
		case 0x30, 0x60:
			m.ZoomReset()
			return true
		case 0x4C:
			win.PostMessage(app.hwnd, wmAppAction, actFocusAddress, 0)
			return true
		case 0x4E:
			win.PostMessage(app.hwnd, wmAppAction, actNewTab, 0)
			return true
		case 0x44:
			if win.GetKeyState(win.VK_SHIFT) < 0 {
				win.PostMessage(app.hwnd, wmAppAction, actToggleTheme, 0)
				return true
			}
		case 0xBC:
			win.PostMessage(app.hwnd, wmAppAction, actOpenSettings, 0)
			return true
		}
		return false
	}

	debugLog("Embed開始 url=%s", url)
	if !chromium.Embed(uintptr(child)) {
		debugLog("Embed失敗")

		win.MessageBox(m.parent,
			mustUTF16(T("err_webview2")),
			mustUTF16("xbrowser"), win.MB_ICONERROR)
		os.Exit(1)
	}
	debugLog("Embed成功")
	tab.ready = true
	applyZoomTo(chromium)

	if settings, err := chromium.GetSettings(); err == nil {
		_ = settings.PutAreDefaultContextMenusEnabled(true)
		_ = settings.PutAreDevToolsEnabled(false)

		_ = settings.PutIsZoomControlEnabled(false)
	}

	if controller := chromium.GetController(); controller != nil {
		if c2 := controller.GetICoreWebView2Controller2(); c2 != nil {
			_ = c2.PutDefaultBackgroundColor(edge.COREWEBVIEW2_COLOR{A: 255, R: 255, G: 255, B: 255})
		}
	}

	applyNotificationSetting(chromium)

	chromium.Init(injectScript())
	if tab.kind != kindWeb {
		debugLog("ローカルページ表示 kind=%s", tab.kind)
		mapBackgroundHost(chromium)
		tab.loadLocal()
	} else {
		debugLog("Navigate呼び出し url=%s", url)
		chromium.Navigate(url)
	}

	m.Switch(newIndex)
}

// WebView2は作成後にDataPathを変えられないため、閉じて同じ位置に開き直す
func (m *TabManager) SwitchTabProfile(idx int, profile string) {
	if idx < 0 || idx >= len(m.tabs) {
		return
	}
	url := m.tabs[idx].url
	if url == "" || (!isLocalURL(url) && !isHTTPURL(url)) {
		url = homeURL
	}
	win.DestroyWindow(m.tabs[idx].hwnd)
	m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)

	m.AddTab(url, profile)
	last := len(m.tabs) - 1
	t := m.tabs[last]
	m.tabs = m.tabs[:last]
	tail := append([]*Tab{t}, m.tabs[idx:]...)
	m.tabs = append(m.tabs[:idx], tail...)
	m.active = idx
}

func (m *TabManager) Switch(i int) {
	if i < 0 || i >= len(m.tabs) {
		return
	}
	for idx, t := range m.tabs {
		if idx != i {
			win.ShowWindow(t.hwnd, win.SW_HIDE)
			if t.ready {
				_ = t.chromium.Hide()
			}
		}
	}
	m.active = i
	win.ShowWindow(m.tabs[i].hwnd, win.SW_SHOW)
	m.Refresh()
	if m.tabs[i].ready {
		m.tabs[i].chromium.Focus()
		m.tabs[i].pushState()
	}
	refreshAddress(false)
}

func (m *TabManager) Refresh() {
	if m.active < 0 || m.active >= len(m.tabs) {
		return
	}
	t := m.tabs[m.active]
	if !t.ready {
		return
	}
	m.resizeActive()
	{
		_ = t.chromium.Show()
		_ = t.chromium.NotifyParentWindowPositionChanged()
	}
}

func (m *TabManager) Close(i int) {
	if i < 0 || i >= len(m.tabs) {
		return
	}
	win.DestroyWindow(m.tabs[i].hwnd)
	m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)

	if len(m.tabs) == 0 {
		win.DestroyWindow(m.parent)
		return
	}
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	}
	m.Switch(m.active)
}

func (m *TabManager) Resize(content win.RECT) {
	m.lastContent = content
	m.resizeActive()
}

func (m *TabManager) resizeActive() {
	if m.active < 0 || m.active >= len(m.tabs) {
		return
	}
	r := m.lastContent
	win.MoveWindow(m.tabs[m.active].hwnd, r.Left, r.Top, r.Right-r.Left, r.Bottom-r.Top, true)
	if m.tabs[m.active].ready {
		m.tabs[m.active].chromium.Resize()
	}
}

const maxMessageLen = 64 * 1024

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func isHTTPURL(raw string) bool {
	if len(raw) > maxExternalURLLen {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

type tabMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func handleTabMessage(tab *Tab, raw string) {
	if len(raw) > maxMessageLen {
		return
	}
	var msg tabMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return
	}
	debugLog("MessageCallback type=%s", msg.Type)
	switch msg.Type {
	case "open_external":
		if tab.kind == kindWeb {
			openExternal(msg.Data)
		}
	case "open_tab":
		if tab.kind == kindWeb && isXURL(msg.Data) && time.Since(tab.lastOpenTab) > 300*time.Millisecond {
			tab.lastOpenTab = time.Now()
			queueOpenTab(msg.Data, tab.profile)
		}
	case "translate":
		if tab.kind == kindWeb {
			var req struct {
				ID   int    `json:"id"`
				Text string `json:"text"`
			}
			if json.Unmarshal([]byte(msg.Data), &req) == nil && time.Since(tab.lastTranslate) > 300*time.Millisecond {
				tab.lastTranslate = time.Now()
				startTranslate(tab, req.ID, req.Text)
			}
		}
	case "translate_open":
		if tab.kind == kindWeb {
			translateText(msg.Data)
		}
	case "selection":
		if tab.kind == kindWeb {
			tab.selection = cleanSelection(msg.Data)
		}
	case "title":
		if tab.kind == kindWeb {
			tab.title = truncateRunes(badgePrefix.ReplaceAllString(msg.Data, ""), 200)
			win.InvalidateRect(app.hwnd, nil, false)
		}
	case "zoom":
		switch msg.Data {
		case "in":
			app.tabs.ZoomStep(1)
		case "out":
			app.tabs.ZoomStep(-1)
		case "reset":
			app.tabs.ZoomReset()
		}
	case "url":
		if tab.kind == kindWeb && isHTTPURL(msg.Data) {
			tab.url = msg.Data
			if tab == app.tabs.Active() {
				refreshAddress(false)
			}
		}
	case "badge":
		if n, err := strconv.Atoi(msg.Data); err == nil && n >= 0 && n != tab.badge {
			tab.badge = n
			invalidateBar()
		}
	case "theme_cookie":
		tab.syncWebTheme(msg.Data)

	case "start_nav":
		if tab.kind == kindStart && isXURL(msg.Data) {
			tab.navigateWeb(msg.Data)
			win.InvalidateRect(app.hwnd, nil, false)
		}
	case "open_settings":
		if tab.kind == kindStart {
			win.PostMessage(app.hwnd, wmAppAction, actOpenSettings, 0)
		}
	case "settings_set":
		if tab.kind == kindSettings {
			handleSettingsSet(tab, msg.Data)
		}
	case "settings_action":
		if tab.kind == kindSettings {
			handleSettingsAction(tab, msg.Data)
		}
	}
}

var badgePrefix = regexp.MustCompile(`^\(\d+\)\s*`)

func (m *TabManager) MoveTab(from, to int) {
	if from == to || from < 0 || to < 0 || from >= len(m.tabs) || to >= len(m.tabs) {
		return
	}
	t := m.tabs[from]
	rest := append(append([]*Tab{}, m.tabs[:from]...), m.tabs[from+1:]...)
	m.tabs = append(append(append([]*Tab{}, rest[:to]...), t), rest[to:]...)
	a := m.active
	switch {
	case a == from:
		m.active = to
	case from < a && a <= to:
		m.active--
	case to <= a && a < from:
		m.active++
	}
}
