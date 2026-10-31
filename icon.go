//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const rtGroupIcon = 14

var procEnumResourceNamesW = syscall.NewLazyDLL("kernel32.dll").NewProc("EnumResourceNamesW")

type iconName struct {
	id   uintptr
	name string
}

// go-winres は "APP" という名前、rsrc などは整数IDでアイコンを埋め込むため、両方に対応する
func embeddedIconNames(hInstance win.HINSTANCE) []iconName {
	var found []iconName
	cb := syscall.NewCallback(func(hModule, lpType, lpName, lParam uintptr) uintptr {
		if lpName>>16 == 0 {
			found = append(found, iconName{id: lpName})
		} else {
			var u []uint16
			for i := 0; i < 256; i++ {
				c := *(*uint16)(unsafe.Pointer(lpName + uintptr(i)*2))
				if c == 0 {
					break
				}
				u = append(u, c)
			}
			found = append(found, iconName{name: syscall.UTF16ToString(u)})
		}
		return 1
	})
	procEnumResourceNamesW.Call(uintptr(hInstance), rtGroupIcon, cb, 0)
	return found
}

func iconPtr(n iconName) *uint16 {
	if n.name != "" {
		p, _ := syscall.UTF16PtrFromString(n.name)
		return p
	}
	return win.MAKEINTRESOURCE(n.id)
}

func loadAppIcons(hInstance win.HINSTANCE) (big win.HICON, small win.HICON) {
	names := embeddedIconNames(hInstance)
	ordered := make([]iconName, 0, len(names))
	for _, want := range []func(iconName) bool{
		func(n iconName) bool { return n.id == 1 },
		func(n iconName) bool { return n.name == "APP" },
		func(n iconName) bool { return true },
	} {
		for _, n := range names {
			if want(n) {
				ordered = append(ordered, n)
			}
		}
	}

	for _, n := range ordered {
		p := iconPtr(n)
		big = win.HICON(win.LoadImage(hInstance, p, win.IMAGE_ICON,
			win.GetSystemMetrics(win.SM_CXICON), win.GetSystemMetrics(win.SM_CYICON), 0))
		small = win.HICON(win.LoadImage(hInstance, p, win.IMAGE_ICON,
			win.GetSystemMetrics(win.SM_CXSMICON), win.GetSystemMetrics(win.SM_CYSMICON), 0))
		if big != 0 {
			break
		}
	}
	if big == 0 {
		big = win.LoadIcon(0, win.MAKEINTRESOURCE(win.IDI_APPLICATION))
	}
	if small == 0 {
		small = big
	}
	debugLog("アイコン リソース数=%d big=%v small=%v", len(names), big != 0, small != 0)
	return
}
