// Settings — floating draggable panel (no scrim), left navigation,
// grouped inset lists, segmented controls, toggle switches, wash picker.
// Add-account wizard remains a scrim modal (action type).

import { h, icon, initials, avatarClass, toast } from './util.js';
import { state, on, emit, set, applyTheme, persistPrefs } from './store.js';
import { api } from './api.js';

const COLOR_SWATCHES = ['#3b6cf6', '#0f8199', '#22863a', '#b08800', '#d3302f', '#a033c4', '#5b4fd4', '#64748b'];

const WASHES = [
  { id: 'plain', name: '纯色' },
  { id: 'rex', name: 'Rex' },
  { id: 'mist', name: 'Mist' },
  { id: 'meadow', name: 'Meadow' },
  { id: 'dusk', name: 'Dusk' },
];

const WASH_PREVIEW = {
  plain: { light: '#eceef4', dark: '#141418' },
  rex: { light: 'linear-gradient(135deg,#ebe0f3,#f3e7f1 34%,#f7eee4 67%,#f6efcf)', dark: 'linear-gradient(135deg,#17141f,#191620 50%,#1b1916)' },
  mist: { light: 'linear-gradient(135deg,#e3e9f4,#ebeff6 34%,#f1f2f3 67%,#f2f1ec)', dark: 'linear-gradient(135deg,#12151b,#14171d 50%,#17181b)' },
  meadow: { light: 'linear-gradient(135deg,#e2eee5,#ebf2e7 34%,#f3f2e2 67%,#f5f0d6)', dark: 'linear-gradient(135deg,#111814,#141a16 50%,#181a14)' },
  dusk: { light: 'linear-gradient(135deg,#e4dbf3,#efdeee 34%,#f6e3dc 67%,#f7e6cf)', dark: 'linear-gradient(135deg,#191321,#1e161e 50%,#221a17)' },
};

