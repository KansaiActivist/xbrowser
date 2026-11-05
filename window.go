//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const (
	titleBarH     = 46
	toolbarH      = 44
	chromeH       = titleBarH + toolbarH
	themeBtnW     = 38
	dragThreshold = 6
	navBtnW       = 38
	tabMinW       = 96
	tabMaxW       = 240
	tabGap        = 4
	newTabBtnW    = 38
	winBtnW       = 48
	zoomBtnW      = 30
	zoomLabelW    = 56
	zoomClusterW  = zoomBtnW*2 + zoomLabelW
	settingsBtnW  = 38
	resizeGrip    = 6

	sizeMinimized    = 1
	smCXPaddedBorder = 92
)

type ncCalcSizeParams struct {
	Rgrc  [3]win.RECT
	Lppos uintptr
}

type hitZone int

const (
	zoneNone hitZone = iota
	zoneBack
	zoneForward
	zoneReload
	zoneNewTab
	zoneMinimize
	zoneMaximize
	zoneClose
	zoneTab
	zoneTabClose
	zoneZoomOut
	zoneZoomLabel
	zoneZoomIn
	zoneSettings
	zoneTheme
	zoneAddress
)

type ChromeWindow struct {
	hwnd     win.HWND
	tabs     *TabManager
	hovered  hitZone
	hoverTab int
	tracking bool
	dragIdx  int
	dragX0   int32
	dragging bool
}

var app *ChromeWindow

func initChromeWindow(hwnd win.HWND) {
	app = &ChromeWindow{hwnd: hwnd, hoverTab: -1, dragIdx: -1}
	createAddressBar(hwnd)
	app.tabs = newTabManager(hwnd)

	app.layout()
	app.tabs.AddTab(newTabURL(), "")
	app.layout()
}

func (c *ChromeWindow) contentRect() win.RECT {
	var rc win.RECT
	win.GetClientRect(c.hwnd, &rc)
	rc.Top = chromeH
	return rc
}

func (c *ChromeWindow) layout() {
	if c.tabs != nil {
		c.tabs.Resize(c.contentRect())
	}
	layoutAddressBar()
	win.InvalidateRect(c.hwnd, nil, false)
}

func (c *ChromeWindow) tabRects() []win.RECT {
	var client win.RECT
	win.GetClientRect(c.hwnd, &client)
	width := int32(client.Right - client.Left)

	left := int32(8)
	right := width - int32(winBtnW*3+14)
	avail := right - left - newTabBtnW
	if avail < 0 {
		avail = 0
	}

	n := int32(len(c.tabs.tabs))
	if n == 0 {
		return nil
	}
	tw := int32(tabMaxW)
	if avail/n < int32(tabMaxW) {
		tw = avail / n
	}
	if tw < int32(tabMinW) {
		tw = int32(tabMinW)
	}

	rects := make([]win.RECT, n)
	x := left
	for i := int32(0); i < n; i++ {
		rects[i] = win.RECT{Left: x, Top: 0, Right: x + tw - tabGap, Bottom: titleBarH}
		x += tw
	}
	return rects
}

func (c *ChromeWindow) newTabBtnRect() win.RECT {
	rects := c.tabRects()
	x := int32(8)
	if len(rects) > 0 {
		x = rects[len(rects)-1].Right + tabGap
	}
	return win.RECT{Left: x, Top: 0, Right: x + newTabBtnW, Bottom: titleBarH}
}

func (c *ChromeWindow) winButtonRect(which hitZone) win.RECT {
	var client win.RECT
	win.GetClientRect(c.hwnd, &client)
	width := int32(client.Right - client.Left)
	switch which {
	case zoneClose:
		return win.RECT{Left: width - winBtnW, Top: 0, Right: width, Bottom: titleBarH}
	case zoneMaximize:
		return win.RECT{Left: width - winBtnW*2, Top: 0, Right: width - winBtnW, Bottom: titleBarH}
	case zoneMinimize:
		return win.RECT{Left: width - winBtnW*3, Top: 0, Right: width - winBtnW*2, Bottom: titleBarH}
	}
	return win.RECT{}
}

func (c *ChromeWindow) navButtonRect(which hitZone) win.RECT {
	switch which {
	case zoneBack:
		return win.RECT{Left: 6, Top: titleBarH, Right: 6 + navBtnW, Bottom: chromeH}
	case zoneForward:
		return win.RECT{Left: 6 + navBtnW, Top: titleBarH, Right: 6 + navBtnW*2, Bottom: chromeH}
	case zoneReload:
		return win.RECT{Left: 6 + navBtnW*2, Top: titleBarH, Right: 6 + navBtnW*3, Bottom: chromeH}
	}
	return win.RECT{}
}

func tabVisualRect(slot win.RECT) win.RECT {
	return win.RECT{Left: slot.Left, Top: slot.Top + 8, Right: slot.Right, Bottom: slot.Bottom - 4}
}

