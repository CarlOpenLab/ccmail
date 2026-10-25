// Sidebar — smart views + mounted accounts with folders and unread badges.

import { h, icon, initials, avatarClass, $ } from './util.js';
import { state, on, emit, set } from './store.js';
import { api } from './api.js';

const KIND_ICON = {
  inbox: 'inbox', sent: 'send', drafts: 'draft', trash: 'trash',
  junk: 'junk', archive: 'archive', all: 'archive', flagged: 'star', custom: 'folder',
};

export function renderSidebar() {
  const root = $('#sidebar');
  root.innerHTML = '';

  root.append(h('div', { class: 'sidebar-scroll' },
    smartViews(),
    ...state.accounts.map((account) => accountGroup(account))
  ));

  root.append(h('div', { class: 'sidebar-foot' },
    h('button', { class: 'btn subtle', html: icon('plus', 14) + '<span>添加邮箱账号</span>', onClick: () => emit('open-settings') }),
    h('div', { class: 'foot-hint', text: 'Gloss Mail · IMAP / SMTP 多账号' })
  ));
}

function smartViews() {
  const v = state.view;
  const totalUnread = Object.values(state.unreadByAccount).reduce((s, n) => s + n, 0);
  return h('div', {},
    h('div', { class: 'side-section-label' }, '智能视图'),
    h('div', {},
      navRow({ icon: 'inbox', label: '全部收件箱', count: totalUnread, muted: false,
        active: v.type === 'unified' && v.kind === 'inbox',
        onClick: () => emit('navigate', { type: 'unified', kind: 'inbox', name: '全部收件箱' }) }),
      navRow({ icon: 'star', label: '星标邮件',
        active: v.type === 'unified' && v.kind === 'flagged',
        onClick: () => emit('navigate', { type: 'unified', kind: 'flagged', name: '星标邮件' }) }),
    )
  );
}

function navRow({ icon: ic, label, count, muted, active, onClick }) {
  return h('button', { class: `nav-row${active ? ' active' : ''}`, onClick },
    h('span', { class: 'nav-icon', html: icon(ic, 16, ic === 'star' && active) }),
    h('span', { class: 'nav-label', text: label }),
    count ? h('span', { class: `nav-count${muted ? ' muted' : ''}`, text: count > 99 ? '99+' : String(count) }) : null
  );
}

function accountGroup(account) {
  const boxes = state.mailboxesByAccount[account.id] || [];
  const unread = boxes.filter((b) => b.kind === 'inbox').reduce((s, b) => s + (b.unread || 0), 0);
  const isCollapsed = !!state.collapsed[account.id];
  const activeInAccount = state.view.type === 'folder' && state.view.accountId === account.id;
  const activeBox = activeInAccount ? state.view.path : null;

  return h('div', { class: `account-group${isCollapsed ? ' collapsed' : ''}`, dataset: { accountId: account.id } },
    h('button', {
      class: 'account-head',
      title: `${account.name} · ${statusText(account.status)}`,
      onClick: () => {
        const inbox = boxes.find((b) => b.kind === 'inbox');
        if (inbox && !isCollapsed) {
          emit('navigate', { type: 'folder', accountId: account.id, path: inbox.path, name: inbox.name, kind: 'inbox' });
        } else {
          toggleCollapse(account.id);
        }
      },
    },
      h('span', { class: 'acct-chevron', html: icon('chevron', 14) }),
      h('span', { class: `avatar ${avatarClass(account.email)}`, text: initials(account.name || account.email) }),
      h('span', { style: { flex: '1', minWidth: '0', textAlign: 'left' } },
        h('div', { style: { fontSize: '13px', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }, text: account.name }),
        h('div', { style: { fontSize: '11px', color: 'var(--text-3)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }, text: account.email })
      ),
      unread ? h('span', { class: 'nav-count', text: unread > 99 ? '99+' : String(unread) }) : null,
      h('span', { class: `acct-status-dot ${account.status || 'offline'}`, title: statusText(account.status) })
    ),
    h('div', { class: 'account-folders' },
      boxes.map((b) => navRow({
        icon: KIND_ICON[b.kind] || 'folder',
        label: b.name,
        count: b.unread || null,
        muted: false,
        active: activeBox === b.path,
        onClick: () => emit('navigate', { type: 'folder', accountId: account.id, path: b.path, name: b.name, kind: b.kind }),
      }))
    )
  );
}

function toggleCollapse(accountId) {
  const collapsed = { ...state.collapsed, [accountId]: !state.collapsed[accountId] };
  localStorage.setItem('gloss.collapsed', JSON.stringify(collapsed));
  set({ collapsed }, 'accounts');
  renderSidebar();
}

function statusText(status) {
  switch (status) {
    case 'online': return '已连接';
    case 'connecting': return '连接中…';
    case 'error': return '连接出错';
    default: return '未连接';
  }
}

// --------------------------------------------------------- mailboxes fetch

let fetching = new Set();

export async function refreshMailboxes(accountId) {
  const ids = accountId ? [accountId] : state.accounts.map((a) => a.id);
  for (const id of ids) {
    const account = state.accounts.find((a) => a.id === id);
    if (!account || fetching.has(id)) continue;
    fetching.add(id);
    api.mailboxes(id)
      .then(({ mailboxes }) => {
        const next = { ...state.mailboxesByAccount, [id]: mailboxes };
        const unreadByAccount = { ...state.unreadByAccount };
        unreadByAccount[id] = mailboxes.filter((b) => b.kind === 'inbox').reduce((s, b) => s + (b.unread || 0), 0);
        set({ mailboxesByAccount: next, unreadByAccount }, 'accounts');
        renderSidebar();
      })
      .catch(() => { /* account may be offline; sidebar shows dot */ })
      .finally(() => fetching.delete(id));
  }
}

on('accounts', (patch) => {
  // drop stale account data without re-emitting (would loop)
  const live = new Set(state.accounts.map((a) => a.id));
  const next = {};
  for (const [id, boxes] of Object.entries(state.mailboxesByAccount)) {
    if (live.has(id)) next[id] = boxes;
  }
  state.mailboxesByAccount = next;
  if (patch && patch.accounts) refreshMailboxes();
});