function currentAppearance() {
  return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

on('open-settings', () => openSettings());

export function openSettings(initialTab = 'accounts') {
  let tab = initialTab;
  const root = document.getElementById('modalRoot');
  root.innerHTML = '';

  const NAV = [
    ['accounts', '账号', 'user'],
    ['appearance', '外观', 'sun'],
    ['shortcuts', '快捷键', 'sparkle'],
    ['about', '关于', 'mail'],
  ];

  const nav = h('div', { class: 'settings-nav' });
  const content = h('div', { class: 'settings-content' });
  const panel = h('div', { class: 'settings-panel' },
    h('div', { class: 'settings-head', title: '拖动移动窗口' },
      h('h3', { class: 'modal-title' }, '设置'),
      h('span', { style: { flex: 1 } }),
      h('span', { style: { fontSize: 11, color: 'var(--text-3)', marginRight: 6 } }, '实时预览 · Esc 关闭'),
      h('button', { class: 'icon-btn ghost', html: icon('close', 16), onClick: close })
    ),
    h('div', { class: 'settings-body' }, nav, content)
  );

  const backdrop = h('div', { class: 'modal-backdrop bare' }, panel);

  function renderNav() {
    nav.innerHTML = '';
    for (const [key, label, ic] of NAV) {
      nav.append(h('button', {
        class: `nav-row${tab === key ? ' active' : ''}`,
        onClick: () => { tab = key; renderNav(); renderContent(); },
      },
        h('span', { class: 'nav-icon', html: icon(ic, 16) }),
        h('span', { class: 'nav-label' }, label)
      ));
    }
  }

  function renderContent() {
    content.innerHTML = '';
    if (tab === 'accounts') content.append(accountsTab(renderContent));
    if (tab === 'appearance') content.append(appearanceTab());
    if (tab === 'shortcuts') content.append(shortcutsTab());
    if (tab === 'about') content.append(aboutTab());
  }

  const onEsc = (e) => { if (e.key === 'Escape') close(); };
  function close() {
    document.removeEventListener('keydown', onEsc);
    if (backdrop.parentNode === root) root.removeChild(backdrop);
  }

  // rigid drag on the header, bounded to the viewport (12px margins)
  const head = panel.querySelector('.settings-head');
  let drag = null;
  head.addEventListener('pointerdown', (e) => {
    if (e.target.closest('button')) return;
    const r = panel.getBoundingClientRect();
    drag = { sx: e.clientX, sy: e.clientY, ox: r.left, oy: r.top };
    head.setPointerCapture(e.pointerId);
    e.preventDefault();
  });
  head.addEventListener('pointermove', (e) => {
    if (!drag) return;
    const x = drag.ox + e.clientX - drag.sx;
    const y = drag.oy + e.clientY - drag.sy;
    const maxX = Math.max(12, window.innerWidth - 12 - panel.offsetWidth);
    const maxY = Math.max(12, window.innerHeight - 12 - panel.offsetHeight);
    panel.style.position = 'fixed';
    panel.style.left = `${Math.min(Math.max(12, x), maxX)}px`;
    panel.style.top = `${Math.min(Math.max(12, y), maxY)}px`;
    panel.style.margin = '0';
    backdrop.style.placeItems = 'start';
  });
  head.addEventListener('pointerup', () => { drag = null; });
  head.addEventListener('pointercancel', () => { drag = null; });

  document.addEventListener('keydown', onEsc);
  renderNav();
  renderContent();
  root.append(backdrop);
}

/* ------------------------------------------------------------ components */

function toggleSwitch(on, onChange) {
  const t = h('button', {
    class: `toggle${on ? ' on' : ''}`,
    attr: { role: 'switch', 'aria-checked': String(on) },
    onClick: () => {
      const next = !t.classList.contains('on');
      t.classList.toggle('on', next);
      t.setAttribute('aria-checked', String(next));
      onChange(next);
    },
  }, h('span', { class: 'knob' }));
  return t;
}

function segmented(options, value, onChange) {
  const seg = h('div', { class: 'segmented' },
    options.map(([val, label]) => h('button', {
      class: `seg${val === value ? ' active' : ''}`,
      onClick: () => {
        seg.querySelectorAll('.seg').forEach((s) => s.classList.remove('active'));
        [...seg.querySelectorAll('.seg')][options.findIndex(([v]) => v === val)].classList.add('active');
        onChange(val);
      },
    }, label))
  );
  return seg;
}

/* ------------------------------------------------------------ accounts tab */

function accountsTab(rerender) {
  const wrap = h('div', {});

  for (const account of state.accounts) {
    let editing = false;
    let confirmDelete = false;
    let draftColor = account.color;

    const group = h('div', { class: 'inset-group' });
    const build = () => {
      group.innerHTML = '';
      const row = h('div', { class: 'inset-row' },
        h('span', { class: `avatar ${avatarClass(account.email)}`, style: { background: draftColor, width: 36, height: 36 }, text: initials(account.name || account.email) }),
        h('div', { class: 'ir-main' },
          h('div', { class: 'ir-title' }, account.name),
          h('div', { class: 'ir-sub', text: accountSub(account) })
        ),
        h('span', { class: 'status-text' },
          h('span', { class: `acct-status-dot ${account.status || 'offline'}` }),
          statusText(account.status)
        ),
        h('button', { class: 'btn small ghost', html: icon('edit', 12) + '编辑', onClick: () => { editing = !editing; build(); } }),
        confirmDelete
          ? h('button', { class: 'btn small danger', text: '确认删除？', onClick: async () => {
              await api.deleteAccount(account.id).catch(() => {});
              toast(`已移除账号 ${account.name}`);
              emit('accounts-refresh');
              rerender();
            } })
          : h('button', { class: 'btn small ghost', style: { color: 'var(--danger)' }, html: icon('trash', 12), title: '删除账号', onClick: () => { confirmDelete = true; build(); } })
      );
      group.append(row);

      if (editing) {
        let nameInput = null;
        const editRow = h('div', { class: 'inset-row', style: { flexWrap: 'wrap', gap: 14 } },
          h('div', { style: { flex: 1, minWidth: 200 } },
            h('div', { class: 'ir-sub', style: { marginBottom: 6 } }, '显示名称'),
            nameInput = h('input', { class: 'input', value: account.name })
          ),
          h('div', {},
            h('div', { class: 'ir-sub', style: { marginBottom: 6 } }, '账号颜色'),
            h('div', { class: 'color-swatches' },
              COLOR_SWATCHES.map((c) => h('button', {
                class: `swatch${draftColor === c ? ' selected' : ''}`,
                style: { background: c },
                onClick: (e) => {
                  draftColor = c;
                  editFormSwatches(e.currentTarget);
                },
              })))
          ),
          h('div', { style: { display: 'flex', gap: 8, alignItems: 'flex-end' } },
            h('button', {
              class: 'btn small primary', onClick: async () => {
                await api.patchAccount(account.id, { name: nameInput.value.trim() || account.name, color: draftColor });
                account.name = nameInput.value.trim() || account.name;
                account.color = draftColor;
                toast('已保存', { type: 'ok' });
                emit('accounts-refresh');
                editing = false;
                build();
              },
            }, '保存'),
            h('button', { class: 'btn small ghost', onClick: () => { editing = false; build(); } }, '取消')
          )
        );
        const editFormSwatches = (clicked) => {
          const rowEl = clicked.closest('.inset-row');
          rowEl.querySelectorAll('.swatch').forEach((s) => s.classList.remove('selected'));
          clicked.classList.add('selected');
        };
        group.append(editRow);
      }
    };
    build();
    wrap.append(h('div', { class: 'group-title' }, '邮箱账号'), group);
  }

  wrap.append(h('div', { style: { display: 'flex', gap: 10, marginTop: 20, flexWrap: 'wrap' } },
    h('button', { class: 'btn primary', html: icon('plus', 14) + '<span>添加邮箱账号</span>', onClick: () => openAddWizard() }),
  ));

  // ---- data management ----
  wrap.append(h('div', { class: 'group-title' }, '数据与缓存'));
  wrap.append(dataGroup());
  return wrap;
}

function accountSub(account) {
  const parts = [account.email];
  if (account.lastSync) parts.push(`上次同步 ${account.lastSync}`);
  else if (account.status === 'online') parts.push('已连接');
  return parts.join(' · ');
}

function dataGroup() {
  let confirmClear = false;
  const group = h('div', { class: 'inset-group' });
  const build = () => {
    group.innerHTML = '';
    group.append(h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '邮件缓存'),
        h('div', { class: 'ir-sub' }, '清空已缓存的邮件正文与附件（不影响账号和远端邮件，下次打开会重新拉取）')
      ),
      confirmClear
        ? h('button', { class: 'btn small danger', text: '确认清除？', onClick: async () => {
            try {
              const d = await api.clearCache();
              toast(`已清除缓存（${d.removed || 0} 项）`, { type: 'ok' });
            } catch (e) { toast('清除失败：' + e.message, { type: 'err' }); }
            confirmClear = false;
            build();
          } })
        : h('button', { class: 'btn small ghost', text: '清除', onClick: () => { confirmClear = true; build(); } })
    ));
  };
  build();
  return group;
}

