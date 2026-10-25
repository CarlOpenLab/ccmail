package ccapp

import (
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"

	"ccmail/internal/providers"
)

// View builds the window: title bar, sidebar, and the card that holds the
// message list beside the reader.
func (a *App) View(c *ui.Context) {
	p, _ := themed(c, a.prefs.Accent)
	a.drainToasts(c)

	a.tickSearch()

	bar := c.TitleBar()
	width, _ := c.Size()
	compact := width < 1120
	narrow := width < 880

	root := ui.Column(c).Fill()
	a.paintWindow(c, root, p)
	root.Children(func() {
		a.shortcuts(c)
		a.titlebar(c, p, bar, compact, narrow)
		if narrow && a.sidebarOpen {
			ui.Row(c).Grow(1).Gap(10).Padding(0, 10, 10, 10).AlignItems(ui.Stretch).Children(func() {
				a.sidebar(c, p)
				a.card(c, p, true)
			})
			return
		}
		ui.Row(c).Grow(1).Gap(10).Padding(0, 10, 10, 10).AlignItems(ui.Stretch).Children(func() {
			if !narrow {
				a.sidebar(c, p)
			}
			a.card(c, p, compact)
		})
	})

	if a.settings {
		a.settingsModal(c, p)
	}
	if a.wizard != nil {
		a.wizardModal(c, p)
	}
	if a.compose != nil {
		a.composeModal(c, p)
	}
}

// drainToasts shows what background work asked to say.
func (a *App) drainToasts(c *ui.Context) {
	if len(a.toasts) == 0 {
		return
	}
	for _, t := range a.toasts {
		if t.kind == "error" {
			c.Toast(t.text)
		} else {
			c.Toast(t.text)
		}
	}
	a.toasts = nil
}

// card is the floating surface: list and reader share one rounded panel.
func (a *App) card(c *ui.Context, p palette, compact bool) {
	ui.Column(c).Grow(1).Clip().Background(p.Card).Radius(16).
		Shadow(0, 10, 28, -16, p.Shadow).Children(func() {
		if len(a.accounts) == 0 {
			a.onboarding(c, p)
			return
		}
		if compact {
			// one pane at a time, as on a narrow window
			if a.sel == nil {
				a.listPane(c, p)
			} else {
				a.readerPane(c, p, true)
			}
			return
		}
		ui.Split(c, &a.listW, func() {
			a.listPane(c, p)
		}, func() {
			a.readerPane(c, p, false)
		}).Grow(1)
	})
}

// onboarding is the first-run card: no mailboxes yet.
func (a *App) onboarding(c *ui.Context, p palette) {
	ui.Column(c).Fill().Center().Gap(10).Padding(40).Children(func() {
		ui.Box(c).Size(96, 96).Radius(28).Background(p.Chip).Center().Children(func() {
			ui.Icon(c, iconMail).FontSize(38).TextColor(p.Accent)
		})
		ui.Text(c, "欢迎来到 ccmail").FontSize(19).Bold().TextColor(p.Text)
		ui.Text(c, "添加你的第一个邮箱账号（QQ、Gmail、163、Outlook、iCloud 或自定义 IMAP），开始收发邮件。").
			FontSize(13).TextColor(p.Muted).TextAlign(ui.Center).MaxWidth(410).LineHeight(1.7)
		ui.PrimaryButton(c, "添加邮箱账号").MinWidth(160).OnClick(func() { a.openWizard() })
	})
}

// titlebar is the window chrome the app draws: brand, search, actions.
func (a *App) titlebar(c *ui.Context, p palette, bar ui.TitleBar, compact, narrow bool) {
	left := bar.Left + 12
	right := bar.Right + 12
	ui.Row(c).Height(maxFloat(bar.Height, 56)).DragWindow().Padding(0, right, 0, left).
		Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(a.sidebarW - left).Gap(8).AlignItems(ui.Center).Children(func() {
			if narrow {
				btn := ui.ButtonBase(c).Label("切换侧栏").Tooltip("切换侧栏 (Ctrl+B)").Size(30, 30).Radius(8).Center()
				if btn.Hovered() {
					btn.Background(p.ChipDeep)
				}
				btn.Children(func() { ui.Icon(c, iconMenu).FontSize(18).TextColor(p.Muted) })
				btn.OnClick(func() { a.sidebarOpen = !a.sidebarOpen })
			}
			ui.Box(c).Size(26, 26).Radius(8).Center().
				Gradient(p.Accent, p.Accent.Mix(p.Text, 0.25), 145).Children(func() {
				ui.Icon(c, iconMail).FontSize(15).TextColor(p.AccentText)
			})
			ui.Text(c, "ccmail").FontSize(15.5).Bold().LetterSpacing(-0.2).TextColor(p.Text)
		})

		ui.Row(c).Grow(1).Justify(ui.Center).Children(func() {
			a.searchBar(c, p, compact, narrow)
		})

		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			if !compact {
				a.accountDots(c, p)
			}
			a.iconButton(c, iconRefresh, "刷新列表", p, func() {
				a.reloadList(true)
				a.refreshAccounts()
			})
			a.iconButton(c, iconMoon, "切换深浅色", p, func() { a.toggleTheme() })
			a.iconButton(c, iconSettings, "设置", p, func() { a.settings = true })
			ui.Box(c).Width(1).Height(18).Background(p.Border)
			compose := ui.PrimaryButton(c, "").Height(32).Padding(0, 13).Gap(7).Label("写邮件")
			compose.Children(func() {
				ui.Icon(c, iconSend).FontSize(15).TextColor(p.AccentText)
				if !compact {
					ui.Text(c, "写邮件").FontSize(13).Bold().TextColor(p.AccentText)
				}
			})
			compose.OnClick(func() { a.openCompose("new", nil) })
		})
	})
}

