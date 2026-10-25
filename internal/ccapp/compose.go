package ccapp

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"ccmail/internal/mailx"
	"ccmail/internal/smtpx"
	"ccmail/internal/store"
)

// Attachment is a file the user attached to the draft.
type Attachment struct {
	Path string
	Name string
	Size int64
}

// Compose is one open compose sheet: new mail, a reply or a forward.
type Compose struct {
	Mode    string // new | reply | replyAll | forward
	From    int
	To      string
	Cc      string
	Bcc     string
	Subject string

	ShowCc  bool
	ShowBcc bool

	Doc *richtext.Document

	Atts []Attachment

	Sending bool
	Err     string

	ref *MsgRef
}

// openCompose opens the sheet, prefilled for a reply or forward.
func (a *App) openCompose(mode string, ref *MsgRef) {
	if len(a.accounts) == 0 {
		a.wizard = newWizard()
		return
	}
	cs := &Compose{Mode: mode, ref: ref}
	if ref != nil {
		for i, acc := range a.accounts {
			if acc.ID == ref.AccountID {
				cs.From = i
			}
		}
	}
	if a.msg != nil && ref != nil && a.msg.From.Email != "" {
		addr := mailx.Addr{Name: a.msg.From.Name, Email: a.msg.From.Email}
		switch mode {
		case "reply":
			cs.To = addrLine([]mailx.Addr{addr})
			cs.Subject = replySubject(a.msg.Subject)
		case "replyAll":
			to := []mailx.Addr{addr}
			self := a.accounts[cs.From].Email
			for _, r := range append(append([]mailx.Addr{}, a.msg.To...), a.msg.Cc...) {
				if !strings.EqualFold(r.Email, self) && !strings.EqualFold(r.Email, a.msg.From.Email) {
					to = append(to, r)
				}
			}
			cs.To = addrLine(to)
			cs.Subject = replySubject(a.msg.Subject)
		case "forward":
			cs.Subject = "转发：" + strings.TrimPrefix(strings.TrimPrefix(a.msg.Subject, "转发："), "Fwd: ")
		}
	}
	doc := richtext.NewDocument(richtext.NewText(""))
	if a.prefs.QuoteReply && a.msg != nil && (mode == "reply" || mode == "replyAll" || mode == "forward") {
		intro := fmt.Sprintf("在 %s，%s 写道：\n", quoteDate(a.msg.Date), displayName(a.msg.From))
		if mode == "forward" {
			intro = fmt.Sprintf("---------- 转发邮件 ----------\n发件人：%s\n日期：%s\n主题：%s\n\n",
				displayName(a.msg.From), quoteDate(a.msg.Date), a.msg.Subject)
		}
		quoted := quotedText(a.msg)
		doc.InsertText(intro)
		doc.Insert(quoted)
		doc.SetSelection(0, 0)
	}
	cs.Doc = doc
	a.compose = cs
}

func replySubject(s string) string {
	if strings.HasPrefix(s, "Re:") || strings.HasPrefix(s, "回复：") || strings.HasPrefix(s, "回复:") {
		return s
	}
	return "Re: " + s
}

// quoteDate writes the original's date in full for a reply's attribution.
func quoteDate(s string) string {
	if t, ok := parseDate(s); ok {
		return t.Format("2006年1月2日 15:04")
	}
	return fmtFullDate(s)
}

// quotedText returns the original as a quotation block, keeping its marks.
func quotedText(m *mailx.FullMessage) richtext.Text {
	if m == nil {
		return richtext.NewText("")
	}
	var t richtext.Text
	if m.HTML != "" {
		if clean, _, err := sanitizeEmail(m.HTML, m, false); err == nil {
			t = richtext.ParseHTML(clean)
		}
	}
	if t.PlainText() == "" && m.Text != "" {
		t = richtext.NewText(m.Text)
	}
	if t.PlainText() == "" {
		return t
	}
	n := len([]rune(t.PlainText()))
	return t.FormatParagraphs(0, n, func(p *richtext.ParagraphStyle) { p.Quote = true })
}