function statusText(status) {
  switch (status) {
    case 'online': return '已连接';
    case 'connecting': return '连接中…';
    case 'error': return '出错';
    default: return '离线';
  }
}

/* --------------------------------------------------------- appearance tab */

function appearanceTab() {
  const mode = state.theme === 'auto' ? currentAppearance() : state.theme;
  const washSwatch = (w) => {
    const bg = WASH_PREVIEW[w.id][mode];
    return h('button', {
      class: `wash-chip${state.wash === w.id ? ' selected' : ''}`,
      onClick: () => {
        set({ wash: w.id });
        applyTheme();
        persistPrefs();
        washRow.querySelectorAll('.wash-chip').forEach((c) => c.classList.remove('selected'));
        washRow.querySelector(`[data-wash-id="${w.id}"]`).classList.add('selected');
      },
      dataset: { washId: w.id },
    },
      h('span', { class: 'wash-swatch', style: { background: bg } }),
      h('span', { class: 'wash-name' }, w.name)
    );
  };

  const themeGroup = h('div', { class: 'inset-group' },
    h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '外观模式'),
        h('div', { class: 'ir-sub' }, '浅色、深色或跟随系统')
      ),
      segmented([['auto', '跟随系统'], ['light', '浅色'], ['dark', '深色']], state.theme, (val) => {
        set({ theme: val });
        applyTheme();
        persistPrefs();
        refreshWashPreview();
      })
    ),
    h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '列表密度'),
        h('div', { class: 'ir-sub' }, '紧凑模式隐藏摘要，一屏显示更多邮件')
      ),
      segmented([['cozy', '舒适'], ['compact', '紧凑']], state.density, (val) => {
        set({ density: val });
        applyTheme();
        persistPrefs();
      })
    )
  );

  const washRow = h('div', { class: 'wash-row' },
    WASHES.map(washSwatch)
  );
  const washGroup = h('div', { class: 'inset-group' },
    h('div', { class: 'inset-row', style: { flexWrap: 'wrap' } },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '窗口背景'),
        h('div', { class: 'ir-sub' }, '默认纯色最护眼；渐变水洗为可选个性化主题')
      )
    ),
    h('div', { style: { padding: '4px 0 12px' } }, washRow)
  );

  function refreshWashPreview() {
    const m = state.theme === 'auto' ? currentAppearance() : state.theme;
    washRow.querySelectorAll('.wash-swatch').forEach((sw, i) => {
      sw.style.background = WASH_PREVIEW[WASHES[i].id][m];
    });
  }

  const readingGroup = h('div', { class: 'inset-group' },
    h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '自动标为已读'),
        h('div', { class: 'ir-sub' }, '打开邮件时标记为已读')
      ),
      toggleSwitch(state.markRead, (on) => { set({ markRead: on }); persistPrefs(); })
    ),
    h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '自动加载远程图片'),
        h('div', { class: 'ir-sub' }, '默认屏蔽以保护隐私；开启后直接显示邮件中的远程图片')
      ),
      toggleSwitch(state.autoImages, (on) => { set({ autoImages: on }); persistPrefs(); })
    ),
    h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '回复时引用原文'),
        h('div', { class: 'ir-sub' }, '在回复邮件底部附上原邮件引文')
      ),
      toggleSwitch(state.quoteReply, (on) => { set({ quoteReply: on }); persistPrefs(); })
    )
  );

  // desktop notifications live on the Go side (data/prefs.json)
  const notifyGroup = h('div', { class: 'inset-group' });
  const buildNotify = (enabled) => {
    notifyGroup.innerHTML = '';
    notifyGroup.append(h('div', { class: 'inset-row' },
      h('div', { class: 'ir-main' },
        h('div', { class: 'ir-title' }, '新邮件桌面通知'),
        h('div', { class: 'ir-sub' }, '系统级通知弹窗，点击可回到窗口（仅桌面版生效）')
      ),
      toggleSwitch(enabled, async (on) => {
        try {
          await api.setPrefs({ notifications: on ? 'on' : 'off' });
          buildNotify(on);
        } catch { toast('保存失败', { type: 'err' }); }
      })
    ));
  };
  api.prefs()
    .then(({ prefs }) => buildNotify(prefs.notifications !== 'off'))
    .catch(() => buildNotify(true));

  return h('div', {},
    h('div', { class: 'group-title' }, '主题'),
    themeGroup,
    h('div', { class: 'group-title' }, '背景水洗'),
    washGroup,
    h('p', { class: 'group-note' }, '水洗渐变按 Oklab 色彩空间插值，浅色与深色模式各自调校。'),
    h('div', { class: 'group-title' }, '阅读'),
    readingGroup,
    h('div', { class: 'group-title' }, '通知'),
    notifyGroup,
  );
}

