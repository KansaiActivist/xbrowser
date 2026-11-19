var LABEL = {
  ja: { open: '他のSNSを開く', current: '表示中' },
  en: { open: 'Open other social networks', current: 'Current' },
  zh: { open: '打开其他社交网络', current: '当前' },
  ko: { open: '다른 SNS 열기', current: '현재' }
}[xb.lang] || { open: 'Open other social networks', current: 'Current' };

var cfg = xb.config || {};
var sites = (cfg.sites || []).filter(function (s) {
  try { return new URL(s.url).protocol === 'https:' && xb.isAllowedHost(new URL(s.url).hostname); } catch (e) { return false; }
});

var CSS = [
  ':host { all: initial; }',
  '.btn { position: fixed; bottom: 16px; ' + (cfg.position === 'right' ? 'right' : 'left') + ': 16px; z-index: 2147483000; width: 44px; height: 44px; border-radius: 50%; border: 1px solid var(--line); background: var(--bg); color: var(--fg); cursor: pointer; display: grid; place-items: center; box-shadow: 0 2px 10px rgba(0,0,0,.35); opacity: .75; transition: opacity .15s, transform .15s; padding: 0; }',
  '.btn:hover, .btn[aria-expanded="true"] { opacity: 1; transform: scale(1.06); }',
  '.btn svg { width: 22px; height: 22px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }',
  '.panel { position: fixed; bottom: 68px; ' + (cfg.position === 'right' ? 'right' : 'left') + ': 16px; z-index: 2147483000; width: 240px; max-height: 70vh; overflow-y: auto; padding: 6px; border-radius: 16px; border: 1px solid var(--line); background: var(--bg); color: var(--fg); box-shadow: 0 8px 28px rgba(0,0,0,.45); font: 15px/1.3 system-ui, "Segoe UI", sans-serif; display: none; }',
  '.panel.open { display: block; }',
  '.item { display: flex; align-items: center; gap: 12px; width: 100%; box-sizing: border-box; padding: 10px 12px; border: 0; border-radius: 10px; background: transparent; color: inherit; font: inherit; text-align: left; cursor: pointer; }',
  '.item:hover { background: var(--hover); }',
  '.dot { flex: none; width: 26px; height: 26px; border-radius: 50%; display: grid; place-items: center; color: #fff; font-weight: 700; font-size: 13px; }',
  '.name { flex: 1; font-weight: 600; }',
  '.cur { font-size: 12px; color: var(--sub); }'
].join('\n');

function isDark() {
  var m = /rgba?\((\d+),\s*(\d+),\s*(\d+)/.exec(getComputedStyle(document.body || document.documentElement).backgroundColor);
  if (!m) return window.matchMedia && matchMedia('(prefers-color-scheme: dark)').matches;
  return (0.299 * m[1] + 0.587 * m[2] + 0.114 * m[3]) < 140;
}

function build() {
  if (!sites.length || document.getElementById('xbp-sns-hub')) return;
  var host = document.createElement('div');
  host.id = 'xbp-sns-hub';
  var root = host.attachShadow({ mode: 'open' });
  var sheet;
  try { sheet = new CSSStyleSheet(); sheet.replaceSync(CSS); root.adoptedStyleSheets = [sheet]; }
  catch (e) { var st = document.createElement('style'); st.textContent = CSS; root.appendChild(st); }

  var wrap = document.createElement('div');
  var btn = document.createElement('button');
  btn.className = 'btn';
  btn.type = 'button';
  btn.title = LABEL.open;
  btn.setAttribute('aria-label', LABEL.open);
  btn.setAttribute('aria-expanded', 'false');
  btn.innerHTML = '<svg viewBox="0 0 24 24"><circle cx="6" cy="12" r="2.5"/><circle cx="18" cy="6" r="2.5"/><circle cx="18" cy="18" r="2.5"/><path d="M8.2 10.8l7.6-3.6M8.2 13.2l7.6 3.6"/></svg>';
  var panel = document.createElement('div');
  panel.className = 'panel';

  sites.forEach(function (s) {
    var u = new URL(s.url);
    var item = document.createElement('button');
    item.className = 'item';
    item.type = 'button';
    var dot = document.createElement('span');
    dot.className = 'dot';
    dot.style.background = /^#[0-9a-f]{6}$/i.test(s.color || '') ? s.color : '#71767b';
    dot.textContent = (s.name || u.hostname).charAt(0).toUpperCase();
    var name = document.createElement('span');
    name.className = 'name';
    name.textContent = s.name || u.hostname;
    item.appendChild(dot);
    item.appendChild(name);
    var here = location.hostname === u.hostname || location.hostname.slice(-(u.hostname.replace(/^www\./, '').length + 1)) === '.' + u.hostname.replace(/^www\./, '');
    if (here) {
      var cur = document.createElement('span');
      cur.className = 'cur';
      cur.textContent = LABEL.current;
      item.appendChild(cur);
    }
    item.addEventListener('click', function () {
      panel.classList.remove('open');
      btn.setAttribute('aria-expanded', 'false');
      xb.openTab(s.url);
    });
    panel.appendChild(item);
  });

  function applyTheme() {
    var dark = isDark();
    host.style.cssText = dark
      ? '--bg:#16181c;--fg:#e7e9ea;--line:#2f3336;--hover:#1d1f23;--sub:#71767b'
      : '--bg:#ffffff;--fg:#0f1419;--line:#cfd9de;--hover:#f2f5f7;--sub:#536471';
  }
  applyTheme();
  setInterval(applyTheme, 5000);

  btn.addEventListener('click', function (e) {
    e.stopPropagation();
    var open = !panel.classList.contains('open');
    panel.classList.toggle('open', open);
    btn.setAttribute('aria-expanded', open ? 'true' : 'false');
  });
  document.addEventListener('click', function (e) {
    if (!host.contains(e.target) && panel.classList.contains('open')) {
      panel.classList.remove('open');
      btn.setAttribute('aria-expanded', 'false');
    }
  }, true);
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') { panel.classList.remove('open'); btn.setAttribute('aria-expanded', 'false'); }
  });

  wrap.appendChild(panel);
  wrap.appendChild(btn);
  root.appendChild(wrap);
  document.documentElement.appendChild(host);
}

xb.onReady(function () {
  build();
  setInterval(build, 3000);
});
