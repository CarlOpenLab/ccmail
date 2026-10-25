package ccapp

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ccmail/internal/mailx"
	"ccmail/internal/store"
)

// readerPane shows the selected message: its header, its body, its files.
func (a *App) readerPane(c *ui.Context, p palette, compact bool) {
	ui.Column(c).Fill().Clip().Children(func() {
		switch {
		case a.sel == nil:
			a.emptyReader(c, p)
			return
		case a.msgLoading:
			ui.Column(c).Fill().Center().Gap(10).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "正在加载邮件…").FontSize(12.5).TextColor(p.Faint)
			})
			return
		case a.msgErr != "":
			ui.Column(c).Fill().Center().Gap(8).Padding(36).Children(func() {
				ui.Icon(c, iconJunk).FontSize(28).TextColor(p.Danger)
				ui.Text(c, "无法加载这封邮件").FontSize(15).Bold().TextColor(p.Text)
				ui.Text(c, a.msgErr).FontSize(12.5).TextColor(p.Muted).TextAlign(ui.Center)
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "重试").Clicked() {
						ref := a.sel
						a.openMessage(ref)
					}
					if ui.Button(c, "关闭").Clicked() {
						a.sel = nil
						a.loadBody()
					}
				})
			})
			return
		}

		m := a.msg
		_ = m

		a.readerHeader(c, p, compact)

		if a.blockedImages > 0 {
			guard := ui.Row(c).Margin(0, 20, 8, 20).Padding(8, 12).Radius(10).
				Background(p.Warning.Alpha(0.13)).Gap(10).AlignItems(ui.Center)
			guard.Children(func() {
				ui.Icon(c, iconImage).FontSize(15).TextColor(p.Warning)
				ui.Textf(c, "已屏蔽 %d 张远程图片以保护隐私", a.blockedImages).FontSize(12).
					TextColor(p.Muted).Grow(1)
				load := ui.ButtonBase(c).Height(24).Padding(0, 10).Radius(7).Label("加载图片").
					Background(p.Warning.Alpha(0.18))
				load.Children(func() { ui.Text(c, "加载图片").FontSize(11.5).TextColor(p.Text) })
				load.OnClick(func() { a.reloadImages() })
			})
		}

		if a.msg.HTML != "" && a.web != nil {
			ui.WebView(c, a.web).Grow(1)
		} else {
			ui.Scroll(c).Grow(1).Padding(4, 30, 24, 30).Children(func() {
				ui.Text(c, a.msg.Text).FontSize(14.5).TextColor(p.Text).LineHeight(1.75).Selectable()
			})
		}

		if len(a.msg.Attachments) > 0 {
			a.attachmentsStrip(c, p)
		}
	})
}

// emptyReader shows the guidance when nothing is open.
func (a *App) emptyReader(c *ui.Context, p palette) {
	ui.Column(c).Fill().Center().Gap(4).Padding(40).Children(func() {
		ui.Box(c).Size(76, 76).Radius(24).Center().Background(p.Chip).Children(func() {
			ui.Icon(c, iconMail).FontSize(32).TextColor(p.Accent)
		})
		ui.Text(c, "选择一封邮件").FontSize(16).Bold().TextColor(p.Text).Margin(14, 0, 0)
		ui.Text(c, "从列表选择一封邮件开始阅读，或按 N 写一封新邮件。").FontSize(13).
			TextColor(p.Faint).TextAlign(ui.Center).LineHeight(1.7)
		ui.Row(c).Gap(14).Margin(18, 0, 0).Children(func() {
			for _, h := range [][2]string{{"J", "下一封"}, {"K", "上一封"}, {"N", "写信"}, {"?", "快捷键"}} {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Padding(1, 6).Radius(5).Background(p.Chip).Children(func() {
						ui.Text(c, h[0]).FontSize(11).Bold().TextColor(p.Muted)
					})
					ui.Text(c, h[1]).FontSize(11.5).TextColor(p.Faint)
				})
			}
		})
	})
}

