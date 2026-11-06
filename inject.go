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
    if (e.type === 'auxclick' || e.ctrlKey || e.shiftKey || e.metaKey || (el.target && el.target.toLowerCase() !== '_self')) {
      e.preventDefault();
      e.stopPropagation();
      if (e.type === 'click') location.href = u.href;
    }
  }
  document.addEventListener('click', onLinkClick, true);
  document.addEventListener('auxclick', onLinkClick, true);

  window.open = function (url) {
    try {
      var u = new URL(url, location.href);
      if (u.protocol === 'http:' || u.protocol === 'https:') {
        if (!isAllowedHost(u.hostname)) {
          post('open_external', u.href);
        } else {
          location.href = u.href;
        }
      }
    } catch (e) {}
    return null;
  };

  var LABELS = {
    ja: { copy: 'コピー', translate: '翻訳' },
    en: { copy: 'Copy', translate: 'Translate' },
    zh: { copy: '复制', translate: '翻译' },
    ko: { copy: '복사', translate: '번역' }
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
      '.m{position:fixed;min-width:150px;padding:6px;border-radius:12px;box-sizing:border-box;' +
      'font:14px "Segoe UI","Yu Gothic UI","Microsoft YaHei UI","Malgun Gothic",system-ui,sans-serif;' +
      'box-shadow:0 4px 18px rgba(0,0,0,.35);border:1px solid ' + (dark ? '#2F3336' : '#CFD9DE') + ';' +
      'background:' + (dark ? '#000' : '#fff') + ';color:' + (dark ? '#E7E9EA' : '#0F1419') + '}' +
      '.i{padding:9px 12px;border-radius:8px;cursor:pointer;user-select:none}' +
      '.i:hover{background:' + (dark ? '#16181C' : '#EFF3F4') + '}';
    var box = document.createElement('div');
    box.className = 'm';
    [[L.copy, function () { copyText(text); }],
     [L.translate, function () { post('translate', text); }]].forEach(function (it) {
      var d = document.createElement('div');
      d.className = 'i';
      d.textContent = it[0];
      d.addEventListener('click', function () { it[1](); closeMenu(); });
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
  }, true);
  window.addEventListener('blur', closeMenu);
  window.addEventListener('resize', closeMenu);
  window.addEventListener('scroll', closeMenu, true);
  window.addEventListener('keydown', function (e) { if (e.key === 'Escape') closeMenu(); }, true);

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
	return strings.Replace(injectedJSTemplate, "__XB_LANG__", cfg.Language, 1)
}
