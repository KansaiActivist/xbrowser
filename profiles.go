//go:build windows

package main

import (
	"crypto/sha1"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lxn/win"
)

var slugPattern = regexp.MustCompile(`[^0-9A-Za-z一-龠ぁ-んァ-ヶー_-]+`)

func profileID(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "Default") {
		return "Default"
	}
	return name
}

// IDはログイン情報のフォルダ名の元で変わらない。表示名だけを設定で変えられる
func profileLabel(name string) string {
	id := profileID(name)
	if n := strings.TrimSpace(cfg.ProfileNames[id]); n != "" {
		return n
	}
	return id
}

func profileDataDir(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "Default") {
		return filepath.Join(appBaseDir(), "WebView2Data")
	}
	slug := slugPattern.ReplaceAllString(name, "_")
	if slug == "" {
		slug = "profile"
	}
	return filepath.Join(appBaseDir(), "WebView2Data-"+slug)
}

func allProfiles() []string {
	return append([]string{"Default"}, cfg.Profiles...)
}

func profileMenuLabel(id string) string {
	if id == "Default" && strings.TrimSpace(cfg.ProfileNames[id]) == "" {
		return T("profile_default")
	}
	return profileLabel(id)
}

func findProfileID(name string) string {
	name = strings.TrimSpace(name)
	for _, id := range allProfiles() {
		if strings.EqualFold(id, name) || strings.EqualFold(profileLabel(id), name) {
			return id
		}
	}
	return ""
}

func profileExists(name string) bool { return findProfileID(name) != "" }

func dataDirConflict(name string) bool {
	dir := profileDataDir(name)
	for _, id := range allProfiles() {
		if strings.EqualFold(profileDataDir(id), dir) {
			return true
		}
	}
	return false
}

// 追加してよい名前か調べ、だめなら翻訳キーを返す(OKなら空文字)
func validateNewProfile(name string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "toast_bad_name"
	case len([]rune(name)) > maxProfileNameLen:
		return "toast_toolong"
	case strings.EqualFold(name, "Default") || profileExists(name) || dataDirConflict(name):
		return "toast_exists"
	case len(cfg.Profiles) >= maxProfiles:
		return "toast_limit"
	}
	return ""
}

func addProfile(name string) bool {
	name = strings.TrimSpace(name)
	if validateNewProfile(name) != "" {
		return false
	}
	cfg.Profiles = append(cfg.Profiles, name)
	saveConfig()
	return true
}

func removeProfile(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "Default") {
		return false
	}
	for i, p := range cfg.Profiles {
		if strings.EqualFold(p, name) {
			cfg.Profiles = append(cfg.Profiles[:i], cfg.Profiles[i+1:]...)
			delete(cfg.ProfileNames, p)
			delete(cfg.ProfileColors, p)
			saveConfig()
			return true
		}
	}
	return false
}

var profileDotColors = []win.COLORREF{
	rgb(0x1D, 0x9B, 0xF0),
	rgb(0x17, 0xBF, 0x63),
	rgb(0xF9, 0x1A, 0x80),
	rgb(0xFF, 0xAD, 0x1F),
	rgb(0x79, 0x56, 0xF2),
	rgb(0x00, 0xC2, 0xB8),
	rgb(0xE0, 0x24, 0x5E),
	rgb(0xC4, 0xA0, 0x00),
}

func profileColor(name string) (win.COLORREF, bool) {
	id := profileID(name)
	if hex, ok := cfg.ProfileColors[id]; ok && hexColorRe.MatchString(hex) {
		return parseHex(hex, 0), true
	}
	if id == "Default" {
		return 0, false
	}
	sum := sha1.Sum([]byte(strings.ToLower(id)))
	return profileDotColors[int(sum[0])%len(profileDotColors)], true
}

func profileColorHex(name string) (string, bool) {
	c, ok := profileColor(name)
	if !ok {
		return "", false
	}
	_, custom := cfg.ProfileColors[profileID(name)]
	return colorHex(c), custom
}

func profilePaletteHex() []string {
	out := make([]string, len(profileDotColors))
	for i, c := range profileDotColors {
		out[i] = colorHex(c)
	}
	return out
}

const (
	maxProfileNameLen = 30
	maxProfiles       = 50
)

func renameProfile(id, newName string) string {
	id = findProfileID(id)
	newName = strings.TrimSpace(newName)
	if id == "" || newName == "" {
		return "toast_bad_name"
	}
	if len([]rune(newName)) > maxProfileNameLen {
		return "toast_toolong"
	}
	if other := findProfileID(newName); other != "" && other != id {
		return "toast_exists"
	}
	if cfg.ProfileNames == nil {
		cfg.ProfileNames = map[string]string{}
	}
	if newName == id {
		delete(cfg.ProfileNames, id)
	} else {
		cfg.ProfileNames[id] = newName
	}
	saveConfig()
	return ""
}

func setProfileColor(id, hex string) bool {
	id = findProfileID(id)
	if id == "" {
		return false
	}
	if hex != "" && !hexColorRe.MatchString(hex) {
		return false
	}
	if hex == "" {
		delete(cfg.ProfileColors, id)
	} else {
		if cfg.ProfileColors == nil {
			cfg.ProfileColors = map[string]string{}
		}
		cfg.ProfileColors[id] = strings.ToUpper(hex)
	}
	saveConfig()
	return true
}

func cleanProfileMeta() {
	names, colors := map[string]string{}, map[string]string{}
	for _, id := range allProfiles() {
		if n := strings.TrimSpace(cfg.ProfileNames[id]); n != "" && n != id && len([]rune(n)) <= maxProfileNameLen {
			names[id] = n
		}
		if c := cfg.ProfileColors[id]; hexColorRe.MatchString(c) {
			colors[id] = strings.ToUpper(c)
		}
	}
	cfg.ProfileNames, cfg.ProfileColors = names, colors
}
