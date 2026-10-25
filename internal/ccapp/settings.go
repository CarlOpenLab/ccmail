package ccapp

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ccmail/internal/version"
)

var accountColors = []string{
	"#3b6cf6", "#0f8199", "#22863a", "#b08800", "#d3302f", "#a033c4", "#5b4fd4", "#64748b",
}

// openSettings shows the settings dialog.
func (a *App) openSettings(tab int) {
	a.settingsTab = tab
	a.settings = true
}

// settingsModal draws settings over the window.
func (a *App) settingsModal(c *ui.Context, p palette) {
	open := true
	ui.Modal(c, &open, func() {
		ui.Row(c).Width(800).Height(560).Clip().Background(p.Window).Radius(16).
			Justify(ui.Start).AlignItems(ui.Stretch).
			Shadow(0, 24, 64, -12, p.Shadow).Children(func() {
			// navigation
			ui.Column(c).Width(178).Shrink(0).Padding(10, 8).Gap(2).Children(func() {
				ui.Text(c, "设置").FontSize(14.5).Bold().TextColor(p.Text).Padding(6, 10, 12, 10)
				for i, item := range [][2]string{{"账号", "user"}, {"外观", "palette"}, {"快捷键", "keyboard"}, {"关于", "mail"}} {
					i := i
					row := ui.ButtonBase(c).Height(34).Padding(0, 10).Gap(10).Radius(8).Label(item[0])
					if a.settingsTab == i {
						row.Background(p.AccentSoft)
					} else if row.Hovered() {
						row.Background(p.Chip)
					}
					row.Children(func() {
						color := p.Muted
						if a.settingsTab == i {
							color = p.Accent
						}
						icon := map[int]*ui.SVG{0: iconUser, 1: iconPalette, 2: iconKeyboard, 3: iconMail}[i]
						ui.Icon(c, icon).FontSize(17).TextColor(color)
						ui.Text(c, item[0]).FontSize(13).TextColor(color)
					})
					row.OnClick(func() { a.settingsTab = i })
				}
				ui.Spacer(c)
				ui.ButtonBase(c).Height(34).Padding(0, 10).Gap(10).Radius(8).Label("关闭设置").Children(func() {
					ui.Icon(c, iconClose).FontSize(16).TextColor(p.Muted)
					ui.Text(c, "关闭").FontSize(13).TextColor(p.Muted)
				}).OnClick(func() { a.settings = false })
			})

			ui.Box(c).Width(1).Background(p.Border)

			ui.Column(c).Grow(1).FillHeight().Background(p.Card).Children(func() {
				ui.Scroll(c).Grow(1).Padding(20, 28, 28, 28).Children(func() {
					switch a.settingsTab {
					case 0:
						a.settingsAccounts(c, p)
					case 1:
						a.settingsAppearance(c, p)
					case 2:
						a.settingsShortcuts(c, p)
					default:
						a.settingsAbout(c, p)
					}
				})
			})
		})
	})
	if !open {
		a.settings = false
	}
}

// groupTitle labels a section of settings.
func groupTitle(c *ui.Context, p palette, text string) {
	ui.Text(c, text).FontSize(11.5).Bold().LetterSpacing(0.2).TextColor(p.Faint).
		Margin(18, 0, 8, 4)
}

