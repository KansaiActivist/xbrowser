//go:build windows

package main

import (
	"net/url"
	"strings"
	"syscall"

	"github.com/lxn/win"
)

const (
	enSetFocus  = 0x0100
	enKillFocus = 0x0200
	idAddress   = win.HMENU(100)
)

var (
	addrEdit      win.HWND
	addrOrigProc  uintptr
	addrFont      win.HFONT
	addrBrush     win.HBRUSH
	addrBrushCol  win.COLORREF
	addrBrushInit bool
)

func createAddressBar(parent win.HWND) {
	hInstance := win.GetModuleHandle(nil)
	addrEdit = win.CreateWindowEx(0, syscall.StringToUTF16Ptr("EDIT"), syscall.StringToUTF16Ptr(""),
		win.WS_CHILD|win.WS_VISIBLE|win.ES_AUTOHSCROLL, 0, 0, 100, 22, parent, idAddress, hInstance, nil)
	addrFont = makeFont("Segoe UI", -15, win.FW_NORMAL)
	win.SendMessage(addrEdit, win.WM_SETFONT, uintptr(addrFont), 1)
	addrOrigProc = win.SetWindowLongPtr(addrEdit, win.GWLP_WNDPROC, syscall.NewCallback(addrProc))
}

func addrProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_KEYDOWN:
		switch wParam {
		case uintptr(win.VK_RETURN):
			submitAddress()
			return 0
		case uintptr(win.VK_ESCAPE):
			refreshAddress(true)
			focusActiveTab()
			return 0
		case 'A':
			if win.GetKeyState(win.VK_CONTROL) < 0 {
				win.SendMessage(hwnd, win.EM_SETSEL, 0, ^uintptr(0))
				return 0
			}
		}
	case win.WM_CHAR:
		if wParam == 13 || wParam == 27 || wParam == 1 {
			return 0
		}
	case win.WM_LBUTTONDOWN:
		if win.GetFocus() != hwnd {
			win.SetFocus(hwnd)
			win.SendMessage(hwnd, win.EM_SETSEL, 0, ^uintptr(0))
			return 0
		}
	}
	return win.CallWindowProc(addrOrigProc, hwnd, msg, wParam, lParam)
}

func addressFocused() bool { return addrEdit != 0 && win.GetFocus() == addrEdit }

func addrPillColor() win.COLORREF {
	if addressFocused() {
		return palette.tabActive
	}
	return palette.tabHover
}

func addrCtlColor(hdc win.HDC) uintptr {
	col := addrPillColor()
	win.SetTextColor(hdc, palette.tabText)
	win.SetBkColor(hdc, col)
	if !addrBrushInit || addrBrushCol != col {
		if addrBrushInit {
			win.DeleteObject(win.HGDIOBJ(addrBrush))
		}
		addrBrush = createSolidBrush(col)
		addrBrushCol = col
		addrBrushInit = true
	}
	return uintptr(addrBrush)
}

func layoutAddressBar() {
	if addrEdit == 0 {
		return
	}
	r := app.addressRect()
	const h = 22
	x := r.Left + 38
	w := r.Right - r.Left - 38 - 16
	if w < 20 {
		w = 20
	}
	win.MoveWindow(addrEdit, x, r.Top+(r.Bottom-r.Top-h)/2, w, h, true)
}

func focusAddress() {
	if addrEdit == 0 {
		return
	}
	win.SetFocus(addrEdit)
	win.SendMessage(addrEdit, win.EM_SETSEL, 0, ^uintptr(0))
}

func focusActiveTab() {
	if t := app.tabs.Active(); t.ready {
		t.chromium.Focus()
	}
}

func refreshAddress(force bool) {
	if addrEdit == 0 || app == nil || app.tabs == nil {
		return
	}
	if !force && addressFocused() {
		return
	}
	text := ""
	if t := app.tabs.Active(); t.kind == kindWeb {
		text = t.url
	}
	setWindowText(addrEdit, text)
}

func isXHost(host string) bool {
	host = strings.ToLower(host)
	for _, s := range allowedHostSuffixes {
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}
	return false
}

func searchURL(q string) string {
	return "https://x.com/search?q=" + url.QueryEscape(q) + "&src=typed_query"
}

func resolveAddress(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, " \t") {
		return searchURL(s)
	}
	if strings.HasPrefix(s, "@") && len(s) > 1 {
		return "https://x.com/" + url.PathEscape(s[1:])
	}
	cand := s
	if !strings.Contains(s, "://") {
		if !strings.Contains(s, ".") {
			return searchURL(s)
		}
		cand = "https://" + s
	}
	u, err := url.Parse(cand)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return searchURL(s)
	}
	if !strings.Contains(u.Hostname(), ".") {
		return searchURL(s)
	}
	if u.Scheme == "http" && isXHost(u.Hostname()) {
		u.Scheme = "https"
	}
	return u.String()
}

func submitAddress() {
	buf := make([]uint16, 2048)
	n := getWindowText(addrEdit, buf)
	target := resolveAddress(syscall.UTF16ToString(buf[:n]))
	if target == "" {
		refreshAddress(true)
		return
	}
	if !isXURL(target) {
		openExternal(target)
		refreshAddress(true)
		focusActiveTab()
		return
	}
	t := app.tabs.Active()
	if !t.ready {
		return
	}
	if t.kind != kindWeb {
		t.navigateWeb(target)
	} else {
		t.url = target
		t.chromium.Navigate(target)
	}
	refreshAddress(true)
	invalidateBar()
	t.chromium.Focus()
}
