// Message list pane — grouped rows, filters, paging, quick actions.

import { h, icon, iconEl, initials, avatarClass, fmtListDate, dateGroupLabel, displayName, debounce, $ } from './util.js';
import { state, on, emit, set } from './store.js';
import { api } from './api.js';

const listScroll = () => $('#listScroll');

export async function loadList(reset = true) {
  if (reset) {
    set({ messages: [], listPage: 1, listLoading: true, listError: null, listHasMore: false, selection: null, message: null }, 'list');
    renderList();
  }
  const { view, filter, search } = state;
  const page = state.listPage;
  try {
    let data;
    if (view.type === 'unified') {
      data = await api.unified(view.kind, { search, filter });
      data.messages = data.messages || [];
    } else {
      data = await api.messages(view.accountId, view.path, { page, search, filter });
    }
    const messages = reset ? data.messages : state.messages.concat(data.messages);
    set({
      messages,
      listTotal: data.total,
      listLoading: false,
      listHasMore: messages.length < (data.total || 0),
    }, 'list');
    renderList();
    prefetchSnippets(messages);
  } catch (err) {
    set({ listLoading: false, listError: err.message }, 'list');
    renderList();
  }
}

function prefetchSnippets(messages) {
  const byAccount = new Map();
  for (const m of messages) {
    if (m.snippet || !m.accountId) continue;
    const acc = state.accounts.find((a) => a.id === m.accountId);
    if (!acc) continue;
    if (!byAccount.has(m.accountId)) byAccount.set(m.accountId, { path: m.path, uids: [] });
    const g = byAccount.get(m.accountId);
    if (g.path === m.path && g.uids.length < 20) g.uids.push(m.uid);
  }
  for (const [accountId, g] of byAccount) {
    if (g.uids.length) api.prefetch(accountId, g.path, g.uids).catch(() => {});
  }
}

// -------------------------------------------------------------- rendering

export function renderList() {
  updateHead();
  const root = listScroll();
  root.innerHTML = '';

  if (state.listLoading && !state.messages.length) {
    root.append(...skeletonRows(8));
    return;
  }
  if (state.listError) {
    root.append(h('div', { class: 'list-error' },
      iconEl('junk', 30),
      h('div', { style: { fontWeight: 600, marginTop: 8 } }, '加载失败'),
      h('div', { style: { fontSize: 12.5, marginTop: 4 } }, state.listError),
      h('button', { class: 'btn small', style: { marginTop: 12 }, onClick: () => loadList(true) }, '重试')
    ));
    return;
  }
  if (!state.messages.length) {
    root.append(state.accounts.length ? emptyState() : onboarding());
    return;
  }

  const frag = document.createDocumentFragment();
  let group = null;
  for (const m of state.messages) {
    const label = dateGroupLabel(m.date);
    if (label !== group) {
      group = label;
      frag.append(h('div', { class: 'date-group-label' }, label));
    }
    frag.append(messageRow(m));
  }
  root.append(frag);

  if (state.listHasMore) {
    root.append(h('button', { class: 'btn small load-more', onClick: () => {
      set({ listPage: state.listPage + 1 });
      loadList(false);
    } }, `加载更多 (已显示 ${state.messages.length}/${state.listTotal})`));
  }
}

function updateHead() {
  const { view, search, filter } = state;
  let title = view.name || '收件箱';
  let sub = '';
  if (search) {
    title = `搜索“${search}”`;
  }
  const total = state.listLoading && !state.messages.length ? '' : `${state.listTotal || state.messages.length} 封邮件`;
  sub = [total, view.type === 'folder' ? ownerName(view.accountId) : ''].filter(Boolean).join(' · ');
  $('#listTitle').textContent = title;
  $('#listSub').textContent = sub;
  document.querySelectorAll('#filterChips .chip').forEach((chip) => {
    chip.classList.toggle('active', chip.dataset.filter === filter);
  });
}

function ownerName(accountId) {
  const acc = state.accounts.find((a) => a.id === accountId);
  return acc ? acc.name : '';
}

function onboarding() {
  return h('div', { class: 'list-empty' },
    h('div', { class: 'empty-art', html: `<svg viewBox="0 0 132 96" fill="none"><rect x="26" y="10" width="80" height="72" rx="10" fill="var(--chip)"/><rect x="26" y="10" width="80" height="72" rx="10" stroke="var(--border-strong)" stroke-width="2"/><path d="M36 26l30 22 30-22" stroke="var(--accent)" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/><path d="M46 62h40M46 70h24" stroke="var(--text-3)" stroke-width="2.5" stroke-linecap="round"/></svg>` }),
    h('h3', {}, '欢迎来到 Gloss Mail'),
    h('p', {}, '添加你的第一个邮箱账号（QQ、Gmail、163、Outlook、iCloud 或自定义 IMAP），开始收发邮件。'),
    h('button', { class: 'btn primary', style: { marginTop: 12 }, onClick: () => emit('open-settings') }, '添加邮箱账号')
  );
}

