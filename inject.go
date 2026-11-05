//go:build windows

package main

const injectedJS = `
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
      if (host === s || host.indexOf('.' + s) === host.length - s.length - 1) {
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

  document.addEventListener('click', function (e) {
    var el = e.target;
    while (el && el.tagName !== 'A') el = el.parentElement;
    if (!el || !el.href) return;
    var u;
    try {
      u = new URL(el.href, location.href);
    } catch (err) {
      return;
    }
    if (!isAllowedHost(u.hostname)) {
      e.preventDefault();
      e.stopPropagation();
      post('open_external', u.href);
      return;
    }
    if (el.target && el.target !== '' && el.target.toLowerCase() !== '_self') {
      e.preventDefault();
      e.stopPropagation();
      location.href = u.href;
    }
  }, true);

  window.open = function (url, target, features) {
    try {
      var u = new URL(url, location.href);
      if (!isAllowedHost(u.hostname)) {
        post('open_external', u.href);
      } else {
        location.href = u.href;
      }
    } catch (e) {}
    return null;
  };

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