func tabCloseRect(slot win.RECT) win.RECT {
	vr := tabVisualRect(slot)
	const sz = 24
	top := vr.Top + (vr.Bottom-vr.Top-sz)/2
	return win.RECT{Left: vr.Right - 8 - sz, Top: top, Right: vr.Right - 8, Bottom: top + sz}
}

func (c *ChromeWindow) clientWidth() int32 {
	var client win.RECT
	win.GetClientRect(c.hwnd, &client)
	return client.Right - client.Left
}

func (c *ChromeWindow) zoomRect(which hitZone) win.RECT {
	base := c.clientWidth() - 8 - int32(settingsBtnW+themeBtnW+zoomClusterW)
	top, bottom := int32(titleBarH+7), int32(chromeH-7)
	switch which {
	case zoneZoomOut:
		return win.RECT{Left: base, Top: top, Right: base + zoomBtnW, Bottom: bottom}
	case zoneZoomLabel:
		return win.RECT{Left: base + zoomBtnW, Top: top, Right: base + zoomBtnW + zoomLabelW, Bottom: bottom}
	case zoneZoomIn:
		return win.RECT{Left: base + zoomBtnW + zoomLabelW, Top: top, Right: base + zoomClusterW, Bottom: bottom}
	}
	return win.RECT{}
}

func (c *ChromeWindow) themeRect() win.RECT {
	left := c.clientWidth() - 8 - int32(settingsBtnW+themeBtnW)
	return win.RECT{Left: left, Top: titleBarH, Right: left + themeBtnW, Bottom: chromeH}
}

func (c *ChromeWindow) settingsRect() win.RECT {
	left := c.clientWidth() - 8 - int32(settingsBtnW)
	return win.RECT{Left: left, Top: titleBarH, Right: left + settingsBtnW, Bottom: chromeH}
}

func (c *ChromeWindow) addressRect() win.RECT {
	left := int32(6+navBtnW*3) + 4
	right := c.zoomRect(zoneZoomOut).Left - 8
	if right < left+120 {
		right = left + 120
	}
	return win.RECT{Left: left, Top: titleBarH + 6, Right: right, Bottom: chromeH - 6}
}

func (c *ChromeWindow) tabIndexAt(x int32) int {
	rects := c.tabRects()
	if len(rects) == 0 {
		return -1
	}
	slot := rects[0].Right - rects[0].Left + tabGap
	i := int((x - rects[0].Left) / slot)
	if x < rects[0].Left {
		i = 0
	}
	if i >= len(rects) {
		i = len(rects) - 1
	}
	return i
}

func invalidateBar() {
	r := win.RECT{Left: 0, Top: 0, Right: 32000, Bottom: chromeH}
	win.InvalidateRect(app.hwnd, &r, false)
}

func ptInRect(r win.RECT, x, y int32) bool {
	return x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom
}

func (c *ChromeWindow) hitTest(x, y int32) (hitZone, int) {
	if y >= chromeH {
		return zoneNone, -1
	}
	if ptInRect(c.navButtonRect(zoneBack), x, y) {
		return zoneBack, -1
	}
	if ptInRect(c.navButtonRect(zoneForward), x, y) {
		return zoneForward, -1
	}
	if ptInRect(c.navButtonRect(zoneReload), x, y) {
		return zoneReload, -1
	}
	if ptInRect(c.newTabBtnRect(), x, y) {
		return zoneNewTab, -1
	}
	for _, z := range []hitZone{zoneZoomOut, zoneZoomLabel, zoneZoomIn} {
		if ptInRect(c.zoomRect(z), x, y) {
			return z, -1
		}
	}
	if ptInRect(c.settingsRect(), x, y) {
		return zoneSettings, -1
	}
	if ptInRect(c.themeRect(), x, y) {
		return zoneTheme, -1
	}
	if ptInRect(c.addressRect(), x, y) {
		return zoneAddress, -1
	}
	if ptInRect(c.winButtonRect(zoneMinimize), x, y) {
		return zoneMinimize, -1
	}
	if ptInRect(c.winButtonRect(zoneMaximize), x, y) {
		return zoneMaximize, -1
	}
	if ptInRect(c.winButtonRect(zoneClose), x, y) {
		return zoneClose, -1
	}
	for i, r := range c.tabRects() {
		if ptInRect(r, x, y) {
			if ptInRect(tabCloseRect(r), x, y) {
				return zoneTabClose, i
			}
			return zoneTab, i
		}
	}
	return zoneNone, -1
}

func isMaximized(hwnd win.HWND) bool {
	return win.IsZoomed(hwnd)
}

