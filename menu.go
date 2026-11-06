//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

var pAppendMenuW = user32b.NewProc("AppendMenuW")

func appendMenu(m win.HMENU, flags uint32, id uintptr, text string) {
	var p *uint16
	if flags&win.MF_SEPARATOR == 0 {
		p, _ = syscall.UTF16PtrFromString(text)
	}
	pAppendMenuW.Call(uintptr(m), uintptr(flags), id, uintptr(unsafe.Pointer(p)))
}

const (
	idNewTab            = 1
	idCloseTab          = 2
	idZoomReset         = 3
	idSettings          = 4
	idStartPage         = 5
	idTranslate         = 6
	idProfileAdd        = 15
	idProfileNewBase    = 300
	idProfileSwitchBase = 400
)

func themeChanged() { applyTheme(true) }

func showContextMenu(hwnd win.HWND, x, y int32) {
	zone, tabIdx := app.hitTest(x, y)

	menu := win.CreatePopupMenu()
	defer win.DestroyMenu(menu)

	appendMenu(menu, win.MF_STRING, idNewTab, T("menu_new_tab"))

	profiles := allProfiles()
	newTabProfiles := win.CreatePopupMenu()
	for i, p := range profiles {
		appendMenu(newTabProfiles, win.MF_STRING, uintptr(idProfileNewBase+i), profileMenuLabel(p))
	}
	appendMenu(newTabProfiles, win.MF_SEPARATOR, 0, "")
	appendMenu(newTabProfiles, win.MF_STRING, idProfileAdd, T("menu_profile_add"))
	appendMenu(menu, win.MF_POPUP, uintptr(newTabProfiles), T("menu_profile_new"))

	if zone == zoneTab || zone == zoneTabClose {
		appendMenu(menu, win.MF_STRING, idCloseTab, T("menu_close_tab"))
	}
	if (zone == zoneTab || zone == zoneTabClose) && app.tabs.tabs[tabIdx].kind != kindSettings {

		switchMenu := win.CreatePopupMenu()
		curID := profileID(app.tabs.tabs[tabIdx].profile)
		for i, p := range profiles {
			f := uint32(win.MF_STRING)
			if strings.EqualFold(p, curID) {
				f |= win.MF_CHECKED
			}
			appendMenu(switchMenu, f, uintptr(idProfileSwitchBase+i), profileMenuLabel(p))
		}
		appendMenu(menu, win.MF_POPUP, uintptr(switchMenu), T("menu_tab_profile", profileMenuLabel(curID)))
	}
	if t := app.tabs.Active(); t.kind == kindWeb && t.selection != "" {
		appendMenu(menu, win.MF_STRING, idTranslate, T("menu_translate"))
	}
	appendMenu(menu, win.MF_SEPARATOR, 0, "")

	appendMenu(menu, win.MF_STRING, idSettings, T("menu_settings"))
	appendMenu(menu, win.MF_STRING, idStartPage, T("menu_start"))

	appendMenu(menu, win.MF_STRING, idZoomReset, T("menu_zoom_reset"))

	pt := win.POINT{X: x, Y: y}
	win.ClientToScreen(hwnd, &pt)
	win.SetForegroundWindow(hwnd)
	cmd := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_RIGHTBUTTON, pt.X, pt.Y, 0, hwnd, nil)
	win.PostMessage(hwnd, win.WM_NULL, 0, 0)

	app.hovered, app.hoverTab = zoneNone, -1
	invalidateBar()

	switch {
	case cmd == idNewTab:
		app.tabs.AddTab(newTabURL(), app.tabs.Active().profile)
		app.layout()
	case cmd == idCloseTab:
		app.tabs.Close(tabIdx)
		app.layout()
	case cmd == idProfileAdd:
		if name, ok := showInputBox(hwnd, T("ib_title"), T("ib_prompt"), ""); ok {
			name = strings.TrimSpace(name)
			if name == "" {
				break
			}
			if key := validateNewProfile(name); key != "" {
				win.MessageBox(hwnd, mustUTF16(T(key, name)), mustUTF16("xbrowser"), win.MB_ICONINFORMATION)
				break
			}
			addProfile(name)
			app.tabs.AddTab(newTabURL(), name)
			app.layout()
		}
	case cmd >= idProfileNewBase && cmd < idProfileNewBase+100:
		if i := int(cmd - idProfileNewBase); i < len(profiles) {
			p := profiles[i]
			if p == "Default" {
				p = ""
			}
			app.tabs.AddTab(newTabURL(), p)
			app.layout()
		}
	case cmd >= idProfileSwitchBase && cmd < idProfileSwitchBase+100:
		if i := int(cmd - idProfileSwitchBase); i < len(profiles) && tabIdx >= 0 {
			p := profiles[i]
			if p == "Default" {
				p = ""
			}
			app.tabs.SwitchTabProfile(tabIdx, p)
			app.layout()
		}
	case cmd == idTranslate:
		translateText(app.tabs.Active().selection)
	case cmd == idZoomReset:
		app.tabs.ZoomReset()
	case cmd == idSettings:
		app.tabs.OpenSettings()
		app.layout()
	case cmd == idStartPage:
		app.tabs.AddTab(startURL, app.tabs.Active().profile)
		app.layout()
	}
}
