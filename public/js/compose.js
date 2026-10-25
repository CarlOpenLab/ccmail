// Compose sheet — new mail, reply, reply all, forward.

import { h, icon, fmtFullDate, displayName } from './util.js';
import { state, on, emit } from './store.js';
import { api } from './api.js';
import { toast } from './util.js';

let contactsCache = [];

export function initCompose() {
  api.contacts().then(({ contacts }) => { contactsCache = contacts; }).catch(() => {});
}

on('compose', (opts = {}) => openSheet(opts));

function closeModalAll() {
  document.getElementById('modalRoot').innerHTML = '';
}

export function openSheet({ mode = 'new', message = null, envelope = null } = {}) {
  closeModalAll();

  const accounts = state.accounts;
  if (!accounts.length) {
    toast('请先添加一个邮箱账号', { type: 'err', action: '去添加', onAction: () => emit('open-settings') });
    return;
  }
  let fromId = accounts[0].id;
  if (envelope && envelope.accountId && accounts.some((a) => a.id === envelope.accountId)) {
    fromId = envelope.accountId;
  }

  const prefill = buildPrefill(mode, message, envelope);
  const attachments = [];
  const inputs = {};

  const fromSelect = h('select', { class: 'from-select' },
    accounts.map((a) => h('option', { value: a.id, selected: a.id === fromId }, `${a.name} <${a.email}>`))
  );
  const subjectInput = h('input', { type: 'text', placeholder: '邮件主题', value: prefill.subject });
  const bodyEl = h('div', { class: 'compose-body', contentEditable: 'true', 'data-placeholder': '写下你想说的…', html: prefill.bodyHtml });
  const attsRow = h('div', { class: 'compose-atts', style: { display: 'none' } });
  const statusEl = h('span', { class: 'status-text' });
  const sendBtn = h('button', { class: 'btn primary', html: icon('send', 14) + '<span>发送</span>', onClick: send });
  const fileInput = h('input', { type: 'file', multiple: true, style: { display: 'none' }, onChange: (e) => addFiles(e.target.files) });

  const ccRow = recipientRow('抄送', 'cc', prefill.cc, false);
  const bccRow = recipientRow('密送', 'bcc', prefill.bcc, false);

  const modal = h('div', {
    class: 'modal-backdrop',
    onClick: (e) => { if (e.target === modal) closeModal(); },
  },
    h('div', { class: 'modal compose-sheet' },
      h('div', { class: 'compose-head' },
        h('h3', { class: 'modal-title' }, mode === 'new' ? '新邮件' : mode === 'forward' ? '转发邮件' : '回复邮件'),
        statusEl,
        h('button', { class: 'icon-btn ghost', html: icon('close', 17), title: '关闭 (Esc)', onClick: closeModal })
      ),
      h('div', { class: 'compose-fields' },
        h('div', { class: 'compose-field' },
          h('label', {}, '发件人'),
          fromSelect,
          h('span', { style: { flex: 1 } }),
          h('button', { class: 'field-link', onClick: () => ccRow.classList.toggle('show') }, '抄送'),
          h('button', { class: 'field-link', onClick: () => bccRow.classList.toggle('show') }, '密送')
        ),
        recipientRow('收件人', 'to', prefill.to, true),
        ccRow,
        bccRow,
        h('div', { class: 'compose-field' },
          h('label', {}, '主题'),
          subjectInput
        )
      ),
      h('div', { class: 'compose-toolbar' },
        toolbarBtn('bold', '加粗 (Ctrl+B)', 'bold'),
        toolbarBtn('italic', '斜体 (Ctrl+I)', 'italic'),
        toolbarBtn('underline', '下划线 (Ctrl+U)', 'underline'),
        h('span', { class: 'tb-sep' }),
        toolbarBtn('listUl', '无序列表', 'insertUnorderedList'),
        toolbarBtn('listOl', '有序列表', 'insertOrderedList'),
        toolbarBtn('quote', '引用', 'formatBlock', 'blockquote'),
        h('span', { class: 'tb-sep' }),
        toolbarBtn('link', '添加链接', 'createlink', 'https://'),
        h('span', { class: 'tb-sep' }),
        h('button', { class: 'tb-btn', title: '添加附件', html: icon('paperclip', 15), onClick: () => fileInput.click() })
      ),
      h('div', { class: 'compose-body-wrap' },
        bodyEl,
        attsRow,
        fileInput
      ),
      h('div', { class: 'compose-foot' },
        sendBtn,
        h('button', { class: 'btn ghost', onClick: closeModal }, '取消'),
        h('span', { class: 'hint' }, 'Ctrl / ⌘ + Enter 发送 · 拖拽文件到正文可添加附件')
      )
    )
  );

  function recipientRow(label, key, value, show) {
    const dl = h('datalist', { id: `dl-${key}` },
      contactsCache.map((c) => h('option', { value: c.email }, c.name))
    );
    const input = h('input', {
      type: 'text', placeholder: 'name@example.com，多人用逗号分隔', value: value || '',
      attr: { list: `dl-${key}` }, autocomplete: 'off',
    });
    inputs[key] = input;
    return h('div', { class: `compose-field ${key}-row${show ? ' show' : ''}` },
      h('label', {}, label), input, dl
    );
  }

  function toolbarBtn(ic, title, cmd, value = null) {
    return h('button', {
      class: 'tb-btn', type: 'button', title,
      html: icon(ic, 15),
      onMouseDown: (e) => e.preventDefault(),
      onClick: () => { document.execCommand(cmd, false, value); bodyEl.focus(); },
    });
  }

  function addFiles(files) {
    for (const file of [...files].slice(0, 10)) {
      if (file.size > 20 * 1024 * 1024) {
        toast(`跳过 ${file.name}：超过 20MB`, { type: 'err' });
        continue;
      }
      const reader = new FileReader();
      reader.onload = () => {
        attachments.push({ filename: file.name, contentType: file.type || 'application/octet-stream', base64: String(reader.result).split(',')[1] || '' });
        renderAttachments();
      };
      reader.readAsDataURL(file);
    }
    fileInput.value = '';
  }

  function renderAttachments() {
    attsRow.style.display = attachments.length ? 'flex' : 'none';
    attsRow.innerHTML = '';
    for (const a of attachments) {
      attsRow.append(h('span', { class: 'att-pill' },
        icon('paperclip', 13), a.filename,
        h('button', { title: '移除', onClick: () => { attachments.splice(attachments.indexOf(a), 1); renderAttachments(); } }, '×')
      ));
    }
  }

  function send() {
    const to = parseEmails(inputs.to ? inputs.to.value : '');
    const cc = parseEmails(inputs.cc && inputs.cc.classList.contains('show') ? inputs.cc.value : '');
    const bcc = parseEmails(inputs.bcc && inputs.bcc.classList.contains('show') ? inputs.bcc.value : '');
    const subject = subjectInput.value.trim() || '（无主题）';
    if (!to.length) {
      toast('请填写收件人', { type: 'err' });
      inputs.to && inputs.to.focus();
      return;
    }
    const account = accounts.find((a) => a.id === fromSelect.value) || accounts[0];
    sendBtn.disabled = true;
    statusEl.textContent = '发送中…';
    api.send({
      accountId: account.id,
      to, cc, bcc, subject,
      html: `<div style="font-family:inherit">${bodyEl.innerHTML}</div>`,
      text: bodyEl.innerText,
      attachments: attachments.map((a) => ({ filename: a.filename, contentType: a.contentType, contentBase64: a.base64 })),
    })
      .then((res) => {
        closeModal();
        toast('发送成功', { type: 'ok' });
        emit('mail-sent');
      })
      .catch((err) => {
        sendBtn.disabled = false;
        statusEl.textContent = '';
        toast(err.message, { type: 'err', duration: 6000 });
      });
  }

  // drag & drop
  const wrap = modal.querySelector('.compose-body-wrap');
  wrap.addEventListener('dragover', (e) => { e.preventDefault(); wrap.classList.add('dragover'); });
  wrap.addEventListener('dragleave', () => wrap.classList.remove('dragover'));
  wrap.addEventListener('drop', (e) => {
    e.preventDefault();
    wrap.classList.remove('dragover');
    if (e.dataTransfer && e.dataTransfer.files.length) addFiles(e.dataTransfer.files);
  });

  const onKeydown = (e) => {
    if (e.key === 'Escape') { closeModal(); }
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); send(); }
  };
  document.addEventListener('keydown', onKeydown);

  document.getElementById('modalRoot').append(modal);
  setTimeout(() => {
    if (prefill.to) bodyEl.focus();
    else if (inputs.to) inputs.to.focus();
  }, 80);

  function closeModal() {
    document.removeEventListener('keydown', onKeydown);
    const root = document.getElementById('modalRoot');
    if (modal.parentNode === root) root.removeChild(modal);
  }
}

