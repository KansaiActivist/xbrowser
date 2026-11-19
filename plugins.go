//go:build windows

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
)

const (
	pluginManifestName = "plugin.json"
	maxPlugins         = 32
	maxPluginFileBytes = 512 * 1024
	maxPluginTotal     = 2 * 1024 * 1024
	maxManifestBytes   = 64 * 1024
	maxPluginHosts     = 64
)

var (
	pluginIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)
	pluginHostPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)
	publicSuffixes    = map[string]bool{
		"co.jp": true, "ne.jp": true, "or.jp": true, "ac.jp": true, "go.jp": true,
		"co.uk": true, "org.uk": true, "ac.uk": true, "gov.uk": true,
		"com.au": true, "net.au": true, "org.au": true, "com.cn": true, "net.cn": true,
		"co.kr": true, "or.kr": true, "co.nz": true, "com.br": true, "co.in": true,
		"github.io": true, "pages.dev": true, "workers.dev": true, "herokuapp.com": true,
		"blogspot.com": true, "appspot.com": true, "web.app": true, "netlify.app": true,
		"vercel.app": true, "azurewebsites.net": true, "cloudfront.net": true,
	}
)

type Plugin struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Matches     []string        `json:"matches"`
	AllowHosts  []string        `json:"allowHosts"`
	JS          []string        `json:"js"`
	CSS         []string        `json:"css"`
	Config      json.RawMessage `json:"config"`

	dir    string
	source string
	style  string
	loaded bool
	err    string
}

var (
	plugins     []*Plugin
	pluginHosts []string
)

func pluginDirs() []string {
	dirs := []string{filepath.Join(appBaseDir(), "plugins")}
	if exe := detectedExe(); exe != "" {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "plugins"))
	}
	return dirs
}

func userPluginDir() string { return pluginDirs()[0] }

func normalizePluginHost(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "*.")
	if len(s) > 253 || !pluginHostPattern.MatchString(s) || publicSuffixes[s] {
		return "", false
	}
	tld := s[strings.LastIndexByte(s, '.')+1:]
	if !strings.ContainsAny(tld, "abcdefghijklmnopqrstuvwxyz") {
		return "", false
	}
	return s, true
}