function emptyState() {
  const isSearch = state.search || state.filter;
  return h('div', { class: 'list-empty' },
    h('div', { class: 'empty-art', html: `<svg viewBox="0 0 132 96" fill="none"><rect x="18" y="18" width="96" height="64" rx="8" fill="var(--surface-3)"/><rect x="18" y="18" width="96" height="64" rx="8" stroke="var(--border)" stroke-width="2"/><path d="M26 30l40 28 40-28" stroke="var(--text-3)" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/></svg>` }),
    h('h3', {}, isSearch ? '没有匹配的邮件' : '这里空空如也'),
    h('p', {}, isSearch ? '换个关键词，或清除筛选条件再试试。' : '新的邮件会第一时间出现在这里。'),
    isSearch ? h('button', {
      class: 'btn small subtle', style: { marginTop: 10 }, onClick: () => {
        set({ search: '', filter: '' });
        $('#searchInput').value = '';
        loadList(true);
      },
    }, '清除筛选') : null
  );
}

function skeletonRows(n) {
  const widths = ['62%', '48%', '71%', '55%'];
  return Array.from({ length: n }, (_, i) => {
    const wide = i % 3 === 0;
    return h('div', { class: 'skel-row' },
      h('div', { class: 'skel', style: { width: 38, height: 38, borderRadius: '50%', flex: 'none' } }),
      h('div', { style: { flex: 1 } },
        h('div', { class: 'skel', style: { width: widths[i % 4], height: 13, marginBottom: 8 } }),
        h('div', { class: 'skel', style: { width: wide ? '88%' : '70%', height: 13, marginBottom: 6 } }),
        wide ? h('div', { class: 'skel', style: { width: '55%', height: 11 } }) : null
      )
    );
  });
}

function messageRow(m) {
  const selected = state.selection && state.selection.uid === m.uid && state.selection.accountId === m.accountId;
  const sender = m.from || { name: '未知', email: '' };
  const account = m.accountId ? state.accounts.find((a) => a.id === m.accountId) : null;

  const row = h('div', {
    class: `msg-row${m.flags && !m.flags.seen ? ' unread' : ''}${selected ? ' selected' : ''}`,
    dataset: { uid: m.uid, accountId: m.accountId || '' },
    onClick: () => emit('open-message', m),
    onContextMenu: (e) => { e.preventDefault(); rowMenu(e, m); },
  },
    h('span', { class: `avatar ${avatarClass(sender.email || sender.name)}`, text: initials(sender.name || sender.email) }),
    h('div', { class: 'mr-top' },
      h('span', { class: 'mr-sender', text: displayName(sender) }),
      m.flags && m.flags.answered ? h('span', { class: 'mr-badge', title: '已回复', html: icon('reply', 13) }) : null,
      h('span', { class: 'mr-date', text: fmtListDate(m.date) })
    ),
    h('div', {},
      h('div', { class: 'mr-subject' }, m.subject || '（无主题）'),
      h('div', { class: 'mr-bottom' },
        h('span', { class: 'mr-snippet', text: m.snippet || '' }),
        h('span', { class: 'mr-badges' },
          m.accountId && state.view.type === 'unified' && account
            ? h('span', { class: 'mr-acct-chip', style: { background: account.color || '#7c6cf5' }, text: account.name })
            : null,
          m.hasAttachments ? h('span', { class: 'mr-badge', title: '含附件', html: icon('paperclip', 13) }) : null,
          starBtn(m)
        )
      )
    )
  );
  return row;
}

function starBtn(m) {
  const active = !!(m.flags && m.flags.flagged);
  const btn = h('button', {
    class: `mr-star${active ? ' active' : ''}`,
    title: active ? '取消星标' : '加星标',
    html: icon('star', 15, active),
    onClick: (e) => {
      e.stopPropagation();
      toggleStar(m);
    },
  });
  return btn;
}

// ------------------------------------------------------------- actions