// readerHeader is the subject, sender, details and actions above the body.
func (a *App) readerHeader(c *ui.Context, p palette, compact bool) {
	m := a.msg
	env := m
	flagged := false
	if a.sel != nil {
		flagged = a.sel.Env.Flags.Flagged
	}

	ui.Column(c).Padding(18, 26, 10, 26).Gap(8).Children(func() {
		if compact {
			back := ui.ButtonBase(c).Height(26).Gap(5).Radius(7).Label("返回列表").AlignSelf(ui.Start)
			back.Children(func() {
				ui.Icon(c, iconBack).FontSize(15).TextColor(p.Muted)
				ui.Text(c, "返回列表").FontSize(12).TextColor(p.Muted)
			})
			back.OnClick(func() {
				a.sel = nil
				a.msg = nil
				a.loadBody()
			})
		}
		ui.Text(c, firstNonEmpty(env.Subject, "（无主题）")).FontSize(21).Bold().
			TextColor(p.Text).LineHeight(1.3).MaxLines(2)

		ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
			ui.Avatar(c, displayName(env.From), nil).FontSize(19)
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, displayName(env.From)).FontSize(14.5).Bold().TextColor(p.Text)
					if acc := a.accountVM(a.sel.AccountID); acc != nil {
						chip := ui.Row(c).Padding(1, 7).Radius(8).Background(p.Chip).Gap(4).AlignItems(ui.Center)
						chip.Children(func() {
							ui.Box(c).Size(5, 5).Radius(3).Background(ui.Hex(acc.Color))
							ui.Text(c, acc.Name).FontSize(11).TextColor(p.Muted)
						})
					}
				})
				ui.Text(c, env.From.Email).FontSize(12.5).TextColor(p.Faint).SingleLine()
				ui.Text(c, "收件人："+firstNonEmpty(addrLine(env.To), "我")).FontSize(12).
					TextColor(p.Faint).SingleLine()
			})
			ui.Text(c, fmtFullDate(env.Date)).FontSize(11.5).TextColor(p.Faint).Margin(0, 0, 0, 8)
		})

		ui.Row(c).Gap(4).Margin(10, 0, 0, 0).AlignItems(ui.Center).Children(func() {
			a.pill(c, p, iconReply, "回复", func() { a.openCompose("reply", a.sel) })
			if len(env.To) > 1 || len(env.Cc) > 0 {
				a.pill(c, p, iconReplyAll, "回复全部", func() { a.openCompose("replyAll", a.sel) })
			}
			a.pill(c, p, iconForward, "转发", func() { a.openCompose("forward", a.sel) })
			ui.Spacer(c)
			star := ui.ButtonBase(c).Size(30, 30).Radius(8).Center().
				Label(map[bool]string{true: "取消星标", false: "加星标"}[flagged])
			if star.Hovered() {
				star.Background(p.ChipDeep)
			}
			if flagged {
				star.Children(func() { ui.Icon(c, iconStarFill).FontSize(15).TextColor(p.Star) })
			} else {
				star.Children(func() { ui.Icon(c, iconStar).FontSize(15).TextColor(p.Muted) })
			}
			star.OnClick(func() { a.toggleStar(a.sel) })

			a.iconButton(c, iconEyeOff, "标为未读", p, func() {
				ref := a.sel
				a.markRead(ref, false)
				a.sel = nil
				a.msg = nil
				a.loadBody()
			})
			a.iconButton(c, iconArchive, "归档", p, func() { a.archiveMessage(a.sel) })
			a.iconButton(c, iconTrash, "删除", p, func() { a.deleteMessage(a.sel) })
		})
		ui.Divider(c).Margin(6, 0, 0, 0)
	})
}

// pill is a labelled action beside the message.
func (a *App) pill(c *ui.Context, p palette, icon *ui.SVG, label string, onClick func()) {
	b := ui.ButtonBase(c).Height(28).Padding(0, 10).Gap(6).Label(label).Radius(8).Background(p.Btn)
	if b.Hovered() {
		b.Background(p.BtnHover)
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(p.Muted)
		ui.Text(c, label).FontSize(12.5).TextColor(p.Text)
	})
	b.OnClick(onClick)
}

// attachmentsStrip lists the message's files as buttons that save them.
func (a *App) attachmentsStrip(c *ui.Context, p palette) {
	ui.Column(c).Padding(12, 26, 16, 26).Gap(8).Children(func() {
		ui.Textf(c, "附件 · %d", len(a.msg.Attachments)).FontSize(11).Bold().LetterSpacing(0.4).
			TextColor(p.Faint)
		ui.Row(c).Gap(8).Wrap().Children(func() {
			for _, att := range a.msg.Attachments {
				att := att
				card := ui.Row(c).Padding(6, 10).Radius(9).Background(p.Chip).Gap(8).AlignItems(ui.Center)
				card.Tooltip("保存 " + att.Filename)
				card.Children(func() {
					ui.Box(c).Size(30, 30).Radius(7).Center().Background(p.Card).Children(func() {
						ui.Text(c, extOf(att.Filename)).FontSize(9).Bold().TextColor(p.Accent)
					})
					ui.Column(c).Gap(1).Children(func() {
						ui.Text(c, att.Filename).FontSize(12).TextColor(p.Text).SingleLine().MaxWidth(170)
						ui.Text(c, fmtSize(att.Size)).FontSize(10.5).TextColor(p.Faint)
					})
				})
				card.OnClick(func() { a.saveAttachment(att) })
			}
		})
	})
}

// saveAttachment writes one attachment where the user chooses.
func (a *App) saveAttachment(att mailx.AttachmentMeta) {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      a.win,
			Title:       "保存附件",
			DefaultPath: att.Filename,
		})
		if err != nil || path == "" {
			return
		}
		src, srcErr := attachmentPath(a.sel, att)
		if srcErr != nil {
			a.update(func() { a.flash("保存失败："+srcErr.Error(), "error") })
			return
		}
		data, err := os.ReadFile(src)
		if err == nil {
			err = os.WriteFile(path, data, 0o644)
		}
		if err != nil {
			a.update(func() { a.flash("保存失败："+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash("已保存到 "+filepath.Base(path), "success") })
	}()
}

func attachmentPath(ref *MsgRef, att mailx.AttachmentMeta) (string, error) {
	if att.File != "" {
		if _, err := os.Stat(att.File); err == nil {
			return att.File, nil
		}
	}
	acct := store.Get(ref.AccountID)
	if acct == nil {
		return "", os.ErrNotExist
	}
	path, _, _, err := mailx.ResolveAttachment(acct, ref.Path, ref.Env.UID, att.ID)
	return path, err
}

func extOf(name string) string {
	ext := filepath.Ext(name)
	if len(ext) > 1 {
		return upperASCII(ext[1:minInt(len(ext)-1, 4)])
	}
	return "FILE"
}

func upperASCII(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 32
		}
	}
	return string(out)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
