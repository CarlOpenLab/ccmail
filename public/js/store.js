// Central app state with a tiny subscription model.

const listeners = new Map();

export const state = {
  accounts: [],          // [{id,type,name,email,color,provider,status,...}]
  providers: [],
  mailboxesByAccount: {},// accountId -> [{path,name,kind,unread,messages,...}]
  view: { type: 'unified', kind: 'inbox', name: '全部收件箱' },
  filter: '',
  search: '',
  messages: [],          // current list envelopes (unified items carry accountId)
  listTotal: 0,
  listPage: 1,
  listHasMore: false,
  listLoading: false,
  listError: null,
  selection: null,       // envelope being read
  message: null,         // full parsed message
  messageLoading: false,
  unreadByAccount: {},   // accountId -> number (for toasts)
  theme: localStorage.getItem('gloss.theme') || 'auto',
  wash: localStorage.getItem('gloss.wash') || 'plain',
  density: localStorage.getItem('gloss.density') || 'cozy',
  markRead: localStorage.getItem('gloss.markRead') !== '0',
  autoImages: localStorage.getItem('gloss.autoImages') === '1',
  quoteReply: localStorage.getItem('gloss.quoteReply') !== '0',
  collapsed: JSON.parse(localStorage.getItem('gloss.collapsed') || '{}'),
};

export function on(topic, fn) {
  if (!listeners.has(topic)) listeners.set(topic, new Set());
  listeners.get(topic).add(fn);
  return () => listeners.get(topic).delete(fn);
}

export function emit(topic, payload) {
  const set = listeners.get(topic);
  if (set) for (const fn of [...set]) fn(payload);
}

export function set(patch, topic = 'change') {
  Object.assign(state, patch);
  emit(topic, patch);
}

export function applyTheme() {
  const resolved = state.theme === 'auto'
    ? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
    : state.theme;
  // theme class lives on both <html> (so modals/toasts outside #app inherit
  // the custom properties) and #app itself
  const root = document.documentElement;
  root.classList.remove('theme-light', 'theme-dark');
  root.classList.add(`theme-${resolved}`);
  const app = document.getElementById('app');
  app.className = `app theme-${resolved}`;
  const wash = state.wash && state.wash !== 'plain' ? state.wash : null;
  for (const el of [root, app]) {
    if (wash) el.dataset.wash = wash;
    else delete el.dataset.wash;
    el.dataset.density = state.density || 'cozy';
  }
  root.dataset.theme = resolved;
  return resolved;
}

export function persistPrefs() {
  localStorage.setItem('gloss.theme', state.theme);
  localStorage.setItem('gloss.wash', state.wash);
  localStorage.setItem('gloss.density', state.density);
  localStorage.setItem('gloss.markRead', state.markRead ? '1' : '0');
  localStorage.setItem('gloss.autoImages', state.autoImages ? '1' : '0');
  localStorage.setItem('gloss.quoteReply', state.quoteReply ? '1' : '0');
}

export function accountById(id) {
  return state.accounts.find((a) => a.id === id) || null;
}
