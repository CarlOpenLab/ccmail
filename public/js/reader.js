// Reading pane — the message itself: hero header, sanitized body in a
// sandboxed iframe (remote images blocked by default), attachments.

import { h, icon, iconEl, initials, avatarClass, fmtFullDate, fmtSize, displayName, sanitizeEmailHtml, emailCss, $ } from './util.js';
import { state, on, emit, set } from './store.js';
import { api } from './api.js';
import { toggleStar, markRead, deleteMsg, archiveMsg, renderList } from './list.js';

export function renderReader() {
  const root = $('#readerPane');
  root.innerHTML = '';

  if (!state.selection) {
    root.append(emptyReader());
    return;
  }
  if (state.messageLoading || !state.message) {
    root.append(h('div', { class: 'reader-empty' }, h('div', { class: 'spinner' })));
    return;
  }

  const m = state.message;
  const sel = state.selection;
  const account = state.accounts.find((a) => a.id === sel.accountId);
  const flagged = !!(sel.flags && sel.flags.flagged);

  const cidMap = new Map();
  for (const att of m.attachments || []) {
    if (att.contentId) cidMap.set(att.contentId.replace(/^<|>$/g, ''), api.attachmentUrl(sel.accountId, sel.path, sel.uid, att.id));
  }

  let frame = null;
  let textBody = null;
  let guard = null;
  let sanitized = { html: '', blockedImages: 0 };

  if (m.html) {
    sanitized = sanitizeEmailHtml(m.html, {
      attachmentUrl: (cid) => cidMap.get(cid) || null,
      allowImages: state.autoImages === true,
    });
    frame = h('iframe', {
      class: 'reader-body-frame',
      attr: { sandbox: 'allow-same-origin allow-popups' },
      title: '邮件正文',
    });
  } else {
    textBody = h('div', { class: 'reader-body-text', text: m.text || '（这封邮件没有内容）' });
  }
  if (sanitized.blockedImages > 0) {
    guard = h('div', { class: 'image-guard' },
      iconEl('image', 15),
      h('span', { style: { flex: 1 } }, `已屏蔽 ${sanitized.blockedImages} 张远程图片以保护隐私`),
      h('button', {
        class: 'btn small subtle', onClick: () => {
          renderWithImages();
        },
      }, '加载图片')
    );
  }

  root.append(h('div', { class: 'reader-scroll' },
    h('div', { class: 'reader-inner' },
      h('div', { class: 'reader-card' },
        h('div', { class: 'reader-head' },
          h('h1', { class: 'reader-subject' }, m.subject || state.selection.subject || '（无主题）'),
          h('div', { class: 'rh-from-row' },
            h('span', { class: `avatar ${avatarClass(m.from && (m.from.email || m.from.name))}`, text: initials(m.from && (m.from.name || m.from.email)) }),
            h('div', { class: 'rh-meta' },
              h('div', { class: 'rh-name' },
                displayName(m.from),
                account ? h('span', { class: 'mr-acct-chip', style: { background: account.color, fontWeight: 500 } }, account.name) : null
              ),
              h('div', { class: 'rh-email', text: m.from ? m.from.email : '' }),
              h('div', { class: 'rh-to-line', onclick: toggleDetails, title: '点击展开详情' },
                `收件人：${(m.to || []).map(displayName).join('、') || '我'}`,
                h('span', { class: 'expand-hint' }, ' ▾')
              ),
              h('div', { class: 'rh-details', hidden: true },
                h('dl', {},
                  h('dt', {}, '发件人'), h('dd', {}, m.from ? `${displayName(m.from)} <${m.from.email}>` : '—'),
                  h('dt', {}, '收件人'), h('dd', {}, (m.to || []).map((a) => a.email).join(', ') || '—'),
                  (m.cc && m.cc.length) ? [h('dt', {}, '抄送'), h('dd', {}, m.cc.map((a) => a.email).join(', '))] : null
                )
              )
            ),
            h('div', { class: 'rh-date-block' },
              h('div', {}, fmtFullDate(m.date))
            )
          ),
          h('div', { class: 'reader-actions' },
            actionBtn('reply', '回复', () => emit('compose', { mode: 'reply', message: m, envelope: sel })),
            (m.to && m.to.length > 1) || (m.cc && m.cc.length)
              ? actionBtn('replyAll', '回复全部', () => emit('compose', { mode: 'replyAll', message: m, envelope: sel }))
              : null,
            actionBtn('forward', '转发', () => emit('compose', { mode: 'forward', message: m, envelope: sel })),
            h('span', { class: 'spacer' }),
            h('button', { class: `action-pill${flagged ? ' active' : ''}`, title: '星标', html: icon('star', 14, flagged), onClick: () => { toggleStar(sel); setTimeout(renderReader, 0); } }),
            h('button', { class: 'action-pill', title: '标为未读', html: icon('eyeOff', 14), onClick: () => { markRead(sel, false); emit('reader-reset'); } }),
            h('button', { class: 'action-pill', title: '归档', html: icon('archive', 14), onClick: () => archiveMsg(sel) }),
            h('button', { class: 'action-pill danger', title: '删除', html: icon('trash', 14), onClick: () => deleteMsg(sel) })
          ),
          guard
        ),
        frame || textBody,
        (m.attachments && m.attachments.length)
          ? h('div', { class: 'attachments-strip' },
              h('div', { class: 'attachments-label' }, `附件 · ${m.attachments.length}`),
              h('div', { class: 'attachment-cards' },
                m.attachments.map((att) => {
                  const isImage = /^image\//.test(att.contentType) && !/svg/.test(att.contentType);
                  const ext = (att.filename.split('.').pop() || 'FILE').slice(0, 4).toUpperCase();
                  return h('a', {
                    class: 'att-card', href: api.attachmentUrl(sel.accountId, sel.path, sel.uid, att.id),
                    title: `下载 ${att.filename}`,
                  },
                    h('span', { class: `att-icon${isImage ? ' image' : ''}`, text: ext }),
                    h('span', {},
                      h('div', { class: 'att-name', text: att.filename }),
                      h('div', { class: 'att-size', text: fmtSize(att.size) })
                    )
                  );
                })
              )
            )
          : null
      )
    )
  ));

  if (frame) {
    const srcdoc = `<!doctype html><html><head><base target="_blank">${emailCss}</head><body><div class="gloss-pad">${sanitized.html}</div></body></html>`;
    frame.addEventListener('load', () => {
      try {
        const doc = frame.contentDocument;
        if (doc) frame.style.height = Math.max(120, doc.documentElement.scrollHeight + 4) + 'px';
      } catch { frame.style.height = '420px'; }
    });
    frame.srcdoc = srcdoc;
  }

  function toggleDetails() {
    const details = root.querySelector('.rh-details');
    if (details) details.hidden = !details.hidden;
  }

  function renderWithImages() {
    const open = sanitizeEmailHtml(m.html, {
      attachmentUrl: (cid) => cidMap.get(cid) || null,
      allowImages: true,
    });
    const f = root.querySelector('.reader-body-frame');
    if (f) {
      f.srcdoc = `<!doctype html><html><head><base target="_blank">${emailCss}</head><body><div class="gloss-pad">${open.html}</div></body></html>`;
    }
    if (guard) guard.remove();
  }
}