export function toggleStar(m) {
  const next = !(m.flags && m.flags.flagged);
  m.flags = { ...(m.flags || {}), flagged: next };
  const sel = state.selection;
  if (sel && sel.uid === m.uid && sel.accountId === m.accountId) {
    set({ selection: { ...sel, flags: m.flags } });
  }
  if (state.view.type === 'unified' && state.view.kind === 'flagged' && !next) {
    set({ messages: state.messages.filter((x) => !(x.uid === m.uid && x.accountId === m.accountId)), listTotal: state.listTotal - 1 }, 'list');
  }
  renderList();
  api.setFlags(m.accountId, m.path, [m.uid], { flagged: next })
    .then(() => emit('flags-changed', { accountId: m.accountId, uid: m.uid, flags: m.flags }))
    .catch(() => { /* optimistic */ });
}

function rowMenu(e, m) {
  document.querySelectorAll('.ctx-menu').forEach((n) => n.remove());
  const menu = h('div', { class: 'ctx-menu', style: { left: `${Math.min(e.clientX, innerWidth - 200)}px`, top: `${Math.min(e.clientY, innerHeight - 260)}px` } },
    ctxItem('star', m.flags && m.flags.flagged ? '取消星标' : '加星标', () => toggleStar(m)),
    ctxItem('mail', m.flags && m.flags.seen ? '标为未读' : '标为已读', () => markRead(m, !(m.flags && m.flags.seen))),
    ctxItem('archive', '归档', () => archiveMsg(m)),
    ctxItem('trash', '删除', () => deleteMsg(m), true)
  );
  document.body.append(menu);
  setTimeout(() => {
    const close = (ev) => {
      if (!menu.contains(ev.target)) { menu.remove(); document.removeEventListener('mousedown', close); }
    };
    document.addEventListener('mousedown', close);
  });
}

function ctxItem(ic, label, fn, danger = false) {
  return h('button', { class: `ctx-item${danger ? ' danger' : ''}`, html: icon(ic, 15) + `<span>${label}</span>`, onClick: () => { document.querySelectorAll('.ctx-menu').forEach((n) => n.remove()); fn(); } });
}

export function markRead(m, seen = true) {
  m.flags = { ...(m.flags || {}), seen };
  renderList();
  emit('flags-changed', { accountId: m.accountId, uid: m.uid, flags: m.flags });
  api.setFlags(m.accountId, m.path, [m.uid], { seen }).catch(() => {});
}

export function deleteMsg(m) {
  const uidKey = m.uid;
  api.deleteMessages(m.accountId, m.path, [m.uid])
    .then(() => {
      set({ messages: state.messages.filter((x) => !(x.uid === uidKey && x.accountId === m.accountId)), listTotal: Math.max(0, state.listTotal - 1) }, 'list');
      if (state.selection && state.selection.uid === m.uid && state.selection.accountId === m.accountId) {
        set({ selection: null, message: null });
      }
      renderList();
      emit('reader-reset');
      emit('flags-changed', { accountId: m.accountId, uid: m.uid, deleted: true });
      import('./util.js').then(({ toast }) => toast('已移到废纸篓', { type: 'ok' }));
    })
    .catch((err) => import('./util.js').then(({ toast }) => toast(err.message, { type: 'err' })));
}

export function archiveMsg(m) {
  api.archive(m.accountId, m.path, [m.uid])
    .then(() => {
      set({ messages: state.messages.filter((x) => !(x.uid === m.uid && x.accountId === m.accountId)), listTotal: Math.max(0, state.listTotal - 1) }, 'list');
      if (state.selection && state.selection.uid === m.uid && state.selection.accountId === m.accountId) {
        set({ selection: null, message: null });
      }
      renderList();
      emit('reader-reset');
      import('./util.js').then(({ toast }) => toast('已归档', { type: 'ok' }));
    })
    .catch((err) => import('./util.js').then(({ toast }) => toast('归档失败：' + err.message, { type: 'err' })));
}

// ---------------------------------------------------------- live updates

on('message-updated', ({ accountId, uid, snippet, hasAttachments }) => {
  const m = state.messages.find((x) => x.uid === uid && x.accountId === accountId);
  if (m) {
    m.snippet = snippet || m.snippet;
    m.hasAttachments = hasAttachments || m.hasAttachments;
    const row = listScroll().querySelector(`.msg-row[data-uid="${uid}"][data-account-id="${accountId}"] .mr-snippet`);
    if (row) row.textContent = m.snippet;
  }
});

// keyboard navigation helpers
export function moveSelection(delta) {
  const idx = state.messages.findIndex((m) => state.selection && m.uid === state.selection.uid && m.accountId === state.selection.accountId);
  let next = idx + delta;
  if (idx === -1) next = delta > 0 ? 0 : state.messages.length - 1;
  next = Math.max(0, Math.min(state.messages.length - 1, next));
  const target = state.messages[next];
  if (target) emit('open-message', target);
}

export const searchInputDebounced = debounce((val) => {
  set({ search: val });
  loadList(true);
}, 260);
