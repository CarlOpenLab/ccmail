# Gloss Mail ✦

一个参考 **Gloss Mail 布局**打造的多账号邮件客户端 —— 轻盈、安静、专注。支持 Gmail、QQ 邮箱、163、126、Outlook、iCloud、企业邮箱等任意 IMAP/SMTP 服务，可同时挂载多个账号，聚合到统一收件箱。

**v2：已使用 [mygo](https://github.com/egoist/mygo) 全部重写为 Go 桌面应用** —— 单个 `gloss-mail.exe`（约 19MB，纯 Go 无 cgo，WebView2 渲染）：前端 `go:embed` 内嵌、API 走 `mygo.Protocol` 自定义协议、实时推送走 mygo 事件、`Frameless` 无边框窗口 + 页内绘制标题栏与窗口控制按钮（可拖拽、最小化/最大化/关闭）。

桌面级能力：**IMAP IDLE 实时新邮件推送**（独立连接、25 分钟自动重连、不支持时降级轮询）、**系统通知**（发件人+主题，点击聚焦窗口）、**托盘菜单**（显示/退出）、程序化生成的**应用图标**（窗口/任务栏/托盘）、添加账号时的 **IMAP+SMTP 双验证**。支持 Gmail、QQ 邮箱、163、126、Outlook、iCloud、企业邮箱等任意 IMAP/SMTP 服务，可同时挂载多个账号，聚合到统一收件箱。

UI 遵循 **Agent Desktop UI 设计系统**：浮动卡片画布（14px 圆角卡片 + 8px 环绕间隙）、无边框海拔阴影（浅色复合投影 / 深色顶部发光描边）、纯色窗口底 + 可选 Oklab 渐变水洗（Rex / Mist / Meadow / Dusk）、内嵌分组设置列表、分段控件与开关、无遮罩可拖拽设置面板、滚动磨砂卡片头。

![tech](https://img.shields.io/badge/Node.js-%E2%89%A518-339933) ![proto](https://img.shields.io/badge/IMAP%2FSMTP-SSL%2FTLS-6d5afc)

## 快速开始

```bash
go build -o gloss-mail.exe .               # 需要 Go 1.27+，Win10/11 自带 WebView2
./gloss-mail.exe                           # 打开桌面窗口
./gloss-mail.exe --serve 127.0.0.1:3731    # 同时在浏览器提供访问（调试用）
```

首次启动进入引导页，添加邮箱账号（支持授权码/应用密码）即可开始收发。

## 添加真实邮箱

点左下角「添加邮箱账号」，选择服务商：

| 服务商 | IMAP | SMTP | 登录凭据 |
|---|---|---|---|
| QQ 邮箱 | imap.qq.com:993 | smtp.qq.com:465 | **授权码**（网页版 设置→账户→开启 IMAP） |
| Gmail | imap.gmail.com:993 | smtp.gmail.com:465 | **应用专用密码**（需两步验证） |
| 163 / 126 | imap.163.com:993 | smtp.163.com:465 | **客户端授权密码** |
| Outlook / M365 | outlook.office365.com:993 | smtp.office365.com:587 | 应用密码 |
| iCloud | imap.mail.me.com:993 | smtp.mail.me.com:587 | App 专用密码 |
| 企业微信邮箱 | imap.exmail.qq.com:993 | smtp.exmail.qq.com:465 | 邮箱密码 |
| 阿里企业邮箱 | imap.mxhichina.com:993 | smtp.qiye.aliyun.com:465 | 邮箱密码 |
| 自定义 | 任意 | 任意 | — |

添加向导会先真实连接 IMAP 验证凭据，再尝试 SMTP，全部通过后账号挂载进侧栏。多个账号按颜色区分，头部头像堆叠可快速切换。

## 功能一览

**邮件收发**
- 多账号挂载，统一收件箱聚合 + 星标智能视图
- IMAP IDLE 实时推送（SSE），新邮件即时提醒
- 读信（HTML 正文沙箱渲染、远程图片默认屏蔽、内联 CID 图片）
- 写信 / 回复 / 回复全部 / 转发，富文本工具栏，附件拖拽，Ctrl+Enter 发送
- 已发送自动归档到 Sent 文件夹
- 标星、已读/未读、归档、删除（进废纸篓）、文件夹间移动
- 站内搜索 + 未读/星标/附件筛选，日期分组（今天/昨天/本周）

**体验**
- Agent Desktop UI 设计系统：邮件列表 + 阅读面板合并为一张 14px 浮动卡片（内部发丝线分隔），侧栏直接落在窗口底色上，8px 环绕间隙 / 无边框海拔
- 浅色 / 深色 / 跟随系统主题 + 4 种 Oklab 渐变水洗背景（设置 → 外观，实时预览）
- 列表筛选为分段控件；设置面板为无遮罩浮动窗口（可拖拽、实时预览），含完整偏好中心：
  - 外观：主题 / 列表密度（舒适·紧凑）/ Oklab 背景水洗
  - 阅读：自动标为已读 / 自动加载远程图片 / 回复时引用原文
  - 通知：新邮件系统通知开关（写入 data/prefs.json，后端实时生效）
  - 账号：上次同步时间、账号编辑/删除、邮件缓存清除、演示数据重置
- 完整键盘流：`J/K` 导航、`Enter` 打开、`R/A/F` 回复转发、`S` 星标、`E` 归档、`Del` 删除、`/` 搜索、`?` 查看全部
- 列表右键菜单、hover 快捷星标
- 响应式：桌面单卡片双栏 → 平板阅读全屏 → 手机单栏 + 抽屉侧栏
- 骨架屏、空状态插画、Toast、连接状态指示灯

**隐私**
- 凭据以 AES-256-GCM 加密存储在本机 `data/secret.key` 派生的密钥下，绝不回传浏览器
- 邮件缓存（信封/正文/附件）全部落在本机 `data/cache/`
- 不内置任何统计或遥测

## 架构

```
gloss-mail/
├── main.go               mygo 桌面入口：embed 前端 / api 协议 / Frameless 窗口
├── internal/             Go 后端
│   ├── api/              REST 路由（与 Node 版端点一一对应）
│   ├── mailx/            IMAP 层（go-imap v2）：连接池/同步缓存/解析/标记
│   ├── smtpx/            发信（MIME 构造 + net/smtp + Sent 归档）
│   ├── store/            账号存储（JSON + AES-256-GCM）
│   ├── providers/        服务商预设
│   ├── watchers/         账号状态 + 定时同步
│   └── hub/              事件广播（mygo 事件 + SSE）
├── gloss-mail.exe        go build 产出的单文件桌面应用
├── server/               Node.js 后端
│   ├── index.js          Express API + SSE + 静态托管
│   ├── mail.js           IMAP 层（ImapFlow）：同步/信封缓存/取信/标记/移动
│   ├── smtp.js           发信（Nodemailer）+ Sent 归档
│   ├── watchers.js       每账号一条 IDLE 连接，推送新邮件与在线状态
│   ├── store.js          账号存储（JSON + AES-256-GCM）
│   ├── providers.js      服务商预设（含服务器地址与授权码指引）
│   └── demo-data.js      演示账号内容（含程序化生成的 PNG 海报附件）
├── public/               前端（原生 ES Modules，无构建步骤）
│   ├── styles.css        设计系统（双主题、三栏、组件样式）
│   └── js/               store / api / sidebar / list / reader / compose / settings
└── data/                 运行时数据（可随时删除重置）
```

- **同步策略**：按邮箱维护 UIDVALIDITY + UIDNEXT 增量同步，信封（主题/发件人/标志）缓存于本地；正文按需拉取并缓存，摘要（snippet）后台预取渐进填充列表。
- **零账号引导**：未挂载账号时列表区显示引导页，一键进入添加向导。
- **实时**：每个真实账号一条独立 IDLE 连接监听 INBOX（Go 版原生 IDLE，25 分钟重连，不支持时自动降级轮询），经 mygo 事件/SSE 广播 `mail:counts` / `account:status`，前端刷新徽标、弹出提醒并触发**系统桌面通知**。
- **安全渲染**：HTML 邮件经 DOMParser 白名单式清洗（剥离 script/iframe/on*），在 `sandbox` iframe 中渲染（无脚本执行、禁顶层导航），远程图片默认替换为占位。

## 常用配置

```bash
PORT=8080 npm start        # 自定义端口（默认 3728）
HOST=0.0.0.0 npm start     # 允许局域网访问（默认仅本机）
```

删除 `data/` 目录即可完全重置（账号、缓存、演示数据）。

## 说明

- UI 布局灵感来自 [Gloss Mail](https://gloss.email)，本项目为独立实现，与其无关联。
- Gmail/QQ/163 等服务商要求「授权码 / 应用专用密码」而非网页登录密码，向导内有逐步指引。
- 适合个人自部署使用；如需暴露到公网请自行加反向代理与鉴权。