// composeModal draws the sheet over the window.
func (a *App) composeModal(c *ui.Context, p palette) {
	open := true
	ui.Modal(c, &open, func() {
		cs := a.compose
		if cs == nil {
			return
		}
		panel := ui.Column(c).Width(680).MaxHeight(640).Clip().Background(p.Popover).Radius(16).
			Border(1, p.Border).Shadow(0, 22, 56, -12, p.Shadow)
		panel.Children(func() {
			// header
			ui.Row(c).Padding(16, 18).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, composeTitle(cs.Mode)).FontSize(15.5).Bold().TextColor(p.Text).Grow(1)
				a.iconButton(c, iconClose, "关闭 (Esc)", p, func() { a.compose = nil })
			})
			ui.Divider(c)

			// fields
			ui.Column(c).Children(func() {
				a.composeField(c, p, "发件人", func() {
					names := make([]string, 0, len(a.accounts))
					chosen := ""
					for i, acc := range a.accounts {
						names = append(names, acc.Name+" <"+acc.Email+">")
						if i == cs.From {
							chosen = names[i]
						}
					}
					sel := ui.Select(c, &chosen, names)
					if sel.Changed() {
						for i, name := range names {
							if name == chosen {
								cs.From = i
							}
						}
					}
					ui.Spacer(c)
					ui.Button(c, "抄送").OnClick(func() { cs.ShowCc = !cs.ShowCc })
					ui.Button(c, "密送").OnClick(func() { cs.ShowBcc = !cs.ShowBcc })
				})
				a.composeField(c, p, "收件人", func() {
					ui.TextInput(c, &cs.To).Placeholder("name@example.com，多人用逗号分隔").Grow(1)
				})
				if cs.ShowCc {
					a.composeField(c, p, "抄送", func() {
						ui.TextInput(c, &cs.Cc).Grow(1)
					})
				}
				if cs.ShowBcc {
					a.composeField(c, p, "密送", func() {
						ui.TextInput(c, &cs.Bcc).Grow(1)
					})
				}
				a.composeField(c, p, "主题", func() {
					ui.TextInput(c, &cs.Subject).Grow(1)
				})
			})
			ui.Divider(c)

			// formatting toolbar
			ui.Row(c).Padding(4, 14).Gap(2).AlignItems(ui.Center).Children(func() {
				f := cs.Doc.Formatting()
				tool := func(icon *ui.SVG, label string, on bool, run func()) {
					b := ui.ButtonBase(c).Label(label).Tooltip(label).KeepFocus().
						Size(30, 28).Radius(6).Center()
					switch {
					case on:
						b.Background(p.AccentSoft)
					case b.Hovered():
						b.Background(p.ChipDeep)
					}
					b.Children(func() {
						color := p.Muted
						if on {
							color = p.Accent
						}
						ui.Icon(c, icon).FontSize(15).TextColor(color)
					})
					if b.Clicked() {
						run()
					}
				}
				tool(iconBold, "加粗 (⌘B)", f.Bold == richtext.FormatOn, cs.Doc.ToggleBold)
				tool(iconItalic, "倾斜 (⌘I)", f.Italic == richtext.FormatOn, cs.Doc.ToggleItalic)
				tool(iconUnderline, "下划线 (⌘U)", f.Underline == richtext.FormatOn, cs.Doc.ToggleUnderline)
				ui.Box(c).Width(1).Height(18).Background(p.Border).Margin(0, 6)
				tool(iconList, "项目符号", f.Paragraph.List != richtext.NoList, func() { cs.Doc.ToggleList(richtext.Bullets) })
				tool(iconClip, "添加附件", false, func() { a.pickAttachments() })
			})
			ui.Divider(c)

			// body, which also takes dropped files as attachments
			body := ui.Box(c).Grow(1).MinHeight(220).Clip()
			if files := body.DroppedFiles(); len(files) > 0 {
				cs.addFiles(files)
			}
			if body.FileDragOver() {
				body.Border(2, p.Accent)
			}
			body.Children(func() {
				rt := richtext.DefaultTheme(c.Theme())
				rt.Link = p.Accent
				rt.Selection = p.Selected
				rt.QuoteBar = p.Faint.Alpha(0.55)
				rt.QuoteText = p.Muted
				rt.QuoteBarWidth = 3
				richtext.Editor(c, cs.Doc, richtext.EditorOptions{
					Placeholder: "写下你想说的…",
					Theme:       rt,
				}).Grow(1).Padding(12, 18)
			})

			if len(cs.Atts) > 0 {
				ui.Row(c).Padding(8, 16).Gap(8).Wrap().Children(func() {
					for i, att := range cs.Atts {
						i, att := i, att
						chip := ui.Row(c).Padding(4, 8).Radius(8).Background(p.Chip).Gap(6).AlignItems(ui.Center)
						chip.Children(func() {
							ui.Icon(c, iconClip).FontSize(13).TextColor(p.Faint)
							ui.Text(c, att.Name).FontSize(12).TextColor(p.Text).SingleLine().MaxWidth(180)
							if ui.ButtonBase(c).Size(20, 20).Radius(6).Center().Label("移除附件").Children(func() {
								ui.Icon(c, iconClose).FontSize(12).TextColor(p.Faint)
							}).Clicked() {
								cs.Atts = append(cs.Atts[:i], cs.Atts[i+1:]...)
							}
						})
					}
				})
			}

			if cs.Err != "" {
				ui.Text(c, cs.Err).FontSize(12).TextColor(p.Danger).Padding(0, 18, 6, 18)
			}

			ui.Divider(c)
			ui.Row(c).Padding(12, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				if cs.Sending {
					ui.Spinner(c)
					ui.Text(c, "正在发送…").FontSize(12).TextColor(p.Faint)
				} else {
					ui.PrimaryButton(c, "").Height(32).Padding(0, 14).Gap(7).Label("发送").
						Disabled(cs.Sending).Children(func() {
						ui.Icon(c, iconSend).FontSize(14).TextColor(p.AccentText)
						ui.Text(c, "发送").FontSize(13).Bold().TextColor(p.AccentText)
					}).OnClick(func() { a.sendCompose() })
					ui.Button(c, "取消").OnClick(func() { a.compose = nil })
				}
				ui.Spacer(c)
				ui.Text(c, "⌘ + Enter 发送 · 拖拽文件到正文以添加附件").FontSize(11).TextColor(p.Faint)
			})
		})
		if c.Shortcut(ui.Cmd, ui.KeyEnter) {
			a.sendCompose()
		}
	})
	if !open && a.compose != nil {
		a.compose = nil
	}
}