function actionBtn(ic, label, onClick) {
  return h('button', { class: 'action-pill', html: icon(ic, 14) + `<span>${label}</span>`, onClick });
}

function emptyReader() {
  return h('div', { class: 'reader-empty' },
    h('div', { class: 'glyph', html: icon('sparkle', 32) }),
    h('h3', {}, '选择一封邮件'),
    h('p', {}, '从中间的列表选择一封邮件开始阅读，<br>或按 N 写一封新邮件。'),
    h('div', { class: 'kbd-hints' },
      h('span', {}, h('kbd', {}, 'J'), '下一封'),
      h('span', {}, h('kbd', {}, 'K'), '上一封'),
      h('span', {}, h('kbd', {}, 'R'), '回复'),
      h('span', {}, h('kbd', {}, '?'), '快捷键')
    )
  );
}

// selection / message loading -------------------------------------------

on('open-message', async (env) => {
  set({ selection: env, message: null, messageLoading: true });
  document.getElementById('main').classList.add('reader-open');
  renderReader();
  renderList();
  try {
    const { message } = await api.message(env.accountId, env.path, env.uid, { markSeen: state.markRead });
    // user may have navigated elsewhere while loading
    if (!state.selection || state.selection.uid !== env.uid || state.selection.accountId !== env.accountId) return;
    set({ message, messageLoading: false });
    env.flags = { ...(env.flags || {}), seen: state.markRead ? true : (env.flags && env.flags.seen) };
    renderReader();
    renderList();
    if (state.markRead) emit('flags-changed', { accountId: env.accountId, uid: env.uid, flags: env.flags });
  } catch (err) {
    set({ messageLoading: false });
    const root = $('#readerPane');
    root.innerHTML = '';
    root.append(h('div', { class: 'reader-empty' },
      iconEl('junk', 28),
      h('h3', {}, '无法加载这封邮件'),
      h('p', {}, err.message)
    ));
  }
});

on('reader-reset', () => {
  document.getElementById('main').classList.remove('reader-open');
  renderReader();
});
