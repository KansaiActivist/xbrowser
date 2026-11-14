//go:build windows

package main

import (
	"net/url"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	slotAddNavigationStarting    = 7
	slotRemoveNavigationStarting = 8
	slotAddContentLoading        = 9
	slotRemoveContentLoading     = 10
	slotAddNewWindowRequested    = 44
	slotRemoveNewWindowRequested = 45

	slotNavArgsURI      = 3
	slotNavArgsCancel   = 8
	slotNewWinArgsURI   = 3
	slotNewWinHandled   = 6
	slotNewWinUserInit  = 8
	slotControllerClose = 24
)

type comVtbl struct{ queryInterface, addRef, release, invoke uintptr }

type comObject struct {
	vtbl *comVtbl
	fn   func(sender, args uintptr) uintptr
}

type comHook struct {
	removeSlot int
	token      int64
	object     uintptr
}

var (
	sharedVtbl *comVtbl
	comObjects = map[uintptr]*comObject{}
)

func comVtblInit() *comVtbl {
	if sharedVtbl != nil {
		return sharedVtbl
	}
	ref := syscall.NewCallback(func(this uintptr) uintptr { return 1 })
	sharedVtbl = &comVtbl{
		queryInterface: syscall.NewCallback(func(this, riid, ppv uintptr) uintptr {
			if ppv == 0 {
				return 0x80004003
			}
			*(*uintptr)(unsafe.Pointer(ppv)) = this
			return 0
		}),
		addRef:  ref,
		release: ref,
		invoke: syscall.NewCallback(func(this, sender, args uintptr) (ret uintptr) {
			defer func() {
				if r := recover(); r != nil {
					debugLog("COMイベント処理で例外 err=%v", r)
					ret = 0
				}
			}()
			if o := comObjects[this]; o != nil && o.fn != nil {
				return o.fn(sender, args)
			}
			return 0
		}),
	}
	return sharedVtbl
}

func comMethod(obj uintptr, slot int) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	return *(*uintptr)(unsafe.Pointer(vtbl + uintptr(slot)*unsafe.Sizeof(uintptr(0))))
}

func comCall(obj uintptr, slot int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(comMethod(obj, slot), append([]uintptr{obj}, args...)...)
	return r
}

func comAddHook(wv uintptr, addSlot, removeSlot int, fn func(sender, args uintptr) uintptr) (comHook, bool) {
	o := &comObject{vtbl: comVtblInit(), fn: fn}
	addr := uintptr(unsafe.Pointer(o))
	comObjects[addr] = o
	var token int64
	if hr := comCall(wv, addSlot, addr, uintptr(unsafe.Pointer(&token))); hr != 0 {
		delete(comObjects, addr)
		return comHook{}, false
	}
	return comHook{removeSlot: removeSlot, token: token, object: addr}, true
}

func comRemoveHook(wv uintptr, h comHook) {
	if h.object == 0 {
		return
	}
	comCall(wv, h.removeSlot, uintptr(h.token))
	delete(comObjects, h.object)
}

func comReadString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for n < maxExternalURLLen*2 && *(*uint16)(unsafe.Pointer(p + uintptr(n)*2)) != 0 {
		n++
	}
	s := windows.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(p)), n))
	windows.CoTaskMemFree(unsafe.Pointer(p))
	return s
}

func comArgsURI(args uintptr, slot int) string {
	var p uintptr
	if hr := comCall(args, slot, uintptr(unsafe.Pointer(&p))); hr != 0 {
		return ""
	}
	return comReadString(p)
}

type navAction int

const (
	navAllow navAction = iota
	navExternal
	navBlock
)

func navDecision(raw string) navAction {
	u, err := url.Parse(raw)
	if err != nil {
		return navBlock
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http":
		if isXHost(u.Hostname()) {
			return navAllow
		}
		return navExternal
	case "about", "edge", "chrome-error":
		return navAllow
	}
	return navBlock
}

func (t *Tab) installHooks() {
	wv := uintptr(unsafe.Pointer(t.chromium.GetICoreWebView2_3()))
	if wv == 0 {
		debugLog("WebView2のイベントフックを設定できません(JSによる検知のみで動作します)")
		return
	}
	t.wv = wv
	add := func(addSlot, removeSlot int, fn func(sender, args uintptr) uintptr) {
		if h, ok := comAddHook(wv, addSlot, removeSlot, fn); ok {
			t.hooks = append(t.hooks, h)
		} else {
			debugLog("イベントフックの登録に失敗 slot=%d", addSlot)
		}
	}
	add(slotAddNavigationStarting, slotRemoveNavigationStarting, func(_, args uintptr) uintptr { return t.onNavigationStarting(args) })
	add(slotAddNewWindowRequested, slotRemoveNewWindowRequested, func(_, args uintptr) uintptr { return t.onNewWindowRequested(args) })
	add(slotAddContentLoading, slotRemoveContentLoading, func(_, _ uintptr) uintptr {
		applyZoomTo(t.chromium)
		return 0
	})
}

func (t *Tab) removeHooks() {
	if t.wv != 0 {
		for _, h := range t.hooks {
			comRemoveHook(t.wv, h)
		}
	}
	t.hooks, t.wv = nil, 0
}

func (t *Tab) onNavigationStarting(args uintptr) uintptr {
	if t.kind != kindWeb {
		return 0
	}
	uri := comArgsURI(args, slotNavArgsURI)
	switch navDecision(uri) {
	case navAllow:
		return 0
	case navExternal:
		openExternal(uri)
	}
	comCall(args, slotNavArgsCancel, 1)
	return 0
}

func (t *Tab) onNewWindowRequested(args uintptr) uintptr {
	comCall(args, slotNewWinHandled, 1)
	var userInit int32
	comCall(args, slotNewWinUserInit, uintptr(unsafe.Pointer(&userInit)))
	uri := comArgsURI(args, slotNewWinArgsURI)
	if t.kind != kindWeb || userInit == 0 {
		return 0
	}
	switch navDecision(uri) {
	case navAllow:
		if target, ok := normalizeXURL(uri); ok && time.Since(t.lastOpenTab) > 300*time.Millisecond {
			t.lastOpenTab = time.Now()
			queueOpenTab(target, t.profile)
		}
	case navExternal:
		openExternal(uri)
	}
	return 0
}

func closeController(t *Tab) {
	if t.chromium == nil {
		return
	}
	if c := t.chromium.GetController(); c != nil {
		comCall(uintptr(unsafe.Pointer(c)), slotControllerClose)
	}
}
