//go:build windows

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/lxn/win"
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

const translateEndpoint = "https://translate.googleapis.com/translate_a/single"

type translateResult struct {
	tab  *Tab
	id   int
	ok   bool
	text string
	src  string
	tl   string
}

var (
	translateDone   = make(chan translateResult, 16)
	translateClient = &http.Client{Timeout: 10 * time.Second}
)

func targetLang() string {
	if tl := translateLang[cfg.Language]; tl != "" {
		return tl
	}
	return "en"
}

func fetchTranslation(text, tl string) (out, src string, err error) {
	q := url.Values{}
	q.Set("client", "gtx")
	q.Set("sl", "auto")
	q.Set("tl", tl)
	q.Set("dt", "t")
	body := url.Values{"q": {text}}
	req, err := http.NewRequest(http.MethodPost, translateEndpoint+"?"+q.Encode(), strings.NewReader(body.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := translateClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errTranslateStatus
	}
	return parseTranslation(io.LimitReader(resp.Body, 1<<20))
}

func parseTranslation(r io.Reader) (out, src string, err error) {
	var v []any
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return "", "", err
	}
	if len(v) == 0 {
		return "", "", errTranslateFormat
	}
	segs, _ := v[0].([]any)
	var sb strings.Builder
	for _, seg := range segs {
		if arr, ok := seg.([]any); ok && len(arr) > 0 {
			if s, ok := arr[0].(string); ok {
				sb.WriteString(s)
			}
		}
	}
	if sb.Len() == 0 {
		return "", "", errTranslateFormat
	}
	if len(v) > 2 {
		src, _ = v[2].(string)
	}
	return sb.String(), src, nil
}

type translateError string

func (e translateError) Error() string { return string(e) }

const (
	errTranslateStatus translateError = "translate: bad status"
	errTranslateFormat translateError = "translate: bad format"
)

func startTranslate(tab *Tab, id int, text string) {
	text = cleanSelection(text)
	if text == "" || id <= 0 {
		return
	}
	hwnd, tl := app.hwnd, targetLang()
	go func() {
		out, src, err := fetchTranslation(text, tl)
		res := translateResult{tab: tab, id: id, ok: err == nil, text: out, src: src, tl: tl}
		select {
		case translateDone <- res:
			win.PostMessage(hwnd, wmAppAction, actTranslateDone, 0)
		default:
		}
	}()
}

func deliverTranslations() {
	for {
		select {
		case r := <-translateDone:
			r.tab.showTranslation(r)
		default:
			return
		}
	}
}

func (t *Tab) showTranslation(r translateResult) {
	alive := false
	for _, x := range app.tabs.tabs {
		if x == t {
			alive = true
			break
		}
	}
	if !alive || !t.ready {
		return
	}
	b, _ := json.Marshal(map[string]any{"id": r.id, "ok": r.ok, "text": r.text, "src": r.src, "tl": r.tl})
	t.Eval("window.__xb_tr&&window.__xb_tr(" + string(b) + ")")
}

func (t *Tab) translateSelection() {
	if t.ready && t.kind == kindWeb {
		t.Eval("window.__xb_translateSel&&window.__xb_translateSel()")
	}
}
