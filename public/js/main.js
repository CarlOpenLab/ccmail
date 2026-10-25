// Gloss Mail — bootstrap & wiring.

import { h, icon, iconEl, initials, avatarClass, toast, debounce, $ } from './util.js';
import { state, on, emit, set, applyTheme, accountById } from './store.js';
import { api, openStream, setBase, isDesktop } from './api.js';
import { renderSidebar, refreshMailboxes } from './sidebar.js';
import { loadList, renderList, searchInputDebounced, moveSelection, toggleStar, markRead, deleteMsg, archiveMsg } from './list.js';
import { renderReader } from './reader.js';
import { openSheet, initCompose } from './compose.js';
import { openSettings } from './settings.js';

// ------------------------------------------------------------------ boot

async function boot() {
  applyTheme();
  wireTitlebar();
  wireShortcuts();
  wireListFrost();

  if (isDesktop()) {
    document.body.classList.add('desktop');
    try {
      const base = await window.mygo.call('Desktop.APIBase');
      setBase(base);
      try {
        const b = await api.bootstrap();
        window.mygo.call('Desktop.Diag', `base=${base} bootstrap ok accounts=${b.accounts.length}`);
      } catch (err) {
        window.mygo.call('Desktop.Diag', `base=${base} bootstrap FAIL ${err.message}`);
      }
    } catch (err) {
      console.error('[mygo] APIBase failed', err);
    }
    renderWindowControls();
  }

  try {
    const [{ accounts }, { providers }] = await Promise.all([api.bootstrap(), api.providers()]);
    set({ accounts, providers: providers }, 'accounts');
  } catch (err) {
    toast('无法连接后端服务：' + err.message, { type: 'err', duration: 8000 });
  }

  renderSidebar();
  renderAvatarStack();
  renderList();
  renderReader();
  initCompose();
  openStream(streamHandlers());

  const matchDark = matchMedia('(prefers-color-scheme: dark)');
  matchDark.addEventListener('change', () => { if (state.theme === 'auto') applyTheme(); });

  // initial view
  emit('navigate', { type: 'unified', kind: 'inbox', name: '全部收件箱' });
}

// ------------------------------------------------------------- navigation

on('navigate', (view) => {
  set({ view, selection: null, message: null, listPage: 1 });
  document.getElementById('main').classList.remove('reader-open');
  document.getElementById('app').classList.remove('sidebar-open');
  loadList(true);
  renderReader();
  if (view.type === 'folder' && view.accountId) refreshMailboxes(view.accountId);
});

async function refreshAccounts() {
  try {
    const { accounts } = await api.bootstrap();
    set({ accounts }, 'accounts');
    renderSidebar();
    renderAvatarStack();
    refreshMailboxes();
  } catch { /* offline */ }
}

on('accounts-refresh', refreshAccounts);

// ------------------------------------------------------------------- SSE

function streamHandlers() {
  return {
    hello: ({ accounts }) => {
      set({ accounts }, 'accounts');
      renderSidebar();
      renderAvatarStack();
    },
    'account:status': ({ accountId, status, detail }) => {
      const acc = accountById(accountId);
      if (!acc) return;
      acc.status = status;
      if (detail) acc.statusDetail = detail;
      renderSidebar();
      renderAvatarStack();
    },
    'mail:counts': ({ accountId, unread, total }) => {
      const prev = state.unreadByAccount[accountId] || 0;
      const acc = accountById(accountId);
      if (unread > prev && acc) {
        toast(`「${acc.name}」收到新邮件`, { type: 'info' });
      }
      state.unreadByAccount[accountId] = unread;
      // refresh the open list if it shows this account's inbox
      const v = state.view;
      if ((v.type === 'unified') || (v.type === 'folder' && v.accountId === accountId && /inbox/i.test(v.path))) {
        debounceRefresh();
      }
      renderSidebar();
    },
    'accounts:changed': () => refreshAccounts(),
    'message:updated': (data) => emit('message-updated', data),
  };
}

const debounceRefresh = debounce(() => {
  if (!state.listLoading) loadList(false); // silent refresh, keeps selection
}, 1500);

on('flags-changed', debounce(() => {
  refreshMailboxes();
}, 800));

on('mail-sent', debounce(() => {
  refreshMailboxes();
  const v = state.view;
  if (v.type === 'folder' && (v.kind === 'sent' || v.kind === 'inbox')) loadList(true);
}, 600));

// -------------------------------------------------------------- titlebar