// iconButton is a 32x32 icon-only button with a name and a tooltip.
func (a *App) iconButton(c *ui.Context, icon *ui.SVG, label string, p palette, onClick func()) {
	b := ui.ButtonBase(c).Label(label).Tooltip(label).Size(32, 32).Radius(9).Center()
	if b.Hovered() {
		b.Background(p.ChipDeep)
	}
	b.Children(func() {
		color := p.Muted
		if b.Hovered() {
			color = p.Text
		}
		ui.Icon(c, icon).FontSize(18).TextColor(color)
	})
	b.OnClick(onClick)
}

// accountDots shows the mounted accounts as colored initials.
func (a *App) accountDots(c *ui.Context, p palette) {
	if len(a.accounts) == 0 {
		return
	}
	ui.Row(c).Gap(3).Padding(0, 6).Children(func() {
		for i, acc := range a.accounts {
			if i == 3 {
				break
			}
			if len(acc.Folders) == 0 && acc.ID == "" {
				continue
			}
			dot := ui.Box(c).Size(27, 27).Radius(14).Center().Background(ui.Hex(acc.Color))
			dot.Tooltip(acc.Name + "\n" + acc.Email)
			dot.Children(func() {
				ui.Text(c, accountBadge(acc)).FontSize(11).Bold().TextColor(ui.Hex("#ffffff"))
			})
			_ = i
		}
	})
}

// accountBadge is the one-letter mark in the titlebar's account stack: the
// provider's monogram, else the first letter of the mail domain.
func accountBadge(acc *AccountVM) string {
	if pr := providers.ByID(acc.Provider); pr != nil && pr.Monogram != "" {
		return pr.Monogram
	}
	domain := acc.Email
	if i := strings.LastIndexByte(domain, '@'); i >= 0 {
		domain = domain[i+1:]
	}
	for _, r := range domain {
		if unicode.IsLetter(r) {
			return strings.ToUpper(string(r))
		}
	}
	return initials(acc.Name)
}

func (a *App) toggleTheme() {
	switch a.prefs.Theme {
	case "auto", "":
		a.prefs.Theme = "light"
	case "light":
		a.prefs.Theme = "dark"
	default:
		a.prefs.Theme = "auto"
	}
	a.savePrefs()
	applyThemePref(a.prefs.Theme)
	a.loadBody()
}

// searchBar is the title bar's field: magnifier, input, clear key.
// tickSearch runs the debounced search once the pause has passed.
func (a *App) tickSearch() {
	if a.queryDue && time.Since(a.queryAt) >= 200*time.Millisecond {
		a.queryDue = false
		a.reloadList(true)
	}
}

func (a *App) searchBar(c *ui.Context, p palette, compact, narrow bool) {
	width := float32(420)
	if narrow {
		width = 220
	} else if compact {
		width = 300
	}
	box := ui.Row(c).Width(width).Height(34).Radius(10).Background(p.Chip).Gap(8).
		Padding(0, 10).AlignItems(ui.Center)
	box.Children(func() {
		ui.Icon(c, iconSearch).FontSize(15).TextColor(p.Faint)
		field := ui.TextInputBase(c, &a.query).Grow(1).Placeholder("搜索邮件…").
			Label("搜索邮件").Bind(&a.search)
		if field.Changed() {
			// type-ahead: search once typing pauses
			a.queryAt = time.Now()
			a.queryDue = true
			c.After(220 * time.Millisecond)
		}
		if field.Submitted() {
			a.queryDue = false
			a.reloadList(true)
		}
		if a.query != "" {
			clear := ui.ButtonBase(c).Label("清空搜索").Tooltip("清空搜索").Size(20, 20).Radius(10).Center()
			clear.Children(func() { ui.Icon(c, iconClose).FontSize(12).TextColor(p.Faint) })
			clear.OnClick(func() {
				a.query = ""
				a.reloadList(true)
			})
		} else {
			ui.Text(c, "/").FontSize(11).TextColor(p.Faint)
		}
	})
}

// maxFloat helps size the title bar above the window controls' room.
func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
