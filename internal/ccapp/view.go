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
				ui.Column(c).Width(280).Shrink(0).FillHeight().Children(func() {
					a.sidebar(c, p, false)
				})
				a.card(c, p, true)
			})
			return
		}
		ui.Row(c).Grow(1).Gap(10).Padding(0, 10, 10, 10).AlignItems(ui.Stretch).Children(func() {
			if !narrow {
				host := ui.Column(c).Key("sidebar-host")
				target := a.sidebarW
				if a.prefs.SidebarShut {
					// 52 keeps the rail's icon column on the expanded
					// sidebar's icon column (host inset 10 + 26)
					target = 52
				}
				w := host.Animate("width", target, 200*time.Millisecond)
				host.Width(w).Shrink(0).FillHeight().Clip()
				host.Children(func() {
					// the rail keeps its final column so the icons never
					// move; only the gutter around them closes
					if a.prefs.SidebarShut {
						ui.Column(c).Width(52).FillHeight().
							Opacity(clamp01((a.sidebarW-w)/96 + 0.3)).Children(func() {
							a.sidebar(c, p, true)
						})
						return
					}
					ui.Column(c).Width(a.sidebarW).FillHeight().Children(func() {
						a.sidebar(c, p, false)
					})
				})
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
		// left wing only clears the window controls; the sidebar toggle
		// sits right beside the search field
		ui.Row(c).Grow(1).AlignItems(ui.Center)

		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			a.sidebarToggle(c, p, narrow)
			a.searchBar(c, p, compact, narrow)
			// balance the toggle so the field keeps the window's center line
			ui.Box(c).Width(38).Height(1)
		})

		ui.Row(c).Grow(1).Justify(ui.End).Gap(4).AlignItems(ui.Center).Children(func() {
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
func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// toggleSidebar collapses the sidebar to its rail, or opens the drawer when
// the window is too narrow for one.
// sidebarToggle collapses the sidebar to its rail, or opens the drawer on a
// narrow window.
func (a *App) sidebarToggle(c *ui.Context, p palette, narrow bool) {
	btn := ui.ButtonBase(c).Size(30, 30).Radius(8).Center()
	if btn.Hovered() {
		btn.Background(p.ChipDeep)
	}
	if narrow {
		btn.Label("打开侧栏").Tooltip("打开侧栏 (⌘B)")
		btn.Children(func() { ui.Icon(c, iconMenu).FontSize(18).TextColor(p.Muted) })
		btn.OnClick(func() { a.sidebarOpen = !a.sidebarOpen })
		return
	}
	shut := a.prefs.SidebarShut
	btn.Label(map[bool]string{true: "展开侧栏", false: "收起侧栏"}[shut]).
		Tooltip(map[bool]string{true: "展开侧栏 (⌘B)", false: "收起侧栏 (⌘B)"}[shut])
	icon := iconSidebarCollapse
	if shut {
		icon = iconSidebarExpand
	}
	btn.Children(func() { ui.Icon(c, icon).FontSize(18).TextColor(p.Muted) })
	btn.OnClick(func() { a.toggleSidebar() })
}

func (a *App) toggleSidebar() {
	a.prefs.SidebarShut = !a.prefs.SidebarShut
	a.savePrefs()
}

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