func (a *App) settingsAccounts(c *ui.Context, p palette) {
	groupTitle(c, p, "已挂载的账号")
	if len(a.accounts) == 0 {
		ui.Text(c, "还没有添加任何邮箱账号。").FontSize(13).TextColor(p.Muted)
	}
	for _, acc := range a.accounts {
		acc := acc
		card := ui.Column(c).Padding(12, 14).Radius(12).Background(p.Chip).Gap(8).
			Border(1, p.Border).Margin(0, 0, 10, 0)
		card.Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(34, 34).Radius(17).Center().Background(ui.Hex(acc.Color)).Children(func() {
					ui.Text(c, initials(acc.Name)).FontSize(12).Bold().TextColor(ui.Hex("#ffffff"))
				})
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					ui.Text(c, acc.Name).FontSize(13.5).Bold().TextColor(p.Text).SingleLine()
					ui.Text(c, acc.Email).FontSize(12).TextColor(p.Faint).SingleLine()
				})
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(statusColor(p, acc.Status))
					ui.Text(c, statusLabel(acc.Status)).FontSize(11).TextColor(p.Faint)
				})
				if ui.ButtonBase(c).Size(28, 28).Radius(8).Center().Label("移除账号").Tooltip("移除账号").
					Children(func() { ui.Icon(c, iconTrash).FontSize(15).TextColor(p.Danger) }).Clicked() {
					a.confirmRemove(acc)
				}
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				for _, hex := range accountColors {
					hex := hex
					sw := ui.ButtonBase(c).Size(20, 20).Radius(10).Label("颜色 " + hex)
					sw.Background(ui.Hex(hex))
					if strings.EqualFold(hex, acc.Color) {
						sw.Shadow(0, 0, 0, 2, p.Text)
					}
					sw.OnClick(func() {
						a.patchAccount(acc.ID, "", hex)
					})
				}
				ui.Spacer(c)
				ui.ButtonBase(c).Height(24).Padding(0, 8).Radius(6).Label("编辑账号").Children(func() {
					ui.Text(c, "编辑").FontSize(11.5).TextColor(p.Muted)
				}).OnClick(func() {
					a.editAccount, a.editName, a.editColor = acc.ID, acc.Name, acc.Color
				})
				if acc.Detail != "" {
					ui.Text(c, acc.Detail).FontSize(11).TextColor(p.Faint).SingleLine().MaxWidth(240)
				}
			})
			if a.editAccount == acc.ID {
				ui.Column(c).Gap(8).Padding(4, 0, 0, 0).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "显示名称").FontSize(11.5).TextColor(p.Faint).Width(60)
						ui.TextInput(c, &a.editName).Grow(1)
					})
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "账号颜色").FontSize(11.5).TextColor(p.Faint).Width(60)
						for _, hex := range accountColors {
							hex := hex
							sw := ui.ButtonBase(c).Size(20, 20).Radius(10).Label("颜色 " + hex)
							sw.Background(ui.Hex(hex))
							if strings.EqualFold(hex, a.editColor) {
								sw.Border(2, p.Text)
							}
							sw.OnClick(func() { a.editColor = hex })
						}
						ui.Spacer(c)
						ui.Button(c, "取消").OnClick(func() { a.editAccount = "" })
						ui.PrimaryButton(c, "保存").OnClick(func() {
							name := strings.TrimSpace(a.editName)
							a.patchAccount(acc.ID, name, a.editColor)
							a.editAccount = ""
							a.flash("账号已更新", "success")
						})
					})
				})
			}
		})
	}

	groupTitle(c, p, "数据与缓存")
	ui.Row(c).Padding(12, 14).Radius(12).Background(p.Chip).Border(1, p.Border).
		Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, "邮件缓存").FontSize(13).Bold().TextColor(p.Text)
			ui.Text(c, "清空已缓存的正文与附件；账号和远端邮件不受影响，下次打开会重新拉取。").
				FontSize(11.5).TextColor(p.Faint).LineHeight(1.5)
		})
		if a.clearAsk {
			ui.Button(c, "确认清除").OnClick(func() {
				a.clearAsk = false
				go func() {
					a.update(func() { a.clearCache() })
				}()
			})
			ui.Button(c, "取消").OnClick(func() { a.clearAsk = false })
		} else {
			ui.Button(c, "清除").OnClick(func() { a.clearAsk = true })
		}
	})

	ui.Row(c).Gap(8).Margin(4, 0, 0, 0).Children(func() {
		ui.PrimaryButton(c, "").Height(32).Padding(0, 14).Gap(6).Label("添加账号").Children(func() {
			ui.Icon(c, iconPlus).FontSize(14).TextColor(p.AccentText)
			ui.Text(c, "添加账号").FontSize(13).Bold().TextColor(p.AccentText)
		}).OnClick(func() {
			a.wizard = newWizard()
		})
	})
}

