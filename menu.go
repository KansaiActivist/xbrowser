//go:build windows

package main

import (
	"strings"
	"unsafe"

	"github.com/lxn/win"
)

var (
	pAppendMenuW = user32b.NewProc("AppendMenuW")
	menuItems    []*menuItem
)

type menuItem struct {
	text    string
	icon    string
	sep     bool
	popup   bool
	checked bool
	dot     win.COLORREF
	hasDot  bool
}

func appendItem(m win.HMENU, flags uint32, id uintptr, it *menuItem) {
	menuItems = append(menuItems, it)
	pAppendMenuW.Call(uintptr(m), uintptr(flags|win.MF_OWNERDRAW), id, uintptr(len(menuItems)))
}

func appendSeparator(m win.HMENU) {
	appendItem(m, win.MF_DISABLED, 0, &menuItem{sep: true})
}

func menuItemFromData(data uintptr) *menuItem {
	if data == 0 || int(data) > len(menuItems) {
		return nil
	}
	return menuItems[data-1]
}

type menuColors struct{ bg, text, sub, hover, sep win.COLORREF }

func currentMenuColors() menuColors {
	if effectivePreset() == "light" {
		return menuColors{rgb(0xFF, 0xFF, 0xFF), rgb(0x0F, 0x14, 0x19), rgb(0x53, 0x64, 0x71), rgb(0xF7, 0xF9, 0xF9), rgb(0xEF, 0xF3, 0xF4)}
	}
	return menuColors{rgb(0x00, 0x00, 0x00), rgb(0xE7, 0xE9, 0xEA), rgb(0x71, 0x76, 0x7B), rgb(0x16, 0x18, 0x1C), rgb(0x2F, 0x33, 0x36)}
}

func menuFont() win.HFONT { return makeFont("Segoe UI", -sc(15), 600) }

func setMenuBackground(m win.HMENU, brush win.HBRUSH) {
	mi := win.MENUINFO{FMask: 0x00000002 | 0x80000000, HbrBack: brush}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	win.SetMenuInfo(m, &mi)
}

func measureMenuItem(hwnd win.HWND, mis *win.MEASUREITEMSTRUCT) {
	it := menuItemFromData(mis.ItemData)
	if it == nil {
		return
	}
	if it.sep {
		mis.ItemWidth, mis.ItemHeight = 1, uint32(sc(9))
		return
	}
	hdc := win.GetDC(hwnd)
	font := menuFont()
	old := win.SelectObject(hdc, win.HGDIOBJ(font))
	u, n := utf16(it.text)
	var size win.SIZE
	win.GetTextExtentPoint32(hdc, u, n, &size)
	win.SelectObject(hdc, old)
	win.DeleteObject(win.HGDIOBJ(font))
	win.ReleaseDC(hwnd, hdc)
	w := size.CX + sc(16) + sc(36) + sc(40)
	if w < sc(220) {
		w = sc(220)
	}
	mis.ItemWidth, mis.ItemHeight = uint32(w), uint32(sc(40))
}

func drawMenuItem(dis *win.DRAWITEMSTRUCT) {
	it := menuItemFromData(dis.ItemData)
	if it == nil {
		return
	}
	col := currentMenuColors()
	p := &painter{
		hdc:   dis.HDC,
		fText: menuFont(),
		fIcon: makeFont(iconFace, -sc(16), win.FW_NORMAL),
	}
	defer win.DeleteObject(win.HGDIOBJ(p.fText))
	defer win.DeleteObject(win.HGDIOBJ(p.fIcon))
	win.SetBkMode(dis.HDC, win.TRANSPARENT)
	r := dis.RcItem
	p.fill(r, col.bg)
	if it.sep {
		line := win.RECT{Left: r.Left + sc(12), Top: r.Top + (r.Bottom-r.Top)/2, Right: r.Right - sc(12), Bottom: r.Top + (r.Bottom-r.Top)/2 + 1}
		p.fill(line, col.sep)
		return
	}
	if dis.ItemState&win.ODS_SELECTED != 0 {
		p.round(inset(r, sc(5), sc(2)), sc(8), col.hover)
	}
	iconBox := win.RECT{Left: r.Left + sc(14), Top: r.Top, Right: r.Left + sc(14) + sc(24), Bottom: r.Bottom}
	switch {
	case it.checked:
		p.icon(iconBox, "\uE73E", col.text, false)
	case it.hasDot:
		d := sc(10)
		top := r.Top + (r.Bottom-r.Top-d)/2
		p.round(win.RECT{Left: iconBox.Left + sc(7), Top: top, Right: iconBox.Left + sc(7) + d, Bottom: top + d}, d/2, it.dot)
	case it.icon != "":
		p.icon(iconBox, it.icon, col.text, false)
	}
	textR := win.RECT{Left: iconBox.Right + sc(10), Top: r.Top, Right: r.Right - sc(36), Bottom: r.Bottom}
	p.text(textR, it.text, col.text, p.fText, win.DT_LEFT|win.DT_END_ELLIPSIS)
	if it.popup {
		p.icon(win.RECT{Left: r.Right - sc(34), Top: r.Top, Right: r.Right - sc(10), Bottom: r.Bottom}, "\uE76C", col.sub, true)
	}
}

