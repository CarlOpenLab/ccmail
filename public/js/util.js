// Small helpers: DOM builder, icons, formatting, toasts.

export function h(tag, props = {}, ...children) {
  const el = document.createElement(tag);
  for (const [key, val] of Object.entries(props || {})) {
    if (val === null || val === undefined || val === false) continue;
    if (key === 'class') el.className = val;
    else if (key === 'html') el.innerHTML = val;
    else if (key === 'text') el.textContent = val;
    else if (key === 'style' && typeof val === 'object') Object.assign(el.style, val);
    else if (key === 'dataset') Object.assign(el.dataset, val);
    else if (key.startsWith('on') && typeof val === 'function') el.addEventListener(key.slice(2).toLowerCase(), val);
    else if (key in el && key !== 'list' && key !== 'form' && key !== 'type' && !key.startsWith('attr:')) el[key] = val;
    else el.setAttribute(key === 'attr:href' ? 'href' : key.replace(/^attr:/, ''), val);
  }
  for (const child of children.flat(Infinity)) {
    if (child === null || child === undefined || child === false) continue;
    el.append(child.nodeType ? child : document.createTextNode(child));
  }
  return el;
}

export const $ = (sel, root = document) => root.querySelector(sel);

// ------------------------------------------------------------------ icons

const P = (d, extra = '') => `<path d="${d}" ${extra}/>`;

const ICONS = {
  inbox: P('M22 12h-6l-2 3h-4l-2-3H2') + P('M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z'),
  star: P('m12 2 3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01z'),
  send: P('m22 2-7 20-4-9-9-4z') + P('M22 2 11 13'),
  draft: P('M12 20h9') + P('M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4z'),
  trash: P('M3 6h18') + P('M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6') + P('M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2') + P('M10 11v6', 'stroke-width="2"') + P('M14 11v6', 'stroke-width="2"'),
  archive: P('M21 8v13H3V8') + P('M1 3h22v5H1z') + P('M10 12h4'),
  folder: P('M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z'),
  junk: P('M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z') + P('M12 9v4') + P('M12 17h.01'),
  alert: P('M22 12h-4l-3 9L9 3l-3 9H2'),
  search: '<circle cx="11" cy="11" r="7"/>' + P('m20 20-3.8-3.8'),
  refresh: P('M23 4v6h-6') + P('M1 20v-6h6') + P('M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15'),
  settings: '<circle cx="12" cy="12" r="3"/>' + P('M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z'),
  sun: '<circle cx="12" cy="12" r="4"/>' + P('M12 2v2') + P('M12 20v2') + P('m4.93 4.93 1.41 1.41') + P('m17.66 17.66 1.41 1.41') + P('M2 12h2') + P('M20 12h2') + P('m6.34 17.66-1.41 1.41') + P('m19.07 4.93-1.41 1.41'),
  moon: P('M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z'),
  chevron: P('m6 9 6 6 6-6'),
  plus: P('M12 5v14') + P('M5 12h14'),
  close: P('M18 6 6 18') + P('m6 6 12 12'),
  check: P('M20 6 9 17l-5-5'),
  mail: '<rect x="2" y="4" width="20" height="16" rx="2"/>' + P('m22 6-10 7L2 6'),
  reply: P('M9 17 4 12l5-5') + P('M20 18v-2a4 4 0 0 0-4-4H4'),
  replyAll: P('m7 17-5-5 5-5') + P('m12 17-5-5 5-5') + P('M22 18v-2a4 4 0 0 0-4-4H6'),
  forward: P('m15 17 5-5-5-5') + P('M4 18v-2a4 4 0 0 1 4-4h12'),
  paperclip: P('m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48'),
  image: '<rect x="3" y="3" width="18" height="18" rx="2"/>' + '<circle cx="8.5" cy="8.5" r="1.5"/>' + P('m21 15-5-5L5 21'),
  download: P('M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4') + P('m7 10 5 5 5-5') + P('M12 15V3'),
  more: '<circle cx="5" cy="12" r="1.7" fill="currentColor" stroke="none"/>' + '<circle cx="12" cy="12" r="1.7" fill="currentColor" stroke="none"/>' + '<circle cx="19" cy="12" r="1.7" fill="currentColor" stroke="none"/>',
  edit: P('M12 20h9') + P('M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z'),
  user: P('M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2') + '<circle cx="12" cy="7" r="4"/>',
  bold: P('M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z') + P('M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z'),
  italic: P('M19 4h-9') + P('M14 20H5') + P('M15 4 9 20'),
  underline: P('M6 3v7a6 6 0 0 0 12 0V3') + P('M4 21h16'),
  listUl: P('M8 6h13') + P('M8 12h13') + P('M8 18h13') + P('M3 6h.01') + P('M3 12h.01') + P('M3 18h.01'),
  listOl: P('M10 6h11') + P('M10 12h11') + P('M10 18h11') + P('M4 6h1v4') + P('M4 10h2') + P('m3 18 1.5-2.5L3 14h3'),
  link: P('M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71') + P('M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71'),
  quote: P('M6 17h3l2-4V7H5v6h3zM14 17h3l2-4V7h-6v6h3z'),
  eye: P('M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z') + '<circle cx="12" cy="12" r="3"/>',
  eyeOff: P('M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94') + P('M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19') + P('M14.12 14.12a3 3 0 1 1-4.24-4.24') + P('m1 1 22 22'),
  menu: P('M3 12h18') + P('M3 6h18') + P('M3 18h18'),
  back: P('m15 18-6-6 6-6'),
  sparkle: P('m12 3 1.9 6.1L20 11l-6.1 1.9L12 19l-1.9-6.1L4 11l6.1-1.9z'),
  filter: P('M22 3H2l8 9.46V19l4 2v-8.54z'),
};

