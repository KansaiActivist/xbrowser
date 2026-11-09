//go:build windows

package main

import (
	"strings"
	u16 "unicode/utf16"
	"unsafe"

	"github.com/lxn/win"
)

const (
	copyDataOpenURL = 0x58425531
	maxOpenTabs     = 40
	maxPendingTabs  = 8
	asfwAny         = ^uintptr(0)
)

type copyDataStruct struct {
	DwData uintptr
	CbData uint32
	LpData uintptr
}

type pendingTab struct{ url, profile string }

var (
	pendingTabs []pendingTab
	startupURL  string

	pAllowSetForegroundWindow = user32b.NewProc("AllowSetForegroundWindow")
)

func normalizeXURL(arg string) (string, bool) {
	arg = strings.TrimSpace(arg)
	if arg == "" || len(arg) > maxExternalURLLen {
		return "", false
	}
	if !strings.Contains(arg, "://") {
		arg = "https://" + arg
	}
	if strings.HasPrefix(strings.ToLower(arg), "http://") {
		arg = "https://" + arg[len("http://"):]
	}
	if !isXURL(arg) {
		return "", false
	}
	return arg, true
}

func urlFromArgs(args []string) string {
	for _, a := range args {
		if u, ok := normalizeXURL(a); ok {
			return u
		}
	}
	return ""
}

func queueOpenTab(url, profile string) {
	if len(pendingTabs) >= maxPendingTabs || app == nil {
		return
	}
	pendingTabs = append(pendingTabs, pendingTab{url, profile})
	win.PostMessage(app.hwnd, wmAppAction, actOpenTabs, 0)
}

func openPendingTabs() {
	list := pendingTabs
	pendingTabs = nil
	for _, p := range list {
		if len(app.tabs.tabs) >= maxOpenTabs {
			return
		}
		app.tabs.AddTab(p.url, p.profile)
	}
}

func bringToFront(hwnd win.HWND) {
	if win.IsIconic(hwnd) {
		win.ShowWindow(hwnd, win.SW_RESTORE)
	}
	win.SetForegroundWindow(hwnd)
}

func sendURLToRunning(existing win.HWND, url string) {
	u := u16.Encode([]rune(url))
	if len(u) == 0 {
		return
	}
	pAllowSetForegroundWindow.Call(asfwAny)
	cds := copyDataStruct{DwData: copyDataOpenURL, CbData: uint32(len(u) * 2), LpData: uintptr(unsafe.Pointer(&u[0]))}
	win.SendMessage(existing, win.WM_COPYDATA, 0, uintptr(unsafe.Pointer(&cds)))
}

func handleCopyData(lParam uintptr) {
	cds := (*copyDataStruct)(unsafe.Pointer(lParam))
	if cds == nil || cds.DwData != copyDataOpenURL || cds.CbData == 0 || cds.CbData > maxExternalURLLen*2 || cds.CbData%2 != 0 {
		return
	}
	n := int(cds.CbData / 2)
	u := make([]uint16, n)
	copy(u, unsafe.Slice((*uint16)(unsafe.Pointer(cds.LpData)), n))
	if target, ok := normalizeXURL(string(u16.Decode(u))); ok && app != nil {
		queueOpenTab(target, app.tabs.Active().profile)
		bringToFront(app.hwnd)
	}
}
