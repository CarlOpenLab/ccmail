// REST + push client for the Gloss Mail backend.
// Two modes: inside a mygo window (API over the custom "api" protocol and
// push over a mygo event) or plain browser (relative URLs + SSE).

let baseUrl = '';

export function setBase(base) {
  baseUrl = base || '';
}

export function isDesktop() {
  return !!(window.mygo && (!window.mygo.isMyGo || window.mygo.isMyGo()));
}

async function request(method, url, body) {
  const res = await fetch(baseUrl + url, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  let data = null;
  try {
    data = await res.json();
  } catch { /* empty body */ }
  if (!res.ok) {
    const err = new Error((data && data.error) || `请求失败 (${res.status})`);
    err.status = res.status;
    throw err;
  }
  return data;
}

export const api = {
  bootstrap: () => request('GET', '/api/bootstrap'),
  providers: () => request('GET', '/api/providers'),

  addAccount: (payload) => request('POST', '/api/accounts', payload),
  patchAccount: (id, patch) => request('PATCH', `/api/accounts/${id}`, patch),
  deleteAccount: (id) => request('DELETE', `/api/accounts/${id}`),

  mailboxes: (accountId) => request('GET', `/api/accounts/${accountId}/mailboxes`),
  messages: (accountId, path, { page = 1, search = '', filter = '' } = {}) => {
    const q = new URLSearchParams({ path, page, });
    if (search) q.set('search', search);
    if (filter) q.set('filter', filter);
    return request('GET', `/api/accounts/${accountId}/messages?${q}`);
  },
  message: (accountId, path, uid, { markSeen = true } = {}) =>
    request('GET', `/api/accounts/${accountId}/message?path=${encodeURIComponent(path)}&uid=${uid}${markSeen ? '' : '&mark=0'}`),
  setFlags: (accountId, path, uids, patch) => request('PATCH', `/api/accounts/${accountId}/flags`, { path, uids, ...patch }),
  move: (accountId, path, uids, dest) => request('POST', `/api/accounts/${accountId}/move`, { path, uids, dest }),
  deleteMessages: (accountId, path, uids) => request('POST', `/api/accounts/${accountId}/delete`, { path, uids }),
  archive: (accountId, path, uids) => request('POST', `/api/accounts/${accountId}/archive`, { path, uids }),
  prefetch: (accountId, path, uids) => request('POST', `/api/accounts/${accountId}/prefetch`, { path, uids }),

  unified: (kind, { search = '', filter = '' } = {}) => {
    const q = new URLSearchParams({ kind });
    if (search) q.set('search', search);
    if (filter) q.set('filter', filter);
    return request('GET', `/api/unified?${q}`);
  },

  send: (payload) => request('POST', '/api/send', payload),
  contacts: () => request('GET', '/api/contacts'),
  prefs: () => request('GET', '/api/prefs'),
  setPrefs: (values) => request('PUT', '/api/prefs', values),
  clearCache: () => request('POST', '/api/cache/clear'),

  attachmentUrl: (accountId, path, uid, attId) =>
    baseUrl + `/api/accounts/${accountId}/attachment?path=${encodeURIComponent(path)}&uid=${uid}&att=${encodeURIComponent(attId)}`,
};

// ---------------------------------------------------------------- push

export function openStream(handlers) {
  if (isDesktop() && window.mygo.on) {
    // mygo window: typed event "push" carries {"event": "...", "data": {...}}
    window.mygo.on('push', (p) => {
      if (p && handlers[p.event]) handlers[p.event](p.data);
    });
    return { close() { /* page lifetime */ } };
  }
  // browser / loopback mode: SSE
  const es = new EventSource(baseUrl + '/api/stream');
  for (const [event, fn] of Object.entries(handlers)) {
    es.addEventListener(event, (e) => {
      try {
        fn(JSON.parse(e.data || '{}'));
      } catch (err) {
        console.error('[sse]', event, err);
      }
    });
  }
  es.onerror = () => { /* browser retries automatically */ };
  return es;
}
