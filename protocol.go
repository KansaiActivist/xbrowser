//go:build windows

package main

import (
	"net/url"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	protocolScheme  = "xbrowser"
	protocolRegPath = `Software\Classes\xbrowser`
	maxShareText    = 2000
)

var shareParams = []string{"text", "url", "hashtags", "via", "related", "in_reply_to"}

func shareURLFromProtocol(arg string) (string, bool) {
	arg = strings.TrimSpace(strings.Trim(arg, "\""))
	if len(arg) > maxExternalURLLen || len(arg) < len(protocolScheme)+1 || !strings.EqualFold(arg[:len(protocolScheme)+1], protocolScheme+":") {
		return "", false
	}
	rest := strings.TrimLeft(arg[len(protocolScheme)+1:], "/")
	if k := strings.IndexByte(rest, '#'); k >= 0 {
		rest = rest[:k]
	}
	name, query := rest, ""
	if k := strings.IndexByte(rest, '?'); k >= 0 {
		name, query = rest[:k], rest[k+1:]
	}
	q, _ := url.ParseQuery(query)
	switch strings.ToLower(strings.Trim(name, "/")) {
	case "share":
		out := url.Values{}
		for _, k := range shareParams {
			if v := cleanShareValue(q.Get(k)); v != "" {
				out.Set(k, v)
			}
		}
		return "https://x.com/intent/post?" + out.Encode(), true
	case "open":
		return normalizeXURL(q.Get("url"))
	}
	return "", false
}

func refreshProtocolRegistration() {
	k, err := registry.OpenKey(registry.CURRENT_USER, protocolRegPath+`\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	k.Close()
	if !protocolRegistered() {
		if err := setProtocolRegistered(true); err != nil {
			debugLog("プロトコル登録の更新に失敗 err=%v", err)
		}
	}
}

func cleanShareValue(s string) string {
	r := []rune(strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' {
			return -1
		}
		return r
	}, s))
	if len(r) > maxShareText {
		r = r[:maxShareText]
	}
	return string(r)
}

func protocolCommand() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return `"` + exe + `" "%1"`
}

func protocolRegistered() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, protocolRegPath+`\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("")
	return err == nil && strings.EqualFold(v, protocolCommand())
}

func setProtocolRegistered(on bool) error {
	if !on {
		for _, sub := range []string{`\shell\open\command`, `\shell\open`, `\shell`, `\DefaultIcon`, ``} {
			if err := registry.DeleteKey(registry.CURRENT_USER, protocolRegPath+sub); err != nil && err != registry.ErrNotExist {
				return err
			}
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	set := func(path, name, value string) error {
		k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
		if err != nil {
			return err
		}
		defer k.Close()
		return k.SetStringValue(name, value)
	}
	if err := set(protocolRegPath, "", "URL:XBrowser"); err != nil {
		return err
	}
	if err := set(protocolRegPath, "URL Protocol", ""); err != nil {
		return err
	}
	if err := set(protocolRegPath+`\DefaultIcon`, "", `"`+exe+`",0`); err != nil {
		return err
	}
	return set(protocolRegPath+`\shell\open\command`, "", protocolCommand())
}