function parseEmails(str) {
  return String(str || '')
    .split(/[,，;；]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map((s) => {
      const m = s.match(/^(.*)<(.+@.+)>$/);
      if (m) return { name: m[1].trim().replace(/^["']|["']$/g, ''), email: m[2].trim() };
      return { name: '', email: s };
    })
    .filter((a) => /.+@.+/.test(a.email));
}

// ---------------------------------------------------------------- prefill

function buildPrefill(mode, message, envelope) {
  if (mode === 'new' || !message) return { to: '', cc: '', bcc: '', subject: '', bodyHtml: '' };

  const from = message.from || {};
  const subjectBase = message.subject || (envelope && envelope.subject) || '';
  const quoteText = state.quoteReply
    ? (message.text || String(message.html || '').replace(/<[^>]+>/g, ' ') || '')
        .replace(/\s+/g, ' ')
        .trim()
        .slice(0, 1200)
    : '';
  const quote = state.quoteReply
    ? `
    <p><br></p>
    <p style="color:#67677a">在 ${fmtFullDate(message.date)}，${displayName(from)} &lt;${from.email || ''}&gt; 写道：</p>
    <blockquote style="margin:8px 0;padding:6px 14px;border-left:3px solid #d9d9e0;color:#67677a">${quoteText}</blockquote>`
    : '';

  if (mode === 'reply') {
    return {
      to: from.email || '',
      cc: '',
      bcc: '',
      subject: /^re[:：]/i.test(subjectBase) ? subjectBase : `Re: ${subjectBase}`,
      bodyHtml: `<p>${displayName(from)} 你好，</p>${quote}`,
    };
  }
  if (mode === 'replyAll') {
    const me = envelope ? (state.accounts.find((a) => a.id === envelope.accountId) || {}).email : '';
    const others = (message.to || []).concat(message.cc || [])
      .filter((a) => a.email && a.email !== me && a.email !== from.email)
      .map((a) => a.email);
    return {
      to: [from.email, ...others].filter(Boolean).join(', '),
      cc: '',
      bcc: '',
      subject: /^re[:：]/i.test(subjectBase) ? subjectBase : `Re: ${subjectBase}`,
      bodyHtml: `<p>${displayName(from)} 你好，</p>${quote}`,
    };
  }
  const fwdMeta = `<p style="color:#67677a">---------- 转发邮件 ----------<br>发件人：${displayName(from)} &lt;${from.email || ''}&gt;<br>日期：${fmtFullDate(message.date)}<br>主题：${subjectBase}</p><p><br></p>`;
  return {
    to: '',
    cc: '',
    bcc: '',
    subject: /^fwd[:：]|^转发[:：]/i.test(subjectBase) ? subjectBase : `Fwd: ${subjectBase}`,
    bodyHtml: fwdMeta + `<p>${(message.text || '').slice(0, 4000).replace(/\n/g, '<br>')}</p>`,
  };
}