/* ----------------------------------------------------------- other tabs */

function shortcutsTab() {
  const rows = [
    ['J / K', '下一封 / 上一封'],
    ['Enter', '打开选中的邮件'],
    ['N 或 C', '写新邮件'],
    ['R / A', '回复 / 回复全部'],
    ['F', '转发'],
    ['S 或 X', '加 / 取消星标'],
    ['E', '归档'],
    ['Del', '删除'],
    ['U', '返回列表（窄屏）'],
    ['/', '聚焦搜索框'],
    ['Ctrl B', '切换侧栏'],
    ['?', '打开快捷键说明'],
  ];
  return h('div', {},
    h('div', { class: 'group-title' }, '全局快捷键'),
    h('div', { class: 'inset-group' },
      rows.map(([k, v]) => h('div', { class: 'inset-row' },
        h('div', { style: { width: 150, flex: 'none' } }, ...k.split(' ').map((x) => h('kbd', {}, x))),
        h('div', { class: 'ir-sub', style: { marginTop: 0 } }, v)
      ))
    )
  );
}

function aboutTab() {
  return h('div', { style: { textAlign: 'center', padding: '46px 10px' } },
    h('div', { class: 'brand-glyph', style: { width: 54, height: 54, borderRadius: 16, margin: '0 auto 16px' } }),
    h('h3', { style: { margin: '0 0 4px' } }, 'Gloss Mail'),
    h('p', { class: 'field-hint' }, 'v1.1.0 · 多账号 IMAP / SMTP 邮件客户端'),
    h('p', { class: 'field-hint', style: { maxWidth: 360, margin: '10px auto' } },
      '布局遵循 Agent Desktop UI 设计系统：浮动卡片画布、无边框海拔、Oklab 渐变水洗。基于 Node.js + ImapFlow + Nodemailer，数据仅存本机，凭据 AES-256-GCM 加密。')
  );
}

