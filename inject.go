//go:build windows

package main

import "strings"

const injectedJSTemplate = `
(function () {
  if (window.__xbrowser_injected) return;
  window.__xbrowser_injected = true;

  var ALLOWED_SUFFIXES = [
    'x.com', 'twitter.com', 'twimg.com', 'twimg.net'
  ];

  function isAllowedHost(host) {
    host = (host || '').toLowerCase();
    for (var i = 0; i < ALLOWED_SUFFIXES.length; i++) {
      var s = ALLOWED_SUFFIXES[i];
      if (host === s || (host.length > s.length && host.slice(-(s.length + 1)) === '.' + s)) {
        return true;
      }
    }
    return false;
  }

  function post(type, data) {
    try {
      window.chrome.webview.postMessage(JSON.stringify({ type: type, data: data }));
    } catch (e) {}
  }

  function onLinkClick(e) {
    if (e.type === 'auxclick' && e.button !== 1) return;
    var el = e.target && e.target.closest ? e.target.closest('a[href]') : null;
    if (!el) return;
    var u;
    try {
      u = new URL(el.getAttribute('href'), location.href);
    } catch (err) {
      return;
    }
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return;
    if (!isAllowedHost(u.hostname)) {
      e.preventDefault();
      e.stopPropagation();
      post('open_external', u.href);
      return;
    }
    var tg = (el.target || '').toLowerCase();
    var newTab = e.type === 'auxclick' || e.ctrlKey || e.shiftKey || e.metaKey ||
      (tg !== '' && tg !== '_self' && tg !== '_top' && tg !== '_parent');
    if (newTab) {
      e.preventDefault();
      e.stopPropagation();
      post('open_tab', u.href);
    }
  }
  document.addEventListener('click', onLinkClick, true);
  document.addEventListener('auxclick', onLinkClick, true);

  window.open = function (url) {
    try {
      var u = new URL(url, location.href);
      if (u.protocol === 'http:' || u.protocol === 'https:') {
        post(isAllowedHost(u.hostname) ? 'open_tab' : 'open_external', u.href);
      }
    } catch (e) {}
    return null;
  };

  var LABELS = {
    ja: { copy: 'コピー', translate: '翻訳', busy: '翻訳中…', fail: '翻訳できませんでした', google: 'Google翻訳で開く', close: '閉じる' },
    en: { copy: 'Copy', translate: 'Translate', busy: 'Translating…', fail: 'Could not translate', google: 'Open in Google Translate', close: 'Close' },
    zh: { copy: '复制', translate: '翻译', busy: '翻译中…', fail: '无法翻译', google: '在 Google 翻译中打开', close: '关闭' },
    ko: { copy: '복사', translate: '번역', busy: '번역 중…', fail: '번역하지 못했습니다', google: 'Google 번역에서 열기', close: '닫기' }
  };
  var MAX_TEXT = 2000;
  var menuHost = null;

  function closeMenu() {
    if (menuHost) { menuHost.remove(); menuHost = null; }
  }
  function isDarkPage() {
    var c = (getComputedStyle(document.body).backgroundColor || '').match(/\d+/g);
    return c ? (0.299 * c[0] + 0.587 * c[1] + 0.114 * c[2]) < 128 : false;
  }
  function copyText(text) {
    function fallback() {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.style.cssText = 'position:fixed;opacity:0;left:-9999px';
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand('copy'); } catch (e) {}
      ta.remove();
    }
    try {
      navigator.clipboard.writeText(text).catch(fallback);
    } catch (e) {
      fallback();
    }
  }
  function showMenu(x, y, text) {
    closeMenu();
    var L = LABELS[window.__xb_lang || '__XB_LANG__'] || LABELS.en;
    var dark = isDarkPage();
    menuHost = document.createElement('div');
    menuHost.style.cssText = 'all:initial;position:fixed;left:0;top:0;z-index:2147483647';
    var root = menuHost.attachShadow({ mode: 'closed' });
    var style = document.createElement('style');
    style.textContent =
      '.m{position:fixed;min-width:210px;padding:4px 0;border-radius:16px;box-sizing:border-box;overflow:hidden;' +
      'font:700 15px/20px "Segoe UI","Yu Gothic UI","Microsoft YaHei UI","Malgun Gothic",system-ui,sans-serif;' +
      'box-shadow:' + (dark ? 'rgba(255,255,255,.2) 0 0 15px,rgba(255,255,255,.15) 0 0 3px 1px' : 'rgba(101,119,134,.2) 0 0 15px,rgba(101,119,134,.15) 0 0 3px 1px') + ';' +
      'background:' + (dark ? '#000' : '#fff') + ';color:' + (dark ? '#E7E9EA' : '#0F1419') + '}' +
      '.i{display:flex;align-items:center;gap:12px;padding:12px 16px;cursor:pointer;user-select:none}' +
      '.i svg{flex:none;width:19px;height:19px;fill:none;stroke:currentColor;stroke-width:1.9;stroke-linecap:round;stroke-linejoin:round}' +
      '.i:hover{background:' + (dark ? 'rgba(231,233,234,.1)' : 'rgba(15,20,25,.1)') + '}';
    var box = document.createElement('div');
    box.className = 'm';
    var ICONS = {
      copy: '<svg viewBox="0 0 24 24"><rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V6a2 2 0 0 1 2-2h9"/></svg>',
      translate: '<svg viewBox="0 0 24 24"><path d="M3 5h10M8 3v2M5.5 5c.6 3.2 3 6 6 7.5M11.5 5c-.6 3.2-3.3 6.3-7 8"/><path d="M12.5 20l4-9 4 9M14 17h5"/></svg>'
    };
    [['copy', L.copy, function () { copyText(text); }],
     ['translate', L.translate, function () { startTranslate(text, x, y); }]].forEach(function (it) {
      var d = document.createElement('div');
      d.className = 'i';
      d.innerHTML = ICONS[it[0]];
      var sp = document.createElement('span');
      sp.textContent = it[1];
      d.appendChild(sp);
      d.addEventListener('click', function () { it[2](); closeMenu(); });
      box.appendChild(d);
    });
    box.addEventListener('mousedown', function (e) { e.preventDefault(); });
    root.appendChild(style);
    root.appendChild(box);
    document.documentElement.appendChild(menuHost);
    var r = box.getBoundingClientRect();
    box.style.left = Math.max(0, Math.min(x, window.innerWidth - r.width - 4)) + 'px';
    box.style.top = Math.max(0, Math.min(y, window.innerHeight - r.height - 4)) + 'px';
  }
  var popHost = null;
  var popBox = null;
  var reqId = 0;

  function closePopup() {
    reqId++;
    if (popHost) { popHost.remove(); popHost = null; popBox = null; }
  }
  function popCss(dark) {
    return '.p{position:fixed;width:min(420px,calc(100vw - 24px));box-sizing:border-box;padding:12px 16px;border-radius:16px;' +
      'font:14px/1.6 "Segoe UI","Yu Gothic UI","Microsoft YaHei UI","Malgun Gothic",system-ui,sans-serif;' +
      'box-shadow:' + (dark ? 'rgba(255,255,255,.2) 0 0 15px,rgba(255,255,255,.15) 0 0 3px 1px' : 'rgba(101,119,134,.2) 0 0 15px,rgba(101,119,134,.15) 0 0 3px 1px') + ';' +
      'background:' + (dark ? '#000' : '#fff') + ';color:' + (dark ? '#E7E9EA' : '#0F1419') + '}' +
      '.h{display:flex;align-items:center;gap:8px;margin-bottom:8px;color:' + (dark ? '#8B98A5' : '#536471') + ';font-size:12px}' +
      '.h span{flex:1}.x{cursor:pointer;padding:0 6px;font-size:16px;line-height:1}' +
      '.b{max-height:40vh;overflow:auto;white-space:pre-wrap;word-break:break-word;user-select:text}' +
      '.f{display:flex;gap:8px;margin-top:10px;flex-wrap:wrap}' +
      '.k{cursor:pointer;padding:5px 12px;border-radius:999px;font-size:13px;user-select:none;border:1px solid ' + (dark ? '#536471' : '#CFD9DE') + '}' +
      '.k:hover{background:' + (dark ? '#16181C' : '#EFF3F4') + '}';
  }
  function popButton(label, fn) {
    var d = document.createElement('div');
    d.className = 'k';
    d.textContent = label;
    d.addEventListener('click', fn);
    return d;
  }
  function placePopup(x, y) {
    var r = popBox.getBoundingClientRect();
    popBox.style.left = Math.max(8, Math.min(x, window.innerWidth - r.width - 8)) + 'px';
    popBox.style.top = Math.max(8, Math.min(y, window.innerHeight - r.height - 8)) + 'px';
  }
  var popX = 0, popY = 0, popText = '';
  function renderPopup(head, body, buttons) {
    if (!popBox) return;
    popBox.textContent = '';
    var h = document.createElement('div');
    h.className = 'h';
    var hs = document.createElement('span');
    hs.textContent = head;
    var hx = document.createElement('div');
    hx.className = 'x';
    hx.textContent = '×';
    hx.addEventListener('click', closePopup);
    h.appendChild(hs);
    h.appendChild(hx);
    var b = document.createElement('div');
    b.className = 'b';
    b.textContent = body;
    popBox.appendChild(h);
    popBox.appendChild(b);
    if (buttons.length) {
      var f = document.createElement('div');
      f.className = 'f';
      buttons.forEach(function (k) { f.appendChild(k); });
      popBox.appendChild(f);
    }
    placePopup(popX, popY);
  }
  function startTranslate(text, x, y) {
    closeMenu();
    closePopup();
    var id = ++reqId;
    var L = LABELS[window.__xb_lang || '__XB_LANG__'] || LABELS.en;
    popText = text;
    popX = x;
    popY = y;
    popHost = document.createElement('div');
    popHost.style.cssText = 'all:initial;position:fixed;left:0;top:0;z-index:2147483647';
    var root = popHost.attachShadow({ mode: 'closed' });
    var style = document.createElement('style');
    style.textContent = popCss(isDarkPage());
    popBox = document.createElement('div');
    popBox.className = 'p';
    popBox.addEventListener('mousedown', function (e) { e.stopPropagation(); });
    root.appendChild(style);
    root.appendChild(popBox);
    document.documentElement.appendChild(popHost);
    renderPopup(L.translate, L.busy, []);
    post('translate', JSON.stringify({ id: id, text: text }));
  }
  window.__xb_tr = function (r) {
    if (!popBox || !r || r.id !== reqId) return;
    var L = LABELS[window.__xb_lang || '__XB_LANG__'] || LABELS.en;
    var openG = popButton(L.google, function () { post('translate_open', popText); closePopup(); });
    if (r.ok) {
      var out = String(r.text || '');
      var head = L.translate + (r.src ? '  ' + String(r.src).toUpperCase() + ' → ' + String(r.tl).toUpperCase() : '');
      renderPopup(head, out, [popButton(L.copy, function () { copyText(out); }), openG]);
    } else {
      renderPopup(L.translate, L.fail, [openG]);
    }
  };
  window.__xb_translateSel = function () {
    var s = window.getSelection();
    var t = s ? s.toString().trim().slice(0, MAX_TEXT) : '';
    if (t) startTranslate(t, window.innerWidth / 2 - 210, 80);
  };
  function editable(el) {
    return !!(el && el.closest && el.closest('input,textarea,[contenteditable=""],[contenteditable="true"]'));
  }
  document.addEventListener('contextmenu', function (e) {
    if (editable(e.target)) { closeMenu(); return; }
    var sel = window.getSelection();
    var text = sel ? sel.toString().trim() : '';
    if (!(text && sel.containsNode(e.target, true))) {
      var tw = e.target.closest ? e.target.closest('[data-testid="tweetText"]') : null;
      text = tw ? (tw.innerText || '').trim() : '';
    }
    if (!text) { closeMenu(); return; }
    e.preventDefault();
    showMenu(e.clientX, e.clientY, text.slice(0, MAX_TEXT));
  }, true);
  document.addEventListener('mousedown', function (e) {
    if (menuHost && e.target !== menuHost) closeMenu();
    if (popHost && e.target !== popHost) closePopup();
  }, true);
  window.addEventListener('blur', closeMenu);
  window.addEventListener('resize', closeMenu);
  window.addEventListener('scroll', closeMenu, true);
  window.addEventListener('keydown', function (e) { if (e.key === 'Escape') { closeMenu(); closePopup(); } }, true);

  var lastSel = '';
  var selTimer = 0;
  document.addEventListener('selectionchange', function () {
    clearTimeout(selTimer);
    selTimer = setTimeout(function () {
      var s = window.getSelection();
      var t = s ? s.toString().trim().slice(0, MAX_TEXT) : '';
      if (t !== lastSel) { lastSel = t; post('selection', t); }
    }, 300);
  });

  var lastWheel = 0;
  window.addEventListener('wheel', function (e) {
    if (!e.ctrlKey) return;
    e.preventDefault();
    var now = Date.now();
    if (now - lastWheel < 60) return;
    lastWheel = now;
    post('zoom', e.deltaY < 0 ? 'in' : 'out');
  }, { passive: false, capture: true });
  window.addEventListener('keydown', function (e) {
    if (!e.ctrlKey || e.altKey) return;
    if (e.key === '+' || e.key === '=' || e.key === ';') { e.preventDefault(); post('zoom', 'in'); }
    else if (e.key === '-' || e.key === '_') { e.preventDefault(); post('zoom', 'out'); }
    else if (e.key === '0') { e.preventDefault(); post('zoom', 'reset'); }
  }, true);

  var lastTitle = null, lastUrl = null;
  function reportTitle() {
    if (document.title !== lastTitle) { lastTitle = document.title; post('title', lastTitle); }
    if (location.href !== lastUrl) { lastUrl = location.href; post('url', lastUrl); }
  }
  setInterval(reportTitle, 2000);

  ['pushState', 'replaceState'].forEach(function (k) {
    var orig = history[k];
    history[k] = function () {
      var r = orig.apply(this, arguments);
      setTimeout(reportTitle, 0);
      return r;
    };
  });

  var lastBadge = -1;
  function reportBadge() {
    var m = /^\((\d+)\)/.exec(document.title);
    var fromTitle = m ? parseInt(m[1], 10) : 0;
    var fromDom = 0;
    var els = document.querySelectorAll('a[href="/notifications"] [aria-live], a[href="/messages"] [aria-live]');
    for (var i = 0; i < els.length; i++) {
      var n = parseInt((els[i].textContent || '').replace(/[^0-9]/g, ''), 10);
      if (n > 0) fromDom += n;
    }
    var total = fromDom > 0 ? fromDom : fromTitle;
    if (total !== lastBadge) { lastBadge = total; post('badge', String(total)); }
  }
  setInterval(reportBadge, 3000);
  window.addEventListener('load', reportBadge);

  try {
    var c = /(?:^|; )night_mode=(\d)/.exec(document.cookie);
    post('theme_cookie', c ? c[1] : '');
  } catch (e) {}
  var titleEl = document.querySelector('title');
  if (titleEl) {
    new MutationObserver(function () { reportTitle(); reportBadge(); }).observe(titleEl, { childList: true });
  }
  window.addEventListener('popstate', reportTitle);
  reportTitle();
})();
`

func injectScript() string {
	lang := "ja"
	if isLang(cfg.Language) {
		lang = cfg.Language
	}
	return strings.Replace(injectedJSTemplate, "__XB_LANG__", lang, 1)
}
