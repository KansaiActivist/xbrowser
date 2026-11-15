//go:build windows

package main

import (
	"time"
	"unsafe"

	"github.com/lxn/win"
)

const (
	slotTrySuspend = 68
	slotResume     = 69

	trimTimerID    = 0x5842
	trimTimerEvery = 60 * 1000
	suspendAfter   = 3 * time.Minute
)

var suspendHandler uintptr

func comNoopHandler() uintptr {
	if suspendHandler == 0 {
		o := &comObject{vtbl: comVtblInit(), fn: func(_, _ uintptr) uintptr { return 0 }}
		suspendHandler = uintptr(unsafe.Pointer(o))
		comObjects[suspendHandler] = o
	}
	return suspendHandler
}

func startTrimTimer(hwnd win.HWND) {
	win.SetTimer(hwnd, trimTimerID, trimTimerEvery, 0)
}

func stopTrimTimer(hwnd win.HWND) {
	win.KillTimer(hwnd, trimTimerID)
}

func (t *Tab) suspend() {
	if t.wv == 0 || t.suspended || t.kind != kindWeb {
		return
	}
	if hr := comCall(t.wv, slotTrySuspend, comNoopHandler()); hr == 0 {
		t.suspended = true
		debugLog("非表示タブを一時停止 title=%s", t.title)
	}
}

func (t *Tab) resume() {
	t.hiddenAt = time.Time{}
	if t.wv == 0 || !t.suspended {
		return
	}
	t.suspended = false
	comCall(t.wv, slotResume)
}

func (m *TabManager) trimHidden() {
	now := time.Now()
	for i, t := range m.tabs {
		if i == m.active || !t.ready || t.closed || t.suspended {
			continue
		}
		if t.hiddenAt.IsZero() {
			t.hiddenAt = now
			continue
		}
		if now.Sub(t.hiddenAt) >= suspendAfter {
			t.suspend()
		}
	}
}