export function icon(name, size = 18, filled = false) {
  const body = ICONS[name] || ICONS.mail;
  return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="${filled ? 'currentColor' : 'none'}" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${body}</svg>`;
}

export function iconEl(name, size, filled) {
  const t = document.createElement('template');
  t.innerHTML = icon(name, size, filled).trim();
  return t.content.firstChild;
}

// ------------------------------------------------------------- formatting

export function initials(nameOrEmail) {
  const s = String(nameOrEmail || '?').trim();
  if (!s) return '?';
  if (/@/.test(s)) {
    const [name, domain] = s.split('@');
    if (/^[a-z]/i.test(domain) && name) {
      // brand-ish: use domain capital e.g. github
      const brand = domain.split('.')[0];
      return brand.slice(0, 2).toUpperCase();
    }
  }
  const code = s.codePointAt(0);
  const isCJK = (code >= 0x4e00 && code <= 0x9fff) || (code >= 0x3040 && code <= 0x30ff);
  if (isCJK) return String.fromCodePoint(code);
  const parts = s.split(/[\s._-]+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return s.slice(0, 2).toUpperCase();
}

export function avatarClass(seedStr) {
  let hash = 0;
  const s = String(seedStr || '').toLowerCase();
  for (let i = 0; i < s.length; i++) hash = (hash * 31 + s.charCodeAt(i)) >>> 0;
  return `c${hash % 10}`;
}

const DAY_MS = 86400000;

function startOfDay(d) {
  const x = new Date(d);
  x.setHours(0, 0, 0, 0);
  return x;
}

export function fmtListDate(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  const now = new Date();
  const days = Math.round((startOfDay(now) - startOfDay(d)) / DAY_MS);
  const hm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
  if (days === 0) return hm;
  if (days === 1) return '昨天';
  if (days < 7 && days > 0) return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.getDay()];
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日`;
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
}

export function dateGroupLabel(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '更早';
  const now = new Date();
  const days = Math.round((startOfDay(now) - startOfDay(d)) / DAY_MS);
  if (days <= 0) return '今天';
  if (days === 1) return '昨天';
  if (days < 7) return '本周';
  if (days < 30) return '本月';
  return '更早';
}