const odtMenu = 1

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

	var menus []win.HMENU
	newMenu := func() win.HMENU {
		mm := win.CreatePopupMenu()
		menus = append(menus, mm)
		return mm
	}
	menu := newMenu()
	brush := createSolidBrush(currentMenuColors().bg)
	menuItems = nil
	defer func() {
		for _, mm := range menus {
			win.DestroyMenu(mm)
		}
		win.DeleteObject(win.HGDIOBJ(brush))
		menuItems = nil
	}()

	appendItem(menu, win.MF_STRING, idNewTab, &menuItem{text: T("menu_new_tab"), icon: "\uE710"})

	profiles := allProfiles()
	newTabProfiles := newMenu()
	for i, p := range profiles {
		it := &menuItem{text: profileMenuLabel(p)}
		it.dot, it.hasDot = profileColor(p)
		appendItem(newTabProfiles, win.MF_STRING, uintptr(idProfileNewBase+i), it)
	}
	appendSeparator(newTabProfiles)
	appendItem(newTabProfiles, win.MF_STRING, idProfileAdd, &menuItem{text: T("menu_profile_add"), icon: "\uE710"})
	appendItem(menu, win.MF_POPUP, uintptr(newTabProfiles), &menuItem{text: T("menu_profile_new"), icon: "\uE77B", popup: true})

	if zone == zoneTab || zone == zoneTabClose {
		appendItem(menu, win.MF_STRING, idCloseTab, &menuItem{text: T("menu_close_tab"), icon: "\uE711"})
	}
	if (zone == zoneTab || zone == zoneTabClose) && app.tabs.tabs[tabIdx].kind != kindSettings {
		switchMenu := newMenu()
		curID := profileID(app.tabs.tabs[tabIdx].profile)
		for i, p := range profiles {
			it := &menuItem{text: profileMenuLabel(p), checked: strings.EqualFold(p, curID)}
			it.dot, it.hasDot = profileColor(p)
			appendItem(switchMenu, win.MF_STRING, uintptr(idProfileSwitchBase+i), it)
		}
		appendItem(menu, win.MF_POPUP, uintptr(switchMenu), &menuItem{text: T("menu_tab_profile", profileMenuLabel(curID)), icon: "\uE77B", popup: true})
	}
	if t := app.tabs.Active(); t.kind == kindWeb && t.selection != "" {
		appendItem(menu, win.MF_STRING, idTranslate, &menuItem{text: T("menu_translate"), icon: "\uE774"})
	}
	appendSeparator(menu)

	appendItem(menu, win.MF_STRING, idSettings, &menuItem{text: T("menu_settings"), icon: "\uE713"})
	appendItem(menu, win.MF_STRING, idStartPage, &menuItem{text: T("menu_start"), icon: "\uE80F"})
	appendItem(menu, win.MF_STRING, idZoomReset, &menuItem{text: T("menu_zoom_reset")})
	for _, mm := range menus {
		setMenuBackground(mm, brush)
	}

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
		app.tabs.Active().translateSelection()
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