func (c *ChromeWindow) toggleMaximize() {
	if isMaximized(c.hwnd) {
		win.ShowWindow(c.hwnd, win.SW_RESTORE)
	} else {
		win.ShowWindow(c.hwnd, win.SW_MAXIMIZE)
	}
}

func wndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {

	switch msg {
	case win.WM_NCCALCSIZE:
		if wParam != 0 && isMaximized(hwnd) {
			maximizedClientRect(hwnd, lParam)
		}
		return 0
	case win.WM_GETMINMAXINFO:
		mmi := (*win.MINMAXINFO)(unsafe.Pointer(lParam))
		mmi.PtMinTrackSize.X = 480
		mmi.PtMinTrackSize.Y = 360

		if mon := win.MonitorFromWindow(hwnd, win.MONITOR_DEFAULTTONEAREST); mon != 0 {
			var mi win.MONITORINFO
			mi.CbSize = uint32(unsafe.Sizeof(mi))
			if win.GetMonitorInfo(mon, &mi) {
				work, monRc := mi.RcWork, mi.RcMonitor
				mmi.PtMaxPosition.X = work.Left - monRc.Left
				mmi.PtMaxPosition.Y = work.Top - monRc.Top
				mmi.PtMaxSize.X = work.Right - work.Left
				mmi.PtMaxSize.Y = work.Bottom - work.Top
			}
		}
		return 0
	case win.WM_NCHITTEST:
		return ncHitTest(hwnd, lParam)
	case win.WM_NCACTIVATE:

		return win.DefWindowProc(hwnd, msg, wParam, ^uintptr(0))
	case win.WM_NCPAINT:
		return 0
	case win.WM_ERASEBKGND:

		var rc win.RECT
		win.GetClientRect(hwnd, &rc)
		brush := createSolidBrush(palette.bar)
		winFillRect(win.HDC(wParam), &rc, brush)
		win.DeleteObject(win.HGDIOBJ(brush))
		return 1
	}

	if app == nil || app.hwnd != hwnd {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case win.WM_ACTIVATE, win.WM_ACTIVATEAPP, win.WM_SETFOCUS, win.WM_DISPLAYCHANGE, win.WM_WINDOWPOSCHANGED:

		r := win.DefWindowProc(hwnd, msg, wParam, lParam)
		repaintBar()
		return r

	case wmAppAction:

		switch wParam {
		case actOpenSettings:
			app.tabs.OpenSettings()
		case actOpenProfile:
			app.tabs.AddTab(newTabURL(), pendingProfile)
		case actPickImage:
			pickBackgroundImage()
		case actFocusAddress:
			focusAddress()
		case actToggleTheme:
			toggleTheme()
		}
		app.layout()
		return 0

	case win.WM_RBUTTONUP:
		x := int32(int16(lParam & 0xffff))
		y := int32(int16((lParam >> 16) & 0xffff))
		if y < chromeH {
			showContextMenu(hwnd, x, y)
		}
		return 0

	case win.WM_SIZE:
		if wParam == sizeMinimized {
			return 0
		}
		app.layout()
		return 0

	case win.WM_MOVE, win.WM_SHOWWINDOW:
		if app.tabs != nil {
			app.tabs.Refresh()
		}
		return win.DefWindowProc(hwnd, msg, wParam, lParam)

	case win.WM_PAINT:
		paintChrome(hwnd)
		return 0

	case win.WM_LBUTTONDOWN:
		x := int32(int16(lParam & 0xffff))
		y := int32(int16((lParam >> 16) & 0xffff))
		zone, idx := app.hitTest(x, y)
		switch zone {
		case zoneBack:
			app.tabs.Active().Eval("history.back()")
		case zoneForward:
			app.tabs.Active().Eval("history.forward()")
		case zoneReload:
			app.tabs.Active().Eval("location.reload()")
		case zoneNewTab:
			app.tabs.AddTab(newTabURL(), app.tabs.Active().profile)
			app.layout()
		case zoneSettings:
			app.tabs.OpenSettings()
			app.layout()
		case zoneTab:
			app.tabs.Switch(idx)
			app.layout()
			app.dragIdx, app.dragX0, app.dragging = idx, x, false
			win.SetCapture(hwnd)
		case zoneTheme:
			toggleTheme()
		case zoneAddress:
			focusAddress()
		case zoneTabClose:
			app.tabs.Close(idx)
			app.layout()
		case zoneZoomOut:
			app.tabs.ZoomStep(-1)
		case zoneZoomIn:
			app.tabs.ZoomStep(1)
		case zoneZoomLabel:
			app.tabs.ZoomReset()
		case zoneMinimize:
			win.ShowWindow(hwnd, win.SW_MINIMIZE)
		case zoneMaximize:
			app.toggleMaximize()
		case zoneClose:
			win.DestroyWindow(hwnd)
		case zoneNone:
			if y < titleBarH {

				win.ReleaseCapture()
				win.PostMessage(hwnd, win.WM_NCLBUTTONDOWN, uintptr(win.HTCAPTION), 0)
			}
		}
		return 0

	case win.WM_LBUTTONUP, win.WM_CAPTURECHANGED:
		if app.dragIdx >= 0 {
			app.dragIdx, app.dragging = -1, false
			if msg == win.WM_LBUTTONUP {
				win.ReleaseCapture()
			}
			invalidateBar()
		}
		return 0

	case win.WM_CTLCOLOREDIT:
		if win.HWND(lParam) == addrEdit {
			return addrCtlColor(win.HDC(wParam))
		}

	case win.WM_COMMAND:
		if win.HWND(lParam) == addrEdit {
			switch (wParam >> 16) & 0xffff {
			case enSetFocus:
				win.InvalidateRect(addrEdit, nil, true)
				invalidateBar()
			case enKillFocus:
				refreshAddress(true)
				win.InvalidateRect(addrEdit, nil, true)
				invalidateBar()
			}
		}
		return 0

	case win.WM_SETTINGCHANGE:
		systemThemeChanged()

	case win.WM_LBUTTONDBLCLK:
		x := int32(int16(lParam & 0xffff))
		y := int32(int16((lParam >> 16) & 0xffff))
		zone, _ := app.hitTest(x, y)
		if zone == zoneNone && y < titleBarH {
			app.toggleMaximize()
		}
		return 0

	case win.WM_MOUSEMOVE:
		x := int32(int16(lParam & 0xffff))
		y := int32(int16((lParam >> 16) & 0xffff))
		if app.dragIdx >= 0 && wParam&win.MK_LBUTTON != 0 {
			d := x - app.dragX0
			if d < 0 {
				d = -d
			}
			if app.dragging || d >= dragThreshold {
				app.dragging = true
				if to := app.tabIndexAt(x); to >= 0 && to != app.dragIdx {
					app.tabs.MoveTab(app.dragIdx, to)
					app.dragIdx = to
				}
				app.hovered, app.hoverTab = zoneTab, app.dragIdx
				invalidateBar()
				return 0
			}
		}
		zone, idx := app.hitTest(x, y)
		if zone != app.hovered || idx != app.hoverTab {
			app.hovered = zone
			app.hoverTab = idx
			win.InvalidateRect(hwnd, nil, false)
		}
		if !app.tracking {
			app.tracking = true
			tme := win.TRACKMOUSEEVENT{
				CbSize:    uint32(unsafe.Sizeof(win.TRACKMOUSEEVENT{})),
				DwFlags:   win.TME_LEAVE,
				HwndTrack: hwnd,
			}
			win.TrackMouseEvent(&tme)
		}
		return 0

	case win.WM_MOUSELEAVE:
		app.tracking = false
		app.hovered = zoneNone
		app.hoverTab = -1
		win.InvalidateRect(hwnd, nil, false)
		return 0

	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0

	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func ncHitTest(hwnd win.HWND, lParam uintptr) uintptr {
	var rc win.RECT
	win.GetWindowRect(hwnd, &rc)
	x := int32(int16(lParam & 0xffff))
	y := int32(int16((lParam >> 16) & 0xffff))

	left := x < rc.Left+resizeGrip
	right := x >= rc.Right-resizeGrip
	top := y < rc.Top+resizeGrip
	bottom := y >= rc.Bottom-resizeGrip

	switch {
	case top && left:
		return uintptr(win.HTTOPLEFT)
	case top && right:
		return uintptr(win.HTTOPRIGHT)
	case bottom && left:
		return uintptr(win.HTBOTTOMLEFT)
	case bottom && right:
		return uintptr(win.HTBOTTOMRIGHT)
	case left:
		return uintptr(win.HTLEFT)
	case right:
		return uintptr(win.HTRIGHT)
	case top:
		return uintptr(win.HTTOP)
	case bottom:
		return uintptr(win.HTBOTTOM)
	}
	return uintptr(win.HTCLIENT)
}

// 枠なしウィンドウの最大化でタスクバーを隠さないよう、クライアント領域をワークエリアに合わせる
func maximizedClientRect(hwnd win.HWND, lParam uintptr) {
	mon := win.MonitorFromWindow(hwnd, win.MONITOR_DEFAULTTONEAREST)
	if mon == 0 {
		return
	}
	var mi win.MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if !win.GetMonitorInfo(mon, &mi) {
		return
	}
	params := (*ncCalcSizeParams)(unsafe.Pointer(lParam))
	params.Rgrc[0] = mi.RcWork
}

func repaintBar() {
	r := win.RECT{Left: 0, Top: 0, Right: 32000, Bottom: chromeH}
	win.RedrawWindow(app.hwnd, &r, 0, win.RDW_INVALIDATE|win.RDW_UPDATENOW)
}

func mustUTF16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}
