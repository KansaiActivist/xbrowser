//go:build windows

package main

import (
	"net/url"
	"strings"
	"unicode"
)

const maxTranslateRunes = 1500

var translateLang = map[string]string{"ja": "ja", "en": "en", "zh": "zh-CN", "ko": "ko"}

func cleanSelection(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return ' '
		}
		return r
	}, s)
	r := []rune(strings.TrimSpace(s))
	if len(r) > maxTranslateRunes {
		r = r[:maxTranslateRunes]
	}
	return strings.TrimSpace(string(r))
}

func translateURL(text string) string {
	text = cleanSelection(text)
	if text == "" {
		return ""
	}
	tl := translateLang[cfg.Language]
	if tl == "" {
		tl = "en"
	}
	q := url.Values{}
	q.Set("sl", "auto")
	q.Set("tl", tl)
	q.Set("text", text)
	q.Set("op", "translate")
	return "https://translate.google.com/?" + q.Encode()
}

func translateText(text string) {
	if u := translateURL(text); u != "" {
		openExternal(u)
	}
}