export function fmtFullDate(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  const now = new Date();
  const hm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
  const days = Math.round((startOfDay(now) - startOfDay(d)) / DAY_MS);
  const base = `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
  if (days === 0) return `今天 ${hm}`;
  if (days === 1) return `昨天 ${hm}`;
  return `${base} ${hm}`;
}

export function fmtSize(bytes) {
  if (!bytes && bytes !== 0) return '';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function displayName(addr) {
  if (!addr) return '未知发件人';
  return addr.name || (addr.email ? addr.email.split('@')[0] : '未知');
}

export function debounce(fn, ms) {
  let t;
  return (...args) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  };
}

// ------------------------------------------------------------------ toast

export function toast(message, { type = 'info', action, onAction, duration = 3600 } = {}) {
  const root = document.getElementById('toastRoot');
  const iconName = type === 'ok' ? 'check' : type === 'err' ? 'junk' : 'sparkle';
  const el = h('div', { class: 'toast' },
    h('span', { class: `t-icon ${type}`, html: icon(iconName, 15, type === 'ok') }),
    h('span', { text: message }),
    action && h('button', { class: 't-action', text: action, onClick: () => { onAction && onAction(); dismiss(); } })
  );
  root.append(el);
  const dismiss = () => {
    el.classList.add('out');
    setTimeout(() => el.remove(), 240);
  };
  setTimeout(dismiss, duration);
  return dismiss;
}

// --------------------------------------------------------------- sanitize

export function sanitizeEmailHtml(html, { attachmentUrl, accountId, path, uid, allowImages = false } = {}) {
  const doc = new DOMParser().parseFromString(`<!doctype html><body>${html}</body>`, 'text/html');
  const body = doc.body;

  // strip dangerous elements
  for (const t of ['script', 'iframe', 'object', 'embed', 'link', 'meta', 'base', 'form']) {
    body.querySelectorAll(t).forEach((n) => n.remove());
  }

  // strip event handlers & javascript: urls
  body.querySelectorAll('*').forEach((el) => {
    for (const attr of [...el.attributes]) {
      const name = attr.name.toLowerCase();
      const val = String(attr.value || '').trim().toLowerCase();
      if (name.startsWith('on')) el.removeAttribute(attr.name);
      else if ((name === 'href' || name === 'src') && (val.startsWith('javascript:') || val.startsWith('data:text/html'))) {
        el.removeAttribute(attr.name);
      }
    }
  });

  let blockedImages = 0;
  const cidResolve = typeof attachmentUrl === 'function'
    ? attachmentUrl
    : attachmentUrl ? ((cid) => attachmentUrl + encodeURIComponent(cid)) : null;

  body.querySelectorAll('img').forEach((img) => {
    const src = img.getAttribute('src') || '';
    if (src.startsWith('cid:') && cidResolve) {
      const url = cidResolve(src.slice(4));
      if (url) {
        img.setAttribute('src', url);
        return;
      }
    }
    if (!src || src.startsWith('data:')) return;
    if (!allowImages) {
      blockedImages += 1;
      img.setAttribute('data-gloss-src', src);
      img.removeAttribute('src');
      if (!img.getAttribute('width')) img.setAttribute('width', Math.min(520, parseInt(img.getAttribute('width') || '0', 10) || 220));
      img.setAttribute('height', Math.max(40, parseInt(img.getAttribute('height') || '0', 10) || 90));
      img.style.background = 'rgba(109,90,252,0.08)';
      img.style.borderRadius = '8px';
    }
  });

  // absolute-ify relative links (common in newsletters)
  body.querySelectorAll('a[href]').forEach((a) => {
    const href = a.getAttribute('href');
    if (href && !/^(https?:|mailto:|#)/i.test(href)) a.setAttribute('href', '#' + href);
    a.setAttribute('target', '_blank');
    a.setAttribute('rel', 'noreferrer noopener');
  });

  return { html: body.innerHTML, blockedImages };
}

export const emailCss = `
  <style>
    :root { color-scheme: light; }
    html, body { margin: 0; padding: 0; background: #fff; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei UI", sans-serif;
      color: #1d1d21;
      font-size: 14.5px;
      line-height: 1.75;
      word-break: break-word;
    }
    .gloss-pad { padding: 26px 30px 34px; max-width: 680px; margin: 0 auto; }
    img { max-width: 100% !important; height: auto !important; }
    table { max-width: 100% !important; }
    a { color: #5846e8; }
    blockquote { margin: 1em 0; padding: 2px 14px; border-left: 3px solid #d9d9e0; color: #67677a; }
    pre { overflow-x: auto; background: #f6f6f9; padding: 12px 14px; border-radius: 10px; font-size: 13px; }
  </style>
`;