function wireTitlebar() {
  const search = $('#searchInput');
  search.addEventListener('input', () => {
    $('#searchClear').hidden = !search.value;
    const kbd = document.querySelector('.search-kbd');
    if (kbd) kbd.style.display = search.value ? 'none' : '';
    searchInputDebounced(search.value.trim());
  });
  search.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      search.value = '';
      $('#searchClear').hidden = true;
      searchInputDebounced('');
      search.blur();
    }
  });
  $('#searchClear').addEventListener('click', () => {
    search.value = '';
    $('#searchClear').hidden = true;
    searchInputDebounced('');
    search.focus();
  });

  $('#btnRefresh').innerHTML = icon('refresh', 17);
  $('#btnRefresh').addEventListener('click', async () => {
    const btn = $('#btnRefresh');
    btn.classList.add('spinning');
    await Promise.all([loadList(true), refreshMailboxes()]);
    setTimeout(() => btn.classList.remove('spinning'), 400);
  });

  $('#btnTheme').innerHTML = icon('sun', 17);
  $('#btnTheme').addEventListener('click', () => {
    const next = applyTheme() === 'dark' ? 'light' : 'dark';
    set({ theme: next });
    applyTheme();
    localStorage.setItem('gloss.theme', next);
    $('#btnTheme').innerHTML = icon(next === 'dark' ? 'sun' : 'moon', 17);
  });
  $('#btnTheme').innerHTML = icon(applyTheme() === 'dark' ? 'sun' : 'moon', 17);

  $('#btnSettings').innerHTML = icon('settings', 17);
  $('#btnSettings').addEventListener('click', () => openSettings());

  $('#btnCompose').addEventListener('click', () => emit('compose', { mode: 'new' }));

  $('#btnSidebar').innerHTML = icon('menu', 18);
  $('#btnSidebar').addEventListener('click', () => {
    document.getElementById('app').classList.toggle('sidebar-open');
  });
  $('#sidebarBackdrop').addEventListener('click', () => {
    document.getElementById('app').classList.remove('sidebar-open');
  });

  $('#btnBackReader').innerHTML = icon('back', 18);
  $('#btnBackReader').addEventListener('click', () => {
    document.getElementById('main').classList.remove('reader-open');
  });

  $('#btnListMore').innerHTML = icon('more', 17);
  $('#btnListMore').addEventListener('click', (e) => {
    listMoreMenu(e.currentTarget);
  });

  $('#brandHome').addEventListener('click', (e) => {
    e.preventDefault();
    emit('navigate', { type: 'unified', kind: 'inbox', name: '全部收件箱' });
  });

  document.querySelectorAll('#filterChips .chip').forEach((chip) => {
    chip.addEventListener('click', () => {
      set({ filter: chip.dataset.filter });
      loadList(true);
    });
  });
}

function listMoreMenu(anchor) {
  document.querySelectorAll('.ctx-menu').forEach((n) => n.remove());
  const rect = anchor.getBoundingClientRect();
  const menu = h('div', { class: 'ctx-menu', style: { right: `${innerWidth - rect.right}px`, top: `${rect.bottom + 6}px`, left: 'auto' } },
    h('button', {
      class: 'ctx-item', html: icon('refresh', 15) + '<span>刷新列表</span>',
      onClick: () => { menu.remove(); loadList(true); },
    }),
    h('button', {
      class: 'ctx-item', html: icon('check', 15) + '<span>全部标为已读</span>',
      onClick: async () => {
        menu.remove();
        const targets = state.messages.filter((m) => m.flags && !m.flags.seen);
        for (const m of targets) markRead(m, true);
        const byAccount = new Map();
        for (const m of targets) {
          if (!byAccount.has(m.accountId)) byAccount.set(m.accountId, { path: m.path, uids: [] });
          byAccount.get(m.accountId).uids.push(m.uid);
        }
        for (const [accountId, g] of byAccount) {
          api.setFlags(accountId, g.path, g.uids, { seen: true }).catch(() => {});
        }
        refreshMailboxes();
      },
    }),
    h('button', {
      class: 'ctx-item', html: icon('inbox', 15) + '<span>回到全部收件箱</span>',
      onClick: () => { menu.remove(); emit('navigate', { type: 'unified', kind: 'inbox', name: '全部收件箱' }); },
    })
  );
  document.body.append(menu);
  setTimeout(() => {
    const close = (ev) => {
      if (!menu.contains(ev.target)) { menu.remove(); document.removeEventListener('mousedown', close); }
    };
    document.addEventListener('mousedown', close);
  });
}