func composeTitle(mode string) string {
	switch mode {
	case "reply":
		return "回复邮件"
	case "replyAll":
		return "回复全部"
	case "forward":
		return "转发邮件"
	default:
		return "新邮件"
	}
}

// composeField is one labelled row of the sheet.
func (a *App) composeField(c *ui.Context, p palette, label string, fn func()) {
	ui.Row(c).Padding(0, 18).Height(44).Gap(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).
		BorderColor(p.Border).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(p.Faint).Width(46)
		fn()
	})
}

// addFiles records files as attachments of the draft.
func (cs *Compose) addFiles(paths []string) {
	seen := map[string]bool{}
	for _, att := range cs.Atts {
		seen[att.Path] = true
	}
	for _, path := range paths {
		if seen[path] {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		cs.Atts = append(cs.Atts, Attachment{Path: path, Name: filepath.Base(path), Size: info.Size()})
	}
}

// pickAttachments adds files chosen in the system dialog.
func (a *App) pickAttachments() {
	cs := a.compose
	if cs == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   a.win,
			Title:    "添加附件",
			Multiple: true,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.update(func() {
			if a.compose != cs {
				return
			}
			cs.addFiles(paths)
		})
	}()
}

// sendCompose sends the sheet's message.
func (a *App) sendCompose() {
	cs := a.compose
	if cs == nil || cs.Sending {
		return
	}
	to := parseRecipients(cs.To)
	if len(to) == 0 {
		cs.Err = "请填写收件人"
		return
	}
	if cs.From < 0 || cs.From >= len(a.accounts) {
		cs.Err = "请选择发件账号"
		return
	}
	accountID := a.accounts[cs.From].ID

	html := ""
	if t := cs.Doc.Text(); t.PlainText() != "" {
		html = t.HTML()
	}
	atts := make([]smtpx.Attachment, 0, len(cs.Atts))
	for _, f := range cs.Atts {
		data, err := os.ReadFile(f.Path)
		if err != nil {
			cs.Err = "读取附件失败：" + err.Error()
			return
		}
		atts = append(atts, smtpx.Attachment{
			Filename:    f.Name,
			ContentType: mimeOf(f.Name),
			ContentB64:  base64Of(data),
		})
	}
	out := &smtpx.Outgoing{
		To:          to,
		Cc:          parseRecipients(cs.Cc),
		Bcc:         parseRecipients(cs.Bcc),
		Subject:     cs.Subject,
		HTML:        html,
		Text:        cs.Doc.Text().PlainText(),
		Attachments: atts,
	}
	cs.Sending = true
	cs.Err = ""

	go func() {
		acct := store.Get(accountID)
		if acct == nil {
			a.update(func() {
				cs.Sending = false
				cs.Err = "账号不存在"
			})
			return
		}
		_, err := smtpx.Send(acct, out)
		a.update(func() {
			cs.Sending = false
			if err != nil {
				cs.Err = err.Error()
				return
			}
			a.compose = nil
			a.flash("邮件已发送", "success")
		})
	}()
}

// parseRecipients reads comma or semicolon separated addresses.
func parseRecipients(s string) []smtpx.Recipient {
	var out []smtpx.Recipient
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, email := "", part
		if i := strings.LastIndexByte(part, '<'); i >= 0 && strings.HasSuffix(part, ">") {
			name = strings.Trim(strings.TrimSpace(part[:i]), `"'`)
			email = strings.TrimSpace(part[i+1 : len(part)-1])
		}
		if !strings.Contains(email, "@") {
			continue
		}
		out = append(out, smtpx.Recipient{Name: name, Email: email})
	}
	return out
}

func base64Of(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func mimeOf(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md":
		return "text/plain; charset=utf-8"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
