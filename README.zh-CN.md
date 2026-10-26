<p align="center"><img src="docs/logo-blue.png" width="96" alt="ccmail logo"></p>

简体中文 | [English](README.md)

# ccmail

一个安静、专注的多账号邮件客户端。支持 Gmail、QQ 邮箱、163、126、Outlook、iCloud、企业邮箱等任意 IMAP/SMTP 服务，多账号聚合到统一收件箱。

[![CI](https://github.com/CarlOpenLab/ccmail/actions/workflows/ci.yml/badge.svg)](https://github.com/CarlOpenLab/ccmail/actions/workflows/ci.yml)
[![Release](https://github.com/CarlOpenLab/ccmail/actions/workflows/release.yml/badge.svg)](https://github.com/CarlOpenLab/ccmail/actions/workflows/release.yml)

**[官网（在线介绍）](https://carlopenlab.github.io/ccmail/)** · [下载最新安装包](https://github.com/CarlOpenLab/ccmail/releases/latest)

Go + [mygo](https://github.com/egoist/mygo) 桌面应用：**界面是 mygo 原生控件（`github.com/egoist/mygo/ui`）**，没有前端构建、没有页面路由；HTML 邮件正文用 mygo 的 `WebView` 内嵌渲染（系统 WebKit / WebView2）。单一二进制，凭据 AES-256-GCM 加密存储本机。

## 快速开始

不想自己编译？直接从 [Releases](https://github.com/CarlOpenLab/ccmail/releases/latest) 下载 macOS / Windows / Linux 安装包。

```bash
go build -o ccmail .                    # Go 1.27+，直接跑
./ccmail                                # 打开原生窗口

go tool mygo build                       # 打安装包：macOS .app/.dmg、Windows Setup、Linux tar/deb
```

- macOS 已实测；Windows 需要 WebView2（Win10/11 自带），Linux 需要 WebKitGTK（HTML 邮件正文用到）。
- 首次启动是空收件箱引导页，点「添加邮箱账号」挂载账号。
- 运行数据写入当前目录 `data/`，删除即完全重置。

## 添加真实邮箱

| 服务商 | IMAP | SMTP | 登录凭据 |
|---|---|---|---|
| QQ 邮箱 | imap.qq.com:993 | smtp.qq.com:465 | **授权码**（网页版 设置 → 账户 → 开启 IMAP） |
| Gmail | imap.gmail.com:993 | smtp.gmail.com:465 | **应用专用密码**（需两步验证） |
| 163 / 126 | imap.163.com:993 | smtp.163.com:465 | **客户端授权密码** |
| Outlook / M365 | outlook.office365.com:993 | smtp.office365.com:587 | 应用密码 |
| iCloud | imap.mail.me.com:993 | smtp.mail.me.com:587 | App 专用密码 |
| 企业微信邮箱 | imap.exmail.qq.com:993 | smtp.exmail.qq.com:465 | 邮箱密码 |
| 阿里企业邮箱 | imap.mxhichina.com:993 | smtp.qiye.aliyun.com:465 | 邮箱密码 |
| 自定义 | 任意 | 任意 | — |

添加向导会真实连接 IMAP 验证凭据，再尝试 SMTP；验证通过后账号立即挂载并在后台同步。多个账号按颜色区分，标题栏头像可快速切换。

## 功能

**收发**
- 多账号统一收件箱 + 星标视图，按日期分组的原生列表
- IMAP IDLE 实时推送；不支持时降级为轮询
- HTML 正文渲染：白名单清洗（剥离脚本 / iframe / 表单 / `on*`），默认拦截远程图片并可一键加载，内联 CID 图片转 data URL
- 写信 / 回复 / 回复全部 / 转发；富文本编辑器（加粗、斜体、下划线、列表、链接、粘贴格式）、附件、`⌘ + Enter` 发送
- 已发送自动归档；标星、已读/未读、归档、删除、文件夹移动
- 站内搜索 + 未读 / 星标 / 附件筛选

**体验**
- 原生控件绘制：侧栏、消息列表、阅读区、抽屉式设置、对话框与菜单
- 浮动卡片布局：列表与阅读区合并为一张 16px 卡片，侧栏落在窗口底色上
- 浅色 / 深色 / 跟随系统；石墨单色强调（不提供彩色主题），4 种 Oklab 渐变水洗背景
- 侧栏可收成图标栏（带宽度过渡动画，`⌘B`），窄窗口自动改为抽屉
- 设置面板：单层背景、关闭按钮在右上；账号（改名 / 换色 / 移除 / 清缓存）、外观（主题 / 水洗 / 密度 / 阅读与通知）、快捷键、关于
- 桌面通知（发件人 + 主题，点击聚焦窗口）、托盘菜单、程序化应用图标
- 键盘流：`J/K` 导航、`Enter` 打开、`N` 写信、`R/A/F` 回复转发、`S` 星标、`E` 归档、`Del` 删除、`/` 或 `⌘K` 搜索、`⌘B` 收缩侧栏、`?` 快捷键
- 窄窗口自适应：列表 / 阅读单栏切换，侧栏抽屉

**隐私**
- 凭据以 AES-256-GCM 加密存储于本机 `data/secret.key` 派生的密钥下
- 邮件缓存位于 `data/cache/`；无统计、无遥测

## 架构

```
ccmail/
├── main.go               mygo 入口：原生窗口 / 托盘 / 菜单 / 截图调试
├── internal/
│   ├── ccapp/            原生界面：状态、控制器、主题与各窗格视图
│   │   ├── app.go        快捷键、窗口底色、生命周期
│   │   ├── model.go      应用状态
│   │   ├── data.go       IMAP/SMTP 调用、列表与消息加载、事件订阅
│   │   ├── view.go       窗口骨架与标题栏
│   │   ├── sidebar.go    源列表
│   │   ├── list.go       消息列表
│   │   ├── reader.go     阅读区与附件
│   │   ├── compose.go    写信表单与富文本编辑器
│   │   ├── settings.go   设置（账号 / 外观 / 快捷键 / 关于）
│   │   ├── wizard.go     添加账号向导
│   │   ├── htmlmail.go   HTML 清洗与正文文档
│   │   ├── palette.go    设计令牌（明 / 暗）
│   │   ├── icons.go      SVG 图标集
│   ├── mailx/            IMAP（go-imap v2）：连接池、增量同步、解析、标记
│   ├── smtpx/            发信（MIME 构造 + net/smtp + Sent 归档）
│   ├── store/            账号存储（JSON + AES-256-GCM）
│   ├── providers/        服务商预设
│   ├── watchers/         账号状态、定时同步、IDLE、桌面通知
│   ├── hub/              进程内事件广播（原生 UI 订阅）
│   ├── prefs/            本机偏好
│   ├── icon/             程序化生成的窗口 / 托盘 / 安装包图标
│   └── version/          应用名称与版本号
├── resources/icon.png    1024px 应用图标（mygo build 用）
├── mygo.json             打包配置：名称、identifier、版本、输出目录
├── CHANGELOG.md          版本说明，Release 从这里取
└── data/                 运行时数据（可随时删除重置）
```

- **视图模型**：`ui.View` 每帧从 `ccapp.App` 的状态构建界面；后台 goroutine 通过 `Window.Update` 回到主线程写状态并重绘。
- **同步**：按邮箱维护 UIDVALIDITY + UIDNEXT 做增量同步，信封缓存本地；正文按需拉取并缓存，摘要后台渐进填充。
- **实时**：每个真实账号一条独立 IDLE 连接监听 INBOX（25 分钟重连），事件经 `hub` 广播；界面聚合刷新状态与列表。
- **安全渲染**：HTML 先经白名单清洗（`internal/ccapp/htmlmail.go`），再交给内嵌 WebView；远程图片默认替换为占位并可手动加载，相对链接不导航。

## 打包与发布

版本号有两处，必须一致：

- `mygo.json` 的 `version`：打包、安装包与菜单里显示的版本
- `internal/version/version.go` 的 `Version`：「关于」页显示的版本
- `CHANGELOG.md`：`## [x.y.z]` 段落会成为 GitHub Release 的说明

发布流程（GitHub Actions，`.github/workflows/`）：

```bash
# 1. 改好版本号与 CHANGELOG，提交
# 2. 打 tag 并推送，Release 工作流自动构建并发布
git tag v2.2.0
git push origin v2.2.0
```

推送 tag 后：

- `create-release` 校验 tag 与 `mygo.json` / `version.go` 一致，建草稿 release 并写入 CHANGELOG 段落
- `build` 在 macOS / Linux runner 上跑 `go tool mygo build`，产出 `ccmail 2.2.0.dmg`、`ccmail Setup 2.2.0 amd64.exe`、`ccmail-2.2.0-linux-amd64.tar.gz`、`ccmail_2.2.0_amd64.deb`
- `publish` 上传产物并发布草稿

安装包目前是 ad-hoc 签名（没有 Developer ID / 代码签名证书）：macOS 首次打开需要右键「打开」，Windows 会提示 SmartScreen；需要分发签名时在 `mygo.json` 里加 `macos.signingIdentity` / `windows.certificate` 并配置对应 secrets。

日常提交与 PR 走 `CI` 工作流：`gofmt` / `go vet` / `go tool mygo vet` / `go test` / `go build`。

## 说明

- Gmail / QQ / 163 等服务商要求「授权码 / 应用专用密码」而非网页登录密码，向导内有指引，并可一键打开各服务商官方获取页面。
- 适合个人自部署；暴露公网请自行加反向代理与鉴权。
