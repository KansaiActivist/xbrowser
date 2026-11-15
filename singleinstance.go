//go:build windows

package main

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

const errorAlreadyExists = 183

var (
	kernel32b        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32b.NewProc("CreateMutexW")
)

func ensureSingleInstance(forwardURL string) bool {
	name, _ := syscall.UTF16PtrFromString("Local\\XBrowserSingleInstanceMutex")
	h, _, callErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {

		return true
	}
	if errno, ok := callErr.(syscall.Errno); ok && errno == errorAlreadyExists {
		classPtr, _ := syscall.UTF16PtrFromString(className)
		var existing win.HWND
		for i := 0; i < 50 && existing == 0; i++ {
			if existing = win.FindWindow(classPtr, nil); existing == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
		if existing != 0 {
			if forwardURL != "" {
				sendURLToRunning(existing, forwardURL)
			}
			bringToFront(existing)
		}
		return false
	}
	return true
}
