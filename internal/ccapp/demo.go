package ccapp

import (
	"time"

	"ccmail/internal/mailx"
)

// LoadDemoData fills the app with sample mail so the interface can be
// previewed and screenshotted without touching a real mailbox. It never
// writes to the store: everything lives in the app's state.
func (a *App) LoadDemoData() {
	a.accounts = []*AccountVM{
		{ID: "demo-work", Name: "工作邮箱", Email: "me@work.com", Color: "#3b6cf6", Status: "online",
			Folders: []Folder{
				{AccountID: "demo-work", Path: "INBOX", Name: "收件箱", Kind: "inbox", Unread: 3, Total: 128},
				{AccountID: "demo-work", Path: "Drafts", Name: "草稿", Kind: "drafts"},
				{AccountID: "demo-work", Path: "Sent", Name: "已发送", Kind: "sent"},
				{AccountID: "demo-work", Path: "Archive", Name: "归档", Kind: "archive"},
				{AccountID: "demo-work", Path: "Junk", Name: "垃圾邮件", Kind: "junk"},
				{AccountID: "demo-work", Path: "Trash", Name: "废纸篓", Kind: "trash"},
				{AccountID: "demo-work", Path: "Projects", Name: "项目讨论", Kind: "custom"},
			}},
		{ID: "demo-home", Name: "个人邮箱", Email: "me@home.com", Color: "#0f8199", Status: "connecting",
			Folders: []Folder{
				{AccountID: "demo-home", Path: "INBOX", Name: "收件箱", Kind: "inbox", Unread: 2, Total: 64},
				{AccountID: "demo-home", Path: "Sent", Name: "已发送", Kind: "sent"},
				{AccountID: "demo-home", Path: "Trash", Name: "废纸篓", Kind: "trash"},
			}},
	}
	a.unreadTotal = 5
	a.view = ViewSel{Kind: "unified", Name: "全部收件箱"}
	a.sidebarSel = "unified"

	now := time.Now()
	specs := []struct {
		from    mailx.Addr
		subject string
		snippet string
		unread  bool
		star    bool
		att     bool
		ago     time.Duration
		acct    int
	}{
		{mailx.Addr{Name: "张伟", Email: "zhang@example.com"}, "周会纪要：Q4 路线图与资源分配",
			"本周确定的三件事：灰度节奏、埋点口径、以及设计稿的最终交接时间。", true, false, true, 20 * time.Minute, 0},
		{mailx.Addr{Name: "Alice Chen", Email: "alice@example.com"}, "Re: 设计稿评审反馈",
			"我把列表密度的两个版本都放进去了，麻烦看下第二版。", true, true, false, 2 * time.Hour, 1},
		{mailx.Addr{Name: "李娜", Email: "lina@example.com"}, "发票与报销流程更新（请查收）",
			"下个月起统一走线上流程，纸质发票不再收取。", false, false, true, 20 * time.Hour, 1},
		{mailx.Addr{Name: "Bob Smith", Email: "bob@example.com"}, "Weekend hiking plan",
			"Saturday looks clear. Meet at the trailhead at 8?", false, false, false, 26 * time.Hour, 0},
		{mailx.Addr{Name: "GitHub", Email: "noreply@github.com"}, "[ccmail] 3 new pull requests",
			"Review requested: native-ui, imap-idle-reconnect, theme-tokens.", true, false, false, 30 * time.Hour, 0},
		{mailx.Addr{Name: "王芳", Email: "wang@example.com"}, "转发：客户侧需求整理",
			"附件里是客户原始描述，重点在第 4 页。", false, true, true, 52 * time.Hour, 1},
	}
	a.msgs = nil
	for i, s := range specs {
		acct := a.accounts[s.acct]
		a.msgs = append(a.msgs, &MsgRef{
			AccountID: acct.ID, AccountName: acct.Name, Color: acct.Color, Path: "INBOX",
			Env: &mailx.Envelope{
				UID:     uint32(100 + i),
				Subject: s.subject,
				From:    s.from,
				Date:    now.Add(-s.ago).Format(time.RFC1123Z),
				Flags:   mailx.EnvelopeFlags{Seen: !s.unread, Flagged: s.star},
				Snippet: s.snippet,
			},
		})
	}
	a.total = len(a.msgs)
	a.rebuildRows()

	a.sel = a.msgs[1]
	a.msg = demoMessage(a.sel, now)
	a.syncListSelection()
}

// demoMessage is a rich sample body, with the shapes real newsletters use.
func demoMessage(ref *MsgRef, now time.Time) *mailx.FullMessage {
	return &mailx.FullMessage{
		UID:     ref.Env.UID,
		Subject: ref.Env.Subject,
		From:    ref.Env.From,
		To:      []mailx.Addr{{Name: "我", Email: "me@work.com"}},
		Cc:      []mailx.Addr{{Name: "设计组", Email: "design@work.com"}},
		Date:    ref.Env.Date,
		HTML: `<p>先说结论：<b>第二版更稳</b>。</p>
<p>第一版的间距让长主题行读起来有点挤，第二版把行高提到 1.45 之后，跨语言的混排也正常了。</p>
<ul>
  <li>列表密度：两档，默认宽松</li>
  <li>阅读区行高：1.75，正文 14.5px</li>
  <li>焦点状态：实色描边，避免低对比</li>
</ul>
<blockquote>如果没问题，我这周把 token 整理进设计系统文档。</blockquote>
<table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;font-size:13px">
  <tr><th>项目</th><th>状态</th><th>负责人</th></tr>
  <tr><td>列表密度</td><td>已确认</td><td>Alice</td></tr>
  <tr><td>通知文案</td><td>待定</td><td>张伟</td></tr>
</table>
<p>完整稿见 <a href="https://example.com/design-review">设计评审文档</a>。</p>
<img src="https://images.example.com/hero.png" alt="评审截图" width="520">
<p>—— Alice</p>`,
		Snippet: ref.Env.Snippet,
		Text:    "先说结论：第二版更稳。\n\n第一版的间距让长主题行读起来有点挤，第二版把行高提到 1.45 之后，跨语言的混排也正常了。\n\n- 列表密度：两档，默认宽松\n- 阅读区行高：1.75，正文 14.5px\n- 焦点状态：实色描边，避免低对比\n\n如果没问题，我这周把 token 整理进设计系统文档。\n\n—— Alice",
		Attachments: []mailx.AttachmentMeta{
			{ID: "att-1", Filename: "design-review-v2.pdf", Size: 248320, ContentType: "application/pdf"},
			{ID: "att-2", Filename: "tokens.json", Size: 4096, ContentType: "application/json"},
		},
	}
}
