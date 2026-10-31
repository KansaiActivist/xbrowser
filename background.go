//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	utf16pkg "unicode/utf16"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/lxn/win"
)

const (
	bgHost       = "xbrowser.local"
	bgMaxBytes   = 20 << 20
	bgFilePrefix = "background-"
)

var bgExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true}

func startBgDir() string { return filepath.Join(appBaseDir(), "StartBg") }

func startImageURL() string {
	if cfg.StartImage == "" {
		return ""
	}
	p := filepath.Join(startBgDir(), cfg.StartImage)
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return ""
	}

	return fmt.Sprintf("https://%s/%s?v=%d", bgHost, cfg.StartImage, st.ModTime().UnixNano())
}

func mapBackgroundHost(ch *edge.Chromium) {
	if err := os.MkdirAll(startBgDir(), 0o755); err != nil {
		return
	}
	wv3 := ch.GetICoreWebView2_3()
	if wv3 == nil {
		debugLog("仮想ホストの割り当てに非対応のWebView2です(背景画像は表示されません)")
		return
	}
	if err := wv3.SetVirtualHostNameToFolderMapping(bgHost, startBgDir(), edge.COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_ALLOW); err != nil {
		debugLog("仮想ホストの割り当てに失敗 err=%v", err)
	}
}

func pickImageFile(owner win.HWND) (string, bool) {
	filter := T("pick_filter") + "\x00*.jpg;*.jpeg;*.png;*.gif;*.webp;*.bmp\x00\x00"
	// フィルタはNUL区切りなので、NULでpanicするStringToUTF16は使わない
	filterU := utf16pkg.Encode([]rune(filter + "\x00"))
	buf := make([]uint16, 1024)
	title, _ := syscall.UTF16PtrFromString(T("pick_title"))

	ofn := win.OPENFILENAME{
		LStructSize:  uint32(unsafe.Sizeof(win.OPENFILENAME{})),
		HwndOwner:    owner,
		LpstrFilter:  &filterU[0],
		NFilterIndex: 1,
		LpstrFile:    &buf[0],
		NMaxFile:     uint32(len(buf)),
		LpstrTitle:   title,
		Flags:        win.OFN_EXPLORER | win.OFN_FILEMUSTEXIST | win.OFN_PATHMUSTEXIST | win.OFN_HIDEREADONLY | win.OFN_NOCHANGEDIR,
	}
	if !win.GetOpenFileName(&ofn) {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

func importBackground(src string) (errKey string) {
	ext := strings.ToLower(filepath.Ext(src))
	if !bgExts[ext] {
		return "toast_img_bad"
	}
	in, err := os.Open(src)
	if err != nil {
		return "toast_img_bad"
	}
	defer in.Close()
	if st, err := in.Stat(); err != nil || st.IsDir() {
		return "toast_img_bad"
	} else if st.Size() > bgMaxBytes {
		return "toast_img_big"
	}

	if err := os.MkdirAll(startBgDir(), 0o755); err != nil {
		return "toast_img_bad"
	}
	name := fmt.Sprintf("%s%d%s", bgFilePrefix, time.Now().UnixNano(), ext)
	dst := filepath.Join(startBgDir(), name)
	out, err := os.Create(dst)
	if err != nil {
		return "toast_img_bad"
	}
	_, err = io.Copy(out, io.LimitReader(in, bgMaxBytes+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(dst)
		return "toast_img_bad"
	}

	removeBackgroundFiles(name)
	cfg.StartImage = name
	cfg.StartBg = "image"
	saveConfig()
	return ""
}

func removeBackgroundFiles(keep string) {
	entries, err := os.ReadDir(startBgDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() && e.Name() != keep && strings.HasPrefix(e.Name(), bgFilePrefix) {
			_ = os.Remove(filepath.Join(startBgDir(), e.Name()))
		}
	}
}

func clearBackground() {
	removeBackgroundFiles("")
	cfg.StartImage = ""
	if cfg.StartBg == "image" {
		cfg.StartBg = "black"
		if presetName() == "light" {
			cfg.StartBg = "white"
		}
	}
	saveConfig()
}