func normalizeHostList(in []string) ([]string, bool) {
	if len(in) > maxPluginHosts {
		return nil, false
	}
	out := make([]string, 0, len(in))
	for _, h := range in {
		n, ok := normalizePluginHost(h)
		if !ok {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

func hostMatches(host string, list []string) bool {
	host = strings.ToLower(host)
	for _, s := range list {
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}
	return false
}

func pluginDisabled(id string) bool {
	for _, d := range cfg.PluginsDisabled {
		if d == id {
			return true
		}
	}
	return false
}

func readPluginFile(dir, rel string, ext string, total *int) (string, bool) {
	if rel == "" || filepath.IsAbs(rel) || strings.ContainsAny(rel, ":\x00") || !strings.EqualFold(filepath.Ext(rel), ext) {
		return "", false
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", false
	}
	realFile, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", false
	}
	if r, err := filepath.Rel(realDir, realFile); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) || filepath.IsAbs(r) {
		return "", false
	}
	b, err := readLimited(realFile, maxPluginFileBytes)
	if err != nil {
		return "", false
	}
	*total += len(b)
	if *total > maxPluginTotal {
		return "", false
	}
	return strings.TrimPrefix(string(b), "\xef\xbb\xbf"), true
}

func loadPluginDir(dir string) (*Plugin, string) {
	raw, err := readLimited(filepath.Join(dir, pluginManifestName), maxManifestBytes)
	if err != nil {
		return nil, "plugin.json"
	}
	var p Plugin
	if json.Unmarshal(raw, &p) != nil {
		return nil, "plugin.json"
	}
	p.dir = dir
	if !pluginIDPattern.MatchString(p.ID) {
		return &p, "id"
	}
	if p.Name = strings.TrimSpace(p.Name); p.Name == "" {
		p.Name = p.ID
	}
	p.Name = truncateRunes(p.Name, 60)
	p.Version = truncateRunes(strings.TrimSpace(p.Version), 20)
	p.Description = truncateRunes(strings.TrimSpace(p.Description), 300)
	var ok bool
	if p.Matches, ok = normalizeHostList(p.Matches); !ok {
		return &p, "matches"
	}
	if p.AllowHosts, ok = normalizeHostList(p.AllowHosts); !ok {
		return &p, "allowHosts"
	}
	total := 0
	var js, css []string
	for _, f := range p.JS {
		s, ok := readPluginFile(dir, f, ".js", &total)
		if !ok {
			return &p, f
		}
		js = append(js, s)
	}
	for _, f := range p.CSS {
		s, ok := readPluginFile(dir, f, ".css", &total)
		if !ok {
			return &p, f
		}
		css = append(css, s)
	}
	if len(p.Config) > 0 && !json.Valid(p.Config) {
		return &p, "config"
	}
	p.source = strings.Join(js, "\n;\n")
	p.style = strings.Join(css, "\n")
	p.loaded = true
	return &p, ""
}

func loadPlugins() {
	seen := map[string]bool{}
	var list []*Plugin
	for _, root := range pluginDirs() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || len(list) >= maxPlugins {
				continue
			}
			p, bad := loadPluginDir(filepath.Join(root, e.Name()))
			if p == nil {
				debugLog("プラグインを読み込めません dir=%s file=%s", e.Name(), bad)
				continue
			}
			if bad != "" {
				p.err = bad
				debugLog("プラグインが不正です dir=%s 原因=%s", e.Name(), bad)
				if p.ID == "" || !pluginIDPattern.MatchString(p.ID) {
					continue
				}
			}
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			list = append(list, p)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	plugins = list
	rebuildPluginHosts()
}

func pluginActive(p *Plugin) bool { return p.loaded && !pluginDisabled(p.ID) }

func rebuildPluginHosts() {
	var hosts []string
	seen := map[string]bool{}
	for _, p := range plugins {
		if !pluginActive(p) {
			continue
		}
		for _, h := range p.AllowHosts {
			if !seen[h] {
				seen[h] = true
				hosts = append(hosts, h)
			}
		}
	}
	pluginHosts = hosts
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func pluginScript(p *Plugin, lang string) string {
	cfgJSON := "{}"
	if len(p.Config) > 0 {
		cfgJSON = string(p.Config)
	}
	allowed := append(append([]string{}, allowedHostSuffixes...), pluginHosts...)
	r := strings.NewReplacer(
		"__XBP_ID__", mustJSON(p.ID),
		"__XBP_NAME__", mustJSON(p.Name),
		"__XBP_VERSION__", mustJSON(p.Version),
		"__XBP_LANG__", mustJSON(lang),
		"__XBP_ALLOWED__", mustJSON(allowed),
		"__XBP_BASE__", mustJSON(allowedHostSuffixes),
		"__XBP_MATCH__", mustJSON(p.Matches),
		"__XBP_CONFIG__", cfgJSON,
		"__XBP_CSS__", mustJSON(p.style),
	)
	return r.Replace(pluginWrapperHead) + "\n" + p.source + "\n" + pluginWrapperTail
}

func pluginScripts() []string {
	lang := "ja"
	if isLang(cfg.Language) {
		lang = cfg.Language
	}
	var out []string
	for _, p := range plugins {
		if pluginActive(p) {
			out = append(out, pluginScript(p, lang))
		}
	}
	return out
}

func pluginStates() []map[string]any {
	out := make([]map[string]any, 0, len(plugins))
	for _, p := range plugins {
		hosts := p.AllowHosts
		if hosts == nil {
			hosts = []string{}
		}
		out = append(out, map[string]any{
			"id":          p.ID,
			"name":        p.Name,
			"version":     p.Version,
			"description": p.Description,
			"enabled":     !pluginDisabled(p.ID),
			"error":       p.err,
			"hosts":       hosts,
		})
	}
	return out
}

func setPluginEnabled(id string, on bool) {
	var list []string
	for _, d := range cfg.PluginsDisabled {
		if d != id {
			list = append(list, d)
		}
	}
	if !on {
		list = append(list, id)
	}
	cfg.PluginsDisabled = list
	saveConfig()
	rebuildPluginHosts()
}

func openPluginFolder() {
	dir := userPluginDir()
	_ = os.MkdirAll(dir, 0o755)
	win, err := windows.GetWindowsDirectory()
	if err != nil || win == "" {
		return
	}
	_ = exec.Command(filepath.Join(win, "explorer.exe"), dir).Start()
}

const pluginWrapperHead = `(function () {
  if (window.top !== window) return;
  var ALLOWED = __XBP_ALLOWED__, BASE = __XBP_BASE__, MATCH = __XBP_MATCH__;
  function within(host, list) {
    host = (host || '').toLowerCase();
    for (var i = 0; i < list.length; i++) {
      var s = list[i];
      if (host === s || (host.length > s.length && host.slice(-(s.length + 1)) === '.' + s)) return true;
    }
    return false;
  }
  var host = location.hostname;
  if (location.protocol !== 'https:' || !within(host, ALLOWED)) return;
  if (MATCH.length && !within(host, MATCH)) return;
  var ID = __XBP_ID__;
  function send(type, data) {
    try { window.chrome.webview.postMessage(JSON.stringify({ type: type, data: data })); } catch (e) {}
  }
  function safeURL(u) {
    try {
      var x = new URL(String(u), location.href);
      return (x.protocol === 'https:' || x.protocol === 'http:') ? x : null;
    } catch (e) { return null; }
  }
  function ready(fn) {
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', function () { try { fn(); } catch (e) { console.error('[XBrowser plugin ' + ID + ']', e); } });
    else { try { fn(); } catch (e) { console.error('[XBrowser plugin ' + ID + ']', e); } }
  }
  function addCSS(text) {
    if (!text) return;
    try {
      var sheet = new CSSStyleSheet();
      sheet.replaceSync(text);
      document.adoptedStyleSheets = document.adoptedStyleSheets.concat([sheet]);
    } catch (e) {
      ready(function () {
        var el = document.createElement('style');
        el.textContent = text;
        (document.head || document.documentElement).appendChild(el);
      });
    }
  }
  var store = {
    get: function (k, d) { try { var v = localStorage.getItem('__xbp:' + ID + ':' + k); return v === null ? d : JSON.parse(v); } catch (e) { return d; } },
    set: function (k, v) { try { localStorage.setItem('__xbp:' + ID + ':' + k, JSON.stringify(v)); } catch (e) {} },
    remove: function (k) { try { localStorage.removeItem('__xbp:' + ID + ':' + k); } catch (e) {} }
  };
  var xb = Object.freeze({
    id: ID,
    name: __XBP_NAME__,
    version: __XBP_VERSION__,
    host: host,
    isX: within(host, BASE),
    lang: __XBP_LANG__,
    config: __XBP_CONFIG__,
    css: addCSS,
    onReady: ready,
    storage: Object.freeze(store),
    isAllowedHost: function (h) { return within(h, ALLOWED); },
    openTab: function (u) {
      var x = safeURL(u);
      if (x && x.protocol === 'https:' && within(x.hostname, ALLOWED)) send('open_tab', x.href);
    },
    openExternal: function (u) {
      var x = safeURL(u);
      if (x) send('open_external', x.href);
    }
  });
  addCSS(__XBP_CSS__);
  try {
`

const pluginWrapperTail = `
  } catch (e) { console.error('[XBrowser plugin ' + ID + ']', e); }
})();`
