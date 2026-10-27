package ccapp

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ccmail/internal/version"
	"ccmail/locales"
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
		panel := ui.Column(c).Width(800).Height(560).Clip().Background(p.Card).Radius(16).
			Border(1, p.Border).Shadow(0, 24, 64, -12, p.Shadow)
		panel.Children(func() {
			// one header owns the title and the close button, top right
			ui.Row(c).Height(48).Shrink(0).Padding(0, 12, 0, 20).Gap(10).
				AlignItems(ui.Center).Children(func() {
				ui.Text(c, locales.MiscSettings()).FontSize(14.5).Bold().TextColor(p.Text)
				ui.Text(c, locales.SettingsApplyNow()).FontSize(11.5).TextColor(p.Faint)
				ui.Spacer(c)
				a.iconButton(c, iconClose, locales.SettingsClose(), p, func() { a.settings = false })
			})
			ui.Divider(c)

			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Width(180).Shrink(0).Padding(8, 8).Gap(2).Children(func() {
					for i, item := range [][2]string{{locales.SettingsTabAccounts(), "user"}, {locales.SettingsTabAppearance(), "palette"}, {locales.ReaderShortcuts(), "keyboard"}, {locales.SettingsTabAbout(), "mail"}} {
						i := i
						row := ui.ButtonBase(c).Height(34).Padding(0, 10).Gap(10).Radius(8).Label(item[0])
						switch {
						case a.settingsTab == i:
							row.Background(p.ChipDeep)
						case row.Hovered():
							row.Background(p.HoverStrong)
						}
						row.Children(func() {
							color := p.Muted
							if a.settingsTab == i {
								color = p.Text
							}
							icon := map[int]*ui.SVG{0: iconUser, 1: iconPalette, 2: iconKeyboard, 3: iconMail}[i]
							ui.Icon(c, icon).FontSize(17).TextColor(color)
							text := ui.Text(c, item[0]).FontSize(13).TextColor(color)
							if a.settingsTab == i {
								text.Bold()
							}
						})
						row.OnClick(func() { a.settingsTab = i })
					}
				})

				ui.Box(c).Width(1).Background(p.Border)

				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Scroll(c).Grow(1).Padding(18, 26, 26, 26).Children(func() {
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
	groupTitle(c, p, locales.SettingsMountedAccounts())
	if len(a.accounts) == 0 {
		ui.Text(c, locales.SettingsNoAccounts()).FontSize(13).TextColor(p.Muted)
	}
	for i, acc := range a.accounts {
		acc := acc
		if i > 0 {
			ui.Divider(c)
		}
		card := ui.Column(c).Padding(12, 2).Gap(8)
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
				if ui.ButtonBase(c).Size(28, 28).Radius(8).Center().Label(locales.SettingsRemoveAccount()).Tooltip(locales.SettingsRemoveAccount()).
					Children(func() { ui.Icon(c, iconTrash).FontSize(15).TextColor(p.Danger) }).Clicked() {
					a.confirmRemove(acc)
				}
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				for _, hex := range accountColors {
					hex := hex
					sw := ui.ButtonBase(c).Size(20, 20).Radius(10).Label(locales.WizardColor() + hex)
					sw.Background(ui.Hex(hex))
					if strings.EqualFold(hex, acc.Color) {
						sw.Shadow(0, 0, 0, 2, p.Text)
					}
					sw.OnClick(func() {
						a.patchAccount(acc.ID, "", hex)
					})
				}
				ui.Spacer(c)
				ui.ButtonBase(c).Height(24).Padding(0, 8).Radius(6).Label(locales.SettingsEditAccount()).Children(func() {
					ui.Text(c, locales.SettingsEdit()).FontSize(11.5).TextColor(p.Muted)
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
						ui.Text(c, locales.WizardDisplayName()).FontSize(11.5).TextColor(p.Faint).Width(60)
						ui.TextInput(c, &a.editName).Grow(1)
					})
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, locales.WizardAccountColor()).FontSize(11.5).TextColor(p.Faint).Width(60)
						for _, hex := range accountColors {
							hex := hex
							sw := ui.ButtonBase(c).Size(20, 20).Radius(10).Label(locales.WizardColor() + hex)
							sw.Background(ui.Hex(hex))
							if strings.EqualFold(hex, a.editColor) {
								sw.Border(2, p.Text)
							}
							sw.OnClick(func() { a.editColor = hex })
						}
						ui.Spacer(c)
						ui.Button(c, locales.MiscCancel()).OnClick(func() { a.editAccount = "" })
						a.cta(c, p, locales.MiscSave()).OnClick(func() {
							name := strings.TrimSpace(a.editName)
							a.patchAccount(acc.ID, name, a.editColor)
							a.editAccount = ""
							a.flash(locales.SettingsAccountUpdated(), "success")
						})
					})
				})
			}
		})
	}

	groupTitle(c, p, locales.SettingsDataCache())
	ui.Divider(c)
	ui.Row(c).Padding(4, 0, 12, 0).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, locales.SettingsMailCache()).FontSize(13).Bold().TextColor(p.Text)
			ui.Text(c, locales.SettingsClearHint()).
				FontSize(11.5).TextColor(p.Faint).LineHeight(1.5)
		})
		if a.clearAsk {
			ui.Button(c, locales.SettingsClearConfirm()).OnClick(func() {
				a.clearAsk = false
				go func() {
					a.update(func() { a.clearCache() })
				}()
			})
			ui.Button(c, locales.MiscCancel()).OnClick(func() { a.clearAsk = false })
		} else {
			ui.Button(c, locales.SettingsClear()).OnClick(func() { a.clearAsk = true })
		}
	})

	ui.Row(c).Gap(8).Margin(4, 0, 0, 0).Children(func() {
		add := a.ctaBase(c, p, locales.SettingsAddAccount())
		add.Children(func() {
			ui.Icon(c, iconPlus).FontSize(14).TextColor(p.CTAText)
			ui.Text(c, locales.SettingsAddAccount()).FontSize(13).Bold().TextColor(p.CTAText)
		})
		add.OnClick(func() {
			a.wizard = newWizard()
		})
	})
}

