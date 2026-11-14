const api = globalThis.browser || globalThis.chrome;

const HOSTS = new Set(['x.com', 'www.x.com', 'twitter.com', 'www.twitter.com', 'mobile.twitter.com']);
const PATHS = new Set(['/intent/post', '/intent/tweet', '/share']);
const PARAMS = ['text', 'url', 'hashtags', 'via', 'related', 'in_reply_to'];

function shareTarget(raw) {
  let u;
  try {
    u = new URL(raw);
  } catch (e) {
    return null;
  }
  if (u.protocol !== 'https:' || !HOSTS.has(u.hostname)) return null;
  if (!PATHS.has(u.pathname.replace(/\/+$/, ''))) return null;
  const q = new URLSearchParams();
  PARAMS.forEach((k) => {
    const v = u.searchParams.get(k);
    if (v) q.set(k, v);
  });
  return 'xbrowser://share?' + q.toString();
}

async function closeIfPopup(tabId) {
  try {
    const tab = await api.tabs.get(tabId);
    const win = await api.windows.get(tab.windowId);
    if (win.type === 'popup' || tab.openerTabId !== undefined) await api.tabs.remove(tabId);
  } catch (e) {}
}

api.webNavigation.onBeforeNavigate.addListener(
  async (details) => {
    if (details.frameId !== 0) return;
    const target = shareTarget(details.url);
    if (!target) return;
    await api.tabs.update(details.tabId, { url: target });
    setTimeout(() => closeIfPopup(details.tabId), 1500);
  },
  { url: [{ pathPrefix: '/intent/' }, { pathPrefix: '/share' }] }
);

if (typeof module !== 'undefined') module.exports = { shareTarget };
