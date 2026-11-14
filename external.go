//go:build windows

package main

import (
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"golang.org/x/sys/windows"
)

const maxExternalURLLen = 8192

var lastExternalOpen time.Time

func safeExternalURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxExternalURLLen {
		return "", false
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", false
	}
	return u.String(), true
}

func openExternal(raw string) {
	target, ok := safeExternalURL(raw)
	if !ok {
		debugLog("外部URLを拒否しました")
		return
	}
	if time.Since(lastExternalOpen) < 400*time.Millisecond {
		return
	}
	lastExternalOpen = time.Now()
	_ = exec.Command(system32Path("rundll32.exe"), "url.dll,FileProtocolHandler", target).Start()
}

const maxConfigBytes = 1 << 20

func system32Path(name string) string {
	dir, err := windows.GetSystemDirectory()
	if err != nil || dir == "" {
		return name
	}
	return filepath.Join(dir, name)
}

func readLimited(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("file too large")
	}
	return b, nil
}