func (a *App) confirmRemove(acc *AccountVM) {
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  a.win,
			Type:    mygo.MessageWarning,
			Message: locales.SettingsRemovePrefix() + acc.Name + locales.MiscQuestion(),
			Detail:  locales.SettingsRemoveWarn(),
			Buttons: []string{locales.SettingsRemove(), locales.MiscCancel()},
		})
		if err != nil || res.Button != 0 {
			return
		}
		a.update(func() { a.removeAccount(acc.ID) })
	}()
}

func (a *App) settingsAppearance(c *ui.Context, p palette) {
	groupTitle(c, p, locales.SettingsTheme())
	themes := []string{locales.SettingsFollowSystem(), locales.SettingsLight(), locales.SettingsDark()}
	idx := map[string]int{"auto": 0, "light": 1, "dark": 2}[a.prefs.Theme]
	seg := ui.Segmented(c, &idx, themes...).Label(locales.SettingsTheme()).AlignSelf(ui.Start)
	if seg.Changed() {
		a.prefs.Theme = map[int]string{0: "auto", 1: "light", 2: "dark"}[idx]
		a.savePrefs()
		applyThemePref(a.prefs.Theme)
		a.loadBody()
	}

	groupTitle(c, p, locales.SettingsLanguage())
	langs := []string{locales.SettingsFollowSystem(), "English", "简体中文"}
	langIdx := map[string]int{"auto": 0, "en": 1, "zh-Hans": 2}[a.prefs.Language]
	langSeg := ui.Segmented(c, &langIdx, langs...).Label(locales.SettingsLanguage()).AlignSelf(ui.Start)
	if langSeg.Changed() {
		a.prefs.Language = map[int]string{0: "auto", 1: "en", 2: "zh-Hans"}[langIdx]
		a.savePrefs()
		mygo.App.SetLocale(map[bool]string{true: "", false: a.prefs.Language}[a.prefs.Language == "auto"])
	}

	groupTitle(c, p, locales.SettingsWindowBackground())
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

	groupTitle(c, p, locales.SettingsDensityList())
	densities := []string{locales.SettingsDensityRoomy(), locales.SettingsDensityCompact()}
	d := 0
	if a.prefs.Density == "compact" {
		d = 1
	}
	if ui.Segmented(c, &d, densities...).Label(locales.SettingsDensity()).AlignSelf(ui.Start).Changed() {
		a.prefs.Density = map[int]string{0: "cozy", 1: "compact"}[d]
		a.savePrefs()
	}

	groupTitle(c, p, locales.SettingsReading())
	a.prefToggle(c, p, locales.SettingsMarkReadOnOpen(), &a.prefs.MarkRead)
	a.prefToggle(c, p, locales.SettingsNotifications(), &a.prefs.Notifications)
	a.prefToggle(c, p, locales.SettingsAutoLoadImages(), &a.prefs.AutoImages)
	a.prefToggle(c, p, locales.SettingsQuoteReply(), &a.prefs.QuoteReply)
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
	groupTitle(c, p, locales.SettingsTabShortcuts())
	rows := [][2]string{
		{"J / K", locales.SettingsScNav()},
		{"Enter", locales.SettingsScOpen()},
		{"N", locales.SettingsScCompose()},
		{"R / A / F", locales.SettingsScReply()},
		{"S", locales.MiscStar()},
		{"E", locales.MiscArchive()},
		{"⌫", locales.MiscDelete()},
		{modKey() + "B", locales.SettingsScSidebar()},
		{locales.SettingsScSearchKey(modKey()), locales.TitlebarSearch()},
		{"Esc", locales.SettingsScClose()},
		{"?", locales.SettingsOpenShortcuts()},
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
		ui.Text(c, locales.SettingsAboutTagline(version.Version)).FontSize(12).TextColor(p.Faint)
		ui.Text(c, locales.SettingsAboutBody()).
			FontSize(12.5).TextColor(p.Muted).TextAlign(ui.Center).MaxWidth(420).LineHeight(1.7).Margin(8, 0, 0)
	})
}

func statusLabel(status string) string {
	switch status {
	case "online":
		return locales.SettingsOnline()
	case "connecting":
		return locales.SettingsConnecting()
	case "offline":
		return locales.SettingsOffline()
	case "error":
		return locales.WizardConnectFailed()
	default:
		return locales.SettingsNotConnected()
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
	{"plain", locales.SettingsWashPlain()},
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