func (a *App) confirmRemove(acc *AccountVM) {
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  a.win,
			Type:    mygo.MessageWarning,
			Message: "移除 " + acc.Name + "？",
			Detail:  "本机缓存会一并清理，邮箱服务器上的邮件不受影响。",
			Buttons: []string{"移除", "取消"},
		})
		if err != nil || res.Button != 0 {
			return
		}
		a.update(func() { a.removeAccount(acc.ID) })
	}()
}

func (a *App) settingsAppearance(c *ui.Context, p palette) {
	groupTitle(c, p, "主题")
	themes := []string{"跟随系统", "浅色", "深色"}
	idx := map[string]int{"auto": 0, "light": 1, "dark": 2}[a.prefs.Theme]
	seg := ui.Segmented(c, &idx, themes...).Label("主题").AlignSelf(ui.Start)
	if seg.Changed() {
		a.prefs.Theme = map[int]string{0: "auto", 1: "light", 2: "dark"}[idx]
		a.savePrefs()
		applyThemePref(a.prefs.Theme)
		a.loadBody()
	}

	groupTitle(c, p, "强调色")
	ui.Row(c).Gap(14).Children(func() {
		for _, pr := range accentPresets {
			pr := pr
			hex := pr.Light
			if p.Dark {
				hex = pr.Dark
			}
			chip := ui.Column(c).Gap(5).AlignItems(ui.Center).Children(func() {
				sw := ui.Box(c).Size(24, 24).Radius(12).Background(ui.Hex(hex))
				if pr.ID == a.prefs.Accent {
					sw.Shadow(0, 0, 0, 2, p.Text)
				} else {
					sw.Border(1, p.Border)
				}
				label := ui.Text(c, pr.Name).FontSize(11)
				if pr.ID == a.prefs.Accent {
					label.TextColor(p.Text)
				} else {
					label.TextColor(p.Faint)
				}
			})
			chip.OnClick(func() {
				a.prefs.Accent = pr.ID
				a.savePrefs()
				a.loadBody()
			})
		}
	})

	groupTitle(c, p, "窗口底色")
	ui.Row(c).Gap(10).Children(func() {
		for _, w := range washPresets {
			w := w
			chip := ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
				sw := ui.Box(c).Size(52, 34).Radius(9)
				if g, ok := washGradient(w.ID, p.Dark); ok {
					sw.LinearGradient(g)
				} else {
					sw.Background(p.Window)
				}
				if a.prefs.Wash == w.ID {
					sw.Border(2, p.Accent)
				} else {
					sw.Border(1, p.Border)
				}
				ui.Text(c, w.Name).FontSize(11).TextColor(map[bool]ui.Color{true: p.Accent, false: p.Faint}[a.prefs.Wash == w.ID])
			})
			chip.OnClick(func() {
				a.prefs.Wash = w.ID
				a.savePrefs()
			})
		}
	})

	groupTitle(c, p, "列表")
	densities := []string{"宽松", "紧凑"}
	d := 0
	if a.prefs.Density == "compact" {
		d = 1
	}
	if ui.Segmented(c, &d, densities...).Label("列表密度").AlignSelf(ui.Start).Changed() {
		a.prefs.Density = map[int]string{0: "cozy", 1: "compact"}[d]
		a.savePrefs()
	}

	groupTitle(c, p, "阅读与通知")
	a.prefToggle(c, p, "打开邮件后标为已读", &a.prefs.MarkRead)
	a.prefToggle(c, p, "新邮件桌面通知", &a.prefs.Notifications)
	a.prefToggle(c, p, "自动加载远程图片", &a.prefs.AutoImages)
	a.prefToggle(c, p, "回复时引用原文", &a.prefs.QuoteReply)
}

func (a *App) prefToggle(c *ui.Context, p palette, label string, field *bool) {
	ui.Row(c).Padding(9, 4).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(p.Text).Grow(1)
		if ui.Switch(c, field).Label(label).Changed() {
			a.savePrefs()
		}
	})
}

