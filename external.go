//go:build windows

package main

import (
	"net/url"
	"os/exec"
	"strings"
	"time"
	"unicode"
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
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
}
