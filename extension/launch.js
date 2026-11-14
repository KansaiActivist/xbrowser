const api = globalThis.browser || globalThis.chrome;
const target = new URLSearchParams(location.search).get('u') || '';
const valid = /^xbrowser:\/\/share\?[\w%.~!*'();:@&=+$,\/?#\[\]-]*$/.test(target) && target.length < 8000;

const ja = (navigator.language || '').startsWith('ja');
if (!ja) {
  document.documentElement.lang = 'en';
  document.getElementById('title').textContent = 'Opening in XBrowser';
  document.getElementById('msg').textContent = 'If nothing happens, press the button below.';
  document.getElementById('open').textContent = 'Open in XBrowser';
  document.getElementById('close').textContent = 'Close';
}

function launch() {
  if (valid) location.href = target;
}

document.getElementById('open').addEventListener('click', launch);
document.getElementById('close').addEventListener('click', async () => {
  try {
    const tab = await api.tabs.getCurrent();
    await api.tabs.remove(tab.id);
  } catch (e) {
    window.close();
  }
});

if (valid) setTimeout(launch, 100);
