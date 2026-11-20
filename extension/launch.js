const api = globalThis.browser || globalThis.chrome;
const target = new URLSearchParams(location.search).get('u') || '';
const valid = /^xbrowser:\/\/share\?[\w%.~!*'();:@&=+$,\/?#\[\]-]*$/.test(target) && target.length < 8000;

const ja = (navigator.language || '').startsWith('ja');
if (!ja) {
  document.documentElement.lang = 'en';
  document.getElementById('title').textContent = 'Opening in XBrowser';
  document.getElementById('msg').textContent = 'If nothing happens, press the button below. If it still does not open, turn on "Accept shares" in XBrowser settings and check "XBrowser location".';
  document.getElementById('open').textContent = 'Open in XBrowser';
  document.getElementById('close').textContent = 'Close';
}

let launched = false;
let closing = false;

async function closeSelf() {
  if (closing) return;
  closing = true;
  try {
    const tab = await api.tabs.getCurrent();
    await api.tabs.remove(tab.id);
  } catch (e) {
    window.close();
  }
}

function afterAppOpened() {
  if (launched) setTimeout(closeSelf, 800);
}

function launch() {
  if (!valid) return;
  launched = true;
  location.href = target;
}

window.addEventListener('blur', afterAppOpened);
document.addEventListener('visibilitychange', () => {
  if (document.hidden) afterAppOpened();
});

document.getElementById('open').addEventListener('click', launch);
document.getElementById('close').addEventListener('click', closeSelf);

if (valid) setTimeout(launch, 100);