// ------------------------------------------------------ list card frost

// The list header floats above the scroll area; when the list scrolls it
// frosts (blur + hairline) like a macOS card header.
function wireListFrost() {
  const head = document.querySelector('.list-head');
  const scroll = document.getElementById('listScroll');
  if (!head || !scroll) return;

  const relayout = () => {
    scroll.style.paddingTop = `${head.offsetHeight + 6}px`;
  };
  relayout();
  window.addEventListener('resize', relayout);
  if (window.ResizeObserver) new ResizeObserver(relayout).observe(head);

  scroll.addEventListener('scroll', () => {
    head.classList.toggle('scrolled', scroll.scrollTop > 4);
  }, { passive: true });
}

// ------------------------------------------------------ desktop window

// Windows-frame style flat controls drawn by the page (mygo Frameless).
// The raw injected runtime exposes controls at mygo.window.
function renderWindowControls() {
  const win = window.mygo && window.mygo.window;
  if (!win) return;
  const btn = (glyph, title, onClick, cls = '') => h('button', {
    class: `win-btn ${cls}`, title,
    html: glyph,
    onClick,
  });
  $('.tb-right').append(h('div', { class: 'win-controls' },
    btn('<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M5 12h14"/></svg>', '最小化', () => win.minimize()),
    btn('<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linejoin="round"><rect x="5" y="5" width="14" height="14" rx="2"/></svg>', '最大化 / 还原', () => win.toggleMaximize()),
    btn('<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>', '关闭', () => win.close(), 'close')
  ));
}

// ----------------------------------------------------------- avatar stack

function renderAvatarStack() {
  const root = $('#avatarStack');
  root.innerHTML = '';
  for (const account of state.accounts.slice(0, 4)) {
    root.append(h('span', {
      class: `avatar ${avatarClass(account.email)}`,
      style: { background: account.color },
      text: initials(account.name || account.email),
      title: `${account.name}（${account.email}）`,
      onClick: () => emit('navigate', { type: 'folder', accountId: account.id, path: 'INBOX', name: '收件箱', kind: 'inbox' }),
    }));
  }
  root.append(h('span', {
    class: 'avatar add-avatar', text: '+', title: '添加账号',
    onClick: () => openSettings('accounts'),
  }));
}

// ------------------------------------------------------------- shortcuts

function wireShortcuts() {
  document.addEventListener('keydown', (e) => {
    const inField = /INPUT|TEXTAREA|SELECT/.test(document.activeElement && document.activeElement.tagName)
      || (document.activeElement && document.activeElement.isContentEditable);
    const modalOpen = !!document.querySelector('.modal-backdrop');

    if (e.key === '/' && !inField) {
      e.preventDefault();
      $('#searchInput').focus();
      return;
    }
    if (e.key === '?' && !inField && !modalOpen) {
      e.preventDefault();
      openSettings('shortcuts');
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'b') {
      e.preventDefault();
      document.getElementById('app').classList.toggle('sidebar-open');
      return;
    }
    if (modalOpen || inField) return;

    const sel = state.selection;
    switch (e.key.toLowerCase()) {
      case 'j': case 'arrowdown':
        e.preventDefault(); moveSelection(1); break;
      case 'k': case 'arrowup':
        e.preventDefault(); moveSelection(-1); break;
      case 'enter': case 'o':
        if (sel) { e.preventDefault(); emit('open-message', sel); }
        break;
      case 'n': case 'c':
        e.preventDefault(); emit('compose', { mode: 'new' }); break;
      case 'r':
        if (state.message) { e.preventDefault(); emit('compose', { mode: 'reply', message: state.message, envelope: sel }); }
        break;
      case 'a':
        if (state.message) { e.preventDefault(); emit('compose', { mode: 'replyAll', message: state.message, envelope: sel }); }
        break;
      case 'f':
        if (state.message) { e.preventDefault(); emit('compose', { mode: 'forward', message: state.message, envelope: sel }); }
        break;
      case 's': case 'x':
        if (sel) { e.preventDefault(); toggleStar(sel); renderReader(); }
        break;
      case 'e':
        if (sel) { e.preventDefault(); archiveMsg(sel); }
        break;
      case 'delete': case 'backspace':
        if (sel) { e.preventDefault(); deleteMsg(sel); }
        break;
      case 'u':
        document.getElementById('main').classList.remove('reader-open');
        break;
      default:
        break;
    }
  });
}

boot();