func (a *App) settingsShortcuts(c *ui.Context, p palette) {
	groupTitle(c, p, "键盘")
	rows := [][2]string{
		{"J / K", "下一封 / 上一封"},
		{"Enter", "打开选中的邮件"},
		{"N", "写一封新邮件"},
		{"R / A / F", "回复 / 回复全部 / 转发"},
		{"S", "加星标"},
		{"E", "归档"},
		{"⌫", "删除"},
		{"/ 或 ⌘K", "搜索邮件"},
		{"Esc", "关闭阅读区或对话框"},
		{"?", "打开快捷键"},
	}
	for _, row := range rows {
		ui.Row(c).Padding(8, 4).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Text(c, row[0]).FontSize(12).Bold().TextColor(p.Muted).Width(120)
			ui.Text(c, row[1]).FontSize(13).TextColor(p.Text)
		})
		ui.Divider(c)
	}
}

func (a *App) settingsAbout(c *ui.Context, p palette) {
	ui.Column(c).Fill().Center().Gap(8).Padding(40).Children(func() {
		ui.Box(c).Size(72, 72).Radius(22).Center().
			Gradient(p.Accent, p.Accent.Mix(p.Text, 0.25), 145).Children(func() {
			ui.Icon(c, iconMail).FontSize(32).TextColor(p.AccentText)
		})
		ui.Text(c, appName).FontSize(19).Bold().TextColor(p.Text)
		ui.Textf(c, "v%s · 多账号 IMAP / SMTP 邮件客户端", version.Version).FontSize(12).TextColor(p.Faint)
		ui.Text(c, "基于 Go + mygo 原生控件构建：没有 webview，没有前端构建；凭据以 AES-256-GCM 加密存储本机，无统计、无遥测。").
			FontSize(12.5).TextColor(p.Muted).TextAlign(ui.Center).MaxWidth(420).LineHeight(1.7).Margin(8, 0, 0)
	})
}

func statusLabel(status string) string {
	switch status {
	case "online":
		return "在线"
	case "connecting":
		return "连接中"
	case "offline":
		return "离线"
	case "error":
		return "连接失败"
	default:
		return "未连接"
	}
}

func statusColor(p palette, status string) ui.Color {
	switch status {
	case "online":
		return p.Success
	case "connecting":
		return p.Warning
	case "offline", "error":
		return p.Danger
	default:
		return p.Faint
	}
}

// washPresets are the window tints the appearance tab offers.
var washPresets = []struct {
	ID   string
	Name string
}{
	{"plain", "纯色"},
	{"rex", "Rex"},
	{"mist", "Mist"},
	{"meadow", "Meadow"},
	{"dusk", "Dusk"},
}

type washStop struct {
	light, dark [4]string
}

var washes = map[string]washStop{
	"rex":    {[4]string{"#ebe0f3", "#f3e7f1", "#f7eee4", "#f6efcf"}, [4]string{"#17141f", "#191620", "#1a1818", "#1b1916"}},
	"mist":   {[4]string{"#e3e9f4", "#ebeff6", "#f1f2f3", "#f2f1ec"}, [4]string{"#12151b", "#14171d", "#16171b", "#17181b"}},
	"meadow": {[4]string{"#e2eee5", "#ebf2e7", "#f3f2e2", "#f5f0d6"}, [4]string{"#111814", "#141a16", "#171914", "#181a14"}},
	"dusk":   {[4]string{"#e4dbf3", "#efdeee", "#f6e3dc", "#f7e6cf"}, [4]string{"#191321", "#1e161e", "#201a17", "#221a17"}},
}

// washGradient builds the window's Oklab-tuned tint.
func washGradient(id string, dark bool) (ui.LinearGradient, bool) {
	w, ok := washes[id]
	if !ok {
		return ui.LinearGradient{}, false
	}
	cols := w.light
	if dark {
		cols = w.dark
	}
	return ui.LinearGradient{
		Angle: 160,
		Oklab: true,
		Stops: []ui.GradientStop{
			{Color: ui.Hex(cols[0]), At: 0},
			{Color: ui.Hex(cols[1]), At: 0.34},
			{Color: ui.Hex(cols[2]), At: 0.67},
			{Color: ui.Hex(cols[3]), At: 1},
		},
	}, true
}