/* ----------------------------------------------------------- add wizard */

export function openAddWizard(preselectProvider = null) {
  const root = document.getElementById('modalRoot');
  root.innerHTML = '';

  let step = preselectProvider ? 2 : 1;
  let provider = preselectProvider ? state.providers.find((p) => p.id === preselectProvider) : null;
  let color = provider ? provider.color : COLOR_SWATCHES[0];
  let payload = null;
  let submitting = false;

  const title = h('h3', { class: 'modal-title' });
  const body = h('div', { class: 'modal-body' });

  const modal = h('div', { class: 'modal-backdrop', onClick: (e) => { if (e.target === modal) close(); } },
    h('div', { class: 'modal' },
      h('div', { class: 'modal-head' },
        title,
        h('button', { class: 'icon-btn ghost', html: icon('close', 17), onClick: close })
      ),
      body
    )
  );

  const onEsc = (e) => { if (e.key === 'Escape') close(); };
  function close() {
    document.removeEventListener('keydown', onEsc);
    if (modal.parentNode === root) root.removeChild(modal);
  }

  function render() {
    body.innerHTML = '';
    if (step === 1) renderProviders();
    else if (step === 2) renderForm();
    else renderConnecting();
  }

  function renderProviders() {
    title.textContent = '选择邮件服务商';
    body.append(
      h('div', { class: 'provider-grid' },
        state.providers.map((p) => h('button', {
          class: `provider-tile${provider && provider.id === p.id ? ' selected' : ''}`,
          onClick: () => {
            provider = p;
            color = p.color;
            step = 2;
            render();
          },
        },
          h('span', { class: 'provider-mark', style: { background: p.color }, text: p.monogram }),
          h('span', { class: 'pt-name' }, p.name)
        ))
      ),
      h('p', { class: 'field-hint', style: { marginTop: 14 } },
        'QQ / 163 / Gmail 等需要在邮箱网页版开启 IMAP 服务，并使用「授权码 / 应用专用密码」登录，而非网页登录密码。'
      )
    );
  }

  function renderForm() {
    title.textContent = provider ? `添加 ${provider.name}` : '添加邮箱账号';
    const isCustom = !provider || provider.id === 'custom';

    const emailInput = h('input', { class: 'input', type: 'email', placeholder: 'you@example.com', autocomplete: 'off' });
    const passInput = h('input', { class: 'input', type: 'password', placeholder: provider && provider.id === 'qq' ? '16 位授权码' : '密码 / 授权码 / 应用密码' });
    const nameInput = h('input', { class: 'input', placeholder: '发件人昵称（可选）' });
    const imapHost = h('input', { class: 'input', value: (provider && provider.imap.host) || '', placeholder: 'imap.example.com' });
    const imapPort = h('input', { class: 'input', type: 'number', value: (provider && provider.imap.port) || 993 });
    const smtpHost = h('input', { class: 'input', value: (provider && provider.smtp.host) || '', placeholder: 'smtp.example.com' });
    const smtpPort = h('input', { class: 'input', type: 'number', value: (provider && provider.smtp.port) || 465 });

    const adv = h('div', { style: { display: isCustom ? 'block' : 'none' } },
      h('div', { class: 'adv-grid' },
        h('div', { class: 'adv-header' }, 'IMAP 收信服务器'),
        h('div', {}, ''), h('div', { style: { fontSize: 11.5, color: 'var(--text-3)', fontWeight: 600 } }, '端口'),
        imapHost, imapPort, h('div', {}, ''),
        h('div', { class: 'adv-header' }, 'SMTP 发信服务器'),
        h('div', {}, ''), h('div', { style: { fontSize: 11.5, color: 'var(--text-3)', fontWeight: 600 } }, '端口'),
        smtpHost, smtpPort, h('div', {}, '')
      )
    );

    const errEl = h('div', { class: 'field-error', style: { display: 'none' } });
    const showError = (msg) => { errEl.style.display = msg ? 'block' : 'none'; errEl.textContent = msg || ''; };

    const submitBtn = h('button', {
      class: 'btn primary', html: icon('check', 14) + '<span>验证并添加</span>',
      onClick: () => {
        const email = emailInput.value.trim().toLowerCase();
        if (!/.+@.+\..+/.test(email)) return showError('请填写有效的邮箱地址');
        if (!passInput.value) return showError('请填写密码或授权码');
        if (isCustom && !imapHost.value.trim()) return showError('自定义服务需要填写 IMAP 服务器地址');
        if (isCustom && !smtpHost.value.trim()) return showError('自定义服务需要填写 SMTP 服务器地址');
        showError('');
        payload = {
          provider: (provider && provider.id) || 'custom',
          email,
          password: passInput.value,
          name: nameInput.value.trim() || email.split('@')[0],
          color,
          imap: {
            host: imapHost.value.trim() || (provider && provider.imap.host) || '',
            port: Number(imapPort.value) || (provider && provider.imap.port) || 993,
            secure: true,
          },
          smtp: {
            host: smtpHost.value.trim() || (provider && provider.smtp.host) || '',
            port: Number(smtpPort.value) || (provider && provider.smtp.port) || 465,
            secure: !((provider && provider.smtp.requireTLS)),
            requireTLS: !!(provider && provider.smtp.requireTLS),
          },
        };
        step = 3;
        render();
        submit();
      },
    });

    body.append(
      h('div', { class: 'field' }, h('label', {}, '邮箱地址 ', h('span', { class: 'req' }, '*')), emailInput),
      h('div', { class: 'field' },
        h('label', {}, '密码 / 授权码 ', h('span', { class: 'req' }, '*')),
        passInput,
        provider ? h('div', { class: 'field-hint warn' }, provider.hint) : null
      ),
      h('div', { class: 'field' },
        h('label', {}, '账号颜色'),
        h('div', { class: 'color-swatches' },
          COLOR_SWATCHES.map((c) => h('button', {
            class: `swatch${color === c ? ' selected' : ''}`,
            style: { background: c },
            onClick: (e) => {
              color = c;
              body.querySelectorAll('.swatch').forEach((s) => s.classList.remove('selected'));
              e.currentTarget.classList.add('selected');
            },
          })))
      ),
      h('div', { class: 'field' },
        h('label', {}, '发件人昵称'),
        nameInput,
        h('button', {
          class: 'adv-toggle', style: { marginTop: 8 }, onClick: () => {
            adv.style.display = adv.style.display === 'none' ? 'block' : 'none';
          },
          html: icon('settings', 13) + ' 高级：服务器设置' + (isCustom ? '（自定义必填）' : ''),
        })
      ),
      adv,
      errEl,
      h('div', { class: 'modal-foot', style: { padding: '12px 0 0', border: 'none' } },
        h('button', { class: 'btn ghost', onClick: () => { step = 1; render(); } }, '上一步'),
        submitBtn
      )
    );

    emailInput.addEventListener('input', () => {
      if (isCustom) {
        const domain = emailInput.value.split('@')[1] || '';
        const guess = state.providers.find((p) => p.domains.includes(domain.toLowerCase()));
        if (guess) {
          provider = guess;
          color = guess.color;
          render();
        }
      }
    });
    passInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') submitBtn.click(); });
    setTimeout(() => emailInput.focus(), 80);
  }

  function renderConnecting() {
    title.textContent = '正在连接';
    const ring = h('div', { class: 'connect-ring' },
      '<svg viewBox="0 0 66 66"><circle class="track" cx="33" cy="33" r="28" fill="none" stroke-width="5"/><circle class="bar" cx="33" cy="33" r="28" fill="none" stroke-width="5" stroke-dasharray="40 140"/></svg>'
    );
    const steps = h('div', { class: 'connect-steps' },
      h('div', { class: 'step' }, '· 连接 IMAP 服务器…'),
      h('div', { class: 'step', style: { opacity: 0.55 } }, '· 验证凭据…')
    );
    const errBox = h('div', { class: 'field-error', style: { display: 'none' } });
    const retryRow = h('div', { style: { display: 'none', marginTop: 18 } },
      h('button', { class: 'btn ghost', onClick: () => { step = 2; render(); } }, '返回修改'),
      h('button', { class: 'btn primary', style: { marginLeft: 8 }, onClick: () => { step = 3; render(); submit(); } }, '重试')
    );
    body.append(h('div', { class: 'connect-box' },
      ring,
      h('p', { class: 'connect-title' }, submitting ? '正在验证账号…' : '准备验证…'),
      steps, errBox, retryRow
    ));
    connectingRefs.ring = ring;
    connectingRefs.steps = steps;
    connectingRefs.errBox = errBox;
    connectingRefs.retryRow = retryRow;
  }

  const connectingRefs = {};

  async function submit() {
    if (!payload || submitting) return;
    submitting = true;
    try {
      const { account, smtpWarning } = await api.addAccount(payload);
      connectingRefs.ring && connectingRefs.ring.classList.add('done');
      connectingRefs.steps && (connectingRefs.steps.innerHTML = '<div class="step ok">✓ IMAP 连接成功</div><div class="step ok">✓ 账号已挂载到 Gloss</div>');
      const t = body.querySelector('.connect-title');
      if (t) t.textContent = `账号 ${account.name} 添加成功`;
      if (smtpWarning) toast(smtpWarning, { type: 'info', duration: 7000 });
      emit('accounts-refresh');
      setTimeout(() => {
        close();
        emit('navigate', { type: 'folder', accountId: account.id, path: 'INBOX', name: '收件箱', kind: 'inbox' });
      }, 900);
    } catch (err) {
      connectingRefs.ring && connectingRefs.ring.classList.add('error');
      connectingRefs.steps && (connectingRefs.steps.innerHTML = `<div class="step fail">✗ ${err.message}</div>`);
      const t = body.querySelector('.connect-title');
      if (t) t.textContent = '连接失败';
      connectingRefs.errBox && (connectingRefs.errBox.style.display = 'block');
      connectingRefs.retryRow && (connectingRefs.retryRow.style.display = 'block');
    } finally {
      submitting = false;
    }
  }

  document.addEventListener('keydown', onEsc);
  render();
  root.append(modal);
}
