package ccapp

import (
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"

	"ccmail/internal/providers"
	"ccmail/locales"
)

// View builds the window: title bar, sidebar, and the card that holds the
// message list beside the reader.
func (a *App) View(c *ui.Context) {
	p, _ := themed(c)
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
		c.AddToast(ui.Toast{Title: t.text, Type: t.kind})
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
		ui.Text(c, locales.OnboardingTitle()).FontSize(19).Bold().TextColor(p.Text)
		ui.Text(c, locales.OnboardingHint()).
			FontSize(13).TextColor(p.Muted).TextAlign(ui.Center).MaxWidth(410).LineHeight(1.7)
		a.cta(c, p, locales.SidebarAddAccount()).MinWidth(160).OnClick(func() { a.openWizard() })
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
			a.iconButton(c, iconRefresh, locales.TitlebarRefresh(), p, func() {
				a.reloadList(true)
				a.refreshAccounts()
			})
			a.iconButton(c, iconMoon, locales.TitlebarToggleTheme(), p, func() { a.toggleTheme() })
			a.iconButton(c, iconSettings, locales.MiscSettings(), p, func() { a.settings = true })
			ui.Box(c).Width(1).Height(18).Background(p.Border)
			compose := a.ctaBase(c, p, locales.TitlebarCompose()).Padding(0, 13).Gap(7)
			compose.Children(func() {
				ui.Icon(c, iconSend).FontSize(15).TextColor(p.CTAText)
				if !compact {
					ui.Text(c, locales.TitlebarCompose()).FontSize(13).Bold().TextColor(p.CTAText)
				}
			})
			compose.OnClick(func() { a.openCompose("new", nil) })
		})
	})
}

// ctaBase is the primary action button: near-black in light, near-white in
// dark, with the button role and keyboard handling of the widget.
func (a *App) ctaBase(c *ui.Context, p palette, label string) ui.Element {
	b := ui.Button(c, "").Height(32).Padding(0, 14).Gap(6).Radius(9).Label(label)
	b.Background(p.CTA)
	if b.Hovered() {
		b.Background(p.CTAHover)
	}
	b.TextColor(p.CTAText)
	return b
}

// cta is ctaBase showing a text label.
func (a *App) cta(c *ui.Context, p palette, label string) ui.Element {
	b := a.ctaBase(c, p, label)
	b.Children(func() { ui.Text(c, label).FontSize(13).Bold().TextColor(p.CTAText) })
	return b
}

func (a *App) iconButton(c *ui.Context, icon *ui.SVG, label string, p palette, onClick func()) {
	b := ui.ButtonBase(c).Label(label).Tooltip(label).Size(32, 32).Radius(9).Center()
	if b.Hovered() {
		b.Background(p.HoverStrong)
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

// accountDots shows the mounted accounts as colored initials; clicking one
// jumps to that account's inbox.
func (a *App) accountDots(c *ui.Context, p palette) {
	if len(a.accounts) == 0 {
		return
	}
	ui.Row(c).Gap(3).Padding(0, 6).Children(func() {
		for i, acc := range a.accounts {
			if i == 3 {
				break
			}
			if acc.ID == "" {
				continue
			}
			active := a.view.Kind == "folder" && a.view.AccountID == acc.ID
			dot := ui.ButtonBase(c).Size(27, 27).Radius(14).Center().Background(ui.Hex(acc.Color)).
				Label(locales.TitlebarSwitchAccount(acc.Name))
			if active {
				dot.Border(2, p.Text)
			}
			dot.Tooltip(acc.Name + "\n" + acc.Email)
			dot.Children(func() {
				ui.Text(c, accountBadge(acc)).FontSize(11).Bold().TextColor(ui.Hex("#ffffff"))
			})
			dot.OnClick(func() {
				name := locales.MiscInbox()
				for _, f := range acc.Folders {
					if f.Kind == "inbox" {
						name = f.Name
					}
				}
				a.navigate(ViewSel{Kind: "folder", AccountID: acc.ID, Path: "INBOX", Name: name})
			})
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

// clamp01 pins v to the 0–1 range.
func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// sidebarToggle collapses the sidebar to its rail, or opens the drawer on a
// narrow window.
func (a *App) sidebarToggle(c *ui.Context, p palette, narrow bool) {
	btn := ui.ButtonBase(c).Size(30, 30).Radius(8).Center()
	if btn.Hovered() {
		btn.Background(p.HoverStrong)
	}
	if narrow {
		btn.Label(locales.TitlebarOpenSidebar()).Tooltip(locales.TitlebarOpenSidebar() + " (" + modKey() + "B)")
		btn.Children(func() { ui.Icon(c, iconMenu).FontSize(18).TextColor(p.Muted) })
		btn.OnClick(func() { a.sidebarOpen = !a.sidebarOpen })
		return
	}
	shut := a.prefs.SidebarShut
	btn.Label(map[bool]string{true: locales.TitlebarExpandSidebar(), false: locales.TitlebarCollapseSidebar()}[shut]).
		Tooltip(map[bool]string{
			true:  locales.TitlebarExpandSidebar() + " (" + modKey() + "B)",
			false: locales.TitlebarCollapseSidebar() + " (" + modKey() + "B)",
		}[shut])
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

// searchBar is the title bar's field: magnifier, input, clear key.
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
		field := ui.TextInputBase(c, &a.query).Grow(1).Placeholder(locales.TitlebarSearchPlaceholder()).
			Label(locales.TitlebarSearch()).Bind(&a.search)
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
			clear := ui.ButtonBase(c).Label(locales.TitlebarClearSearch()).Tooltip(locales.TitlebarClearSearch()).Size(20, 20).Radius(10).Center()
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
