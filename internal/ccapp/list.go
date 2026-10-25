package ccapp

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"ccmail/internal/mailx"
	"ccmail/internal/store"
	"ccmail/locales"
)

// listPane is the message list: its heading, filters, and the rows.
func (a *App) listPane(c *ui.Context, p palette) {
	ui.Column(c).Fill().Clip().BorderWidth(0, 1, 0, 0).BorderColor(p.Border).Children(func() {
		ui.Column(c).Padding(14, 14, 8, 14).Gap(10).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Gap(3).Children(func() {
					ui.Text(c, a.viewTitle()).FontSize(17).Bold().SingleLine().TextColor(p.Text)
					ui.Text(c, a.viewSubtitle()).FontSize(12).TextColor(p.Faint).SingleLine()
				})
				more := ui.ButtonBase(c).Label(locales.ListMoreActions()).Tooltip(locales.ListMoreActions()).Size(28, 28).Radius(8).Center()
				more.Children(func() { ui.Icon(c, iconMore).FontSize(17).TextColor(p.Muted) })
				if more.Hovered() {
					more.Background(p.HoverStrong)
				}
				more.Menu(func(m *ui.Menu) {
					if m.Item(locales.TitlebarRefresh()).Chosen() {
						a.reloadList(true)
					}
					if m.Item(locales.ListMenuMarkAllRead()).Chosen() {
						a.markAllRead()
					}
					if m.Item(locales.ListMenuBackToUnified()).Chosen() {
						a.navigate(ViewSel{Kind: "unified", Name: locales.MiscUnifiedInbox()})
					}
				})
			})

			seg := ui.Segmented(c, &a.filter, locales.ListFilterAll(), locales.ListFilterUnread(), locales.ListFilterStarred(), locales.ListFilterAttachments()).Label(locales.ListFilter()).AlignSelf(ui.Start)
			if seg.Changed() {
				a.reloadList(true)
			}
		})

		a.list.Key = a.rowKey
		a.list.Label = a.rowLabel
		a.list.Header = func(i int) bool { return i >= 0 && i < len(a.rows) && a.rows[i].isHeader() }
		a.list.Selected = &a.listSel

		// The list paints its chosen row in the theme's accent. Give it the
		// same color as our selection fill for the duration of the build so
		// no hairline of accent shows around the row, then restore.
		base := *c.Theme()
		listTheme := base
		listTheme.Accent = p.SelectedRow
		listTheme.AccentText = p.SelectedText
		c.SetTheme(&listTheme)
		lst := ui.List(c, &a.list, len(a.rows), func(i int) {
			a.listRow(c, p, i)
		}).Grow(1).Padding(4, 8).Gap(2).Label(locales.ListAriaLabel())
		c.SetTheme(&base)

		lst.Children(func() {
			if a.loading && len(a.rows) == 0 {
				ui.Column(c).Fill().Center().Gap(10).Children(func() {
					ui.Spinner(c)
					ui.Text(c, locales.ListSyncing()).FontSize(12.5).TextColor(p.Faint)
				})
				return
			}
			if a.loadErr != "" {
				ui.Column(c).Fill().Center().Gap(8).Padding(30).Children(func() {
					ui.Icon(c, iconJunk).FontSize(28).TextColor(p.Danger)
					ui.Text(c, locales.ListLoadFailed()).FontSize(14).Bold().TextColor(p.Text)
					ui.Text(c, a.loadErr).FontSize(12).TextColor(p.Muted).TextAlign(ui.Center)
					ui.Button(c, locales.MiscRetry()).OnClick(func() { a.reloadList(true) })
				})
				return
			}
			if len(a.rows) == 0 {
				title, hint := locales.ListEmptyTitle(), locales.ListEmptyHint()
				if a.query != "" || a.filter != 0 {
					title, hint = locales.ListNoMatchTitle(), locales.ListNoMatchHint()
				}
				ui.Column(c).Fill().Center().Gap(6).Padding(30).Children(func() {
					ui.Box(c).Size(64, 64).Radius(20).Center().Background(p.Chip).Children(func() {
						ui.Icon(c, iconInbox).FontSize(28).TextColor(p.Faint)
					})
					ui.Text(c, title).FontSize(15).Bold().TextColor(p.Text)
					ui.Text(c, hint).FontSize(12.5).TextColor(p.Faint).TextAlign(ui.Center).MaxWidth(250)
					if a.query != "" || a.filter != 0 {
						ui.Button(c, locales.ListClearFilters()).OnClick(func() {
							a.query = ""
							a.filter = 0
							a.reloadList(true)
						})
					}
				})
			}
		})

		for len(a.pending) > 0 {
			fn := a.pending[0]
			a.pending = a.pending[1:]
			fn()
		}

		if a.pendingScroll >= 0 {
			a.list.ScrollIntoView(a.pendingScroll)
			a.pendingScroll = -1
		}

		if lst.Changed() || lst.Submitted() {
			a.openRow(a.listSel)
		}

		if a.more {
			ui.Row(c).Padding(6, 10, 10, 10).Justify(ui.Center).Children(func() {
				ui.Button(c, locales.ListLoadMore(len(a.msgs), a.total)).OnClick(func() {
					a.page++
					a.reloadList(false)
				})
			})
		}
	})
}

func (a *App) viewTitle() string {
	if a.view.Name != "" {
		return a.view.Name
	}
	return locales.MiscInbox()
}

func (a *App) viewSubtitle() string {
	parts := []string{locales.ListMailCount(a.total)}
	if a.view.Kind == "folder" {
		if acc := a.accountVM(a.view.AccountID); acc != nil {
			parts = append(parts, acc.Name)
		}
	}
	if a.query != "" {
		parts = append(parts, locales.ListSearching(a.query))
	}
	return joinParts(parts)
}

func joinParts(parts []string) string {
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += " · "
		}
		out += s
	}
	return out
}

// listRow builds one heading or one message.
func (a *App) listRow(c *ui.Context, p palette, i int) {
	if i < 0 || i >= len(a.rows) {
		return
	}
	r := a.rows[i]
	if r.isHeader() {
		ui.Text(c, r.header).FontSize(11).Bold().LetterSpacing(0.5).TextColor(p.Faint).
			Padding(14, 10).Background(p.Card)
		return
	}

	m := r.msg
	env := m.Env
	unread := !env.Flags.Seen
	selected := i == a.listSel

	// The native list paints its chosen row in the accent color. The row it
	// gets is only a shell; a filling child paints the design's quieter
	// selection over it, so the accent stays for carets and switches.
	row := ui.Row(c).Radius(10)

	row.ContextMenu(func(menu *ui.Menu) {
		starred := env.Flags.Flagged
		label := locales.MiscStar()
		if starred {
			label = locales.MiscUnstar()
		}
		if menu.Item(label).Chosen() {
			a.deferAction(func() { a.toggleStar(m) })
		}
		readLabel := locales.MiscMarkUnread()
		if !env.Flags.Seen {
			readLabel = locales.MiscMarkRead()
		}
		if menu.Item(readLabel).Chosen() {
			a.deferAction(func() { a.markRead(m, env.Flags.Seen) })
		}
		menu.Separator()
		if menu.Item(locales.MiscArchive()).Chosen() {
			a.deferAction(func() { a.archiveMessage(m) })
		}
		if menu.Item(locales.MiscDelete()).Chosen() {
			a.deferAction(func() { a.deleteMessage(m) })
		}
		if acc := a.accountVM(m.AccountID); acc != nil && len(acc.Folders) > 0 {
			menu.Submenu(locales.ListCtxMoveTo(), func(sub *ui.Menu) {
				for _, f := range visibleFolders(acc.Folders) {
					if f.Path == m.Path {
						continue
					}
					f := f
					if sub.Item(f.Name).Chosen() {
						a.deferAction(func() { a.moveMessage(m, f.Path) })
					}
				}
			})
		}
	})

	onSel := func(normal, chosen ui.Color) ui.Color {
		if selected {
			return chosen
		}
		return normal
	}
	secondary := func(alpha float32) ui.Color {
		if selected {
			return p.SelectedMuted.Alpha(alpha)
		}
		return p.Faint
	}

	row.Children(func() {
		fill := ui.Row(c).FillWidth().Padding(9, 10).Gap(10).Radius(10).AlignItems(ui.Start)
		// The list paints hover itself and draws the keyboard focus ring;
		// the fill only owns the selection, so neither is doubled.
		if selected {
			fill.Background(p.SelectedRow)
		}
		fill.Children(func() {
			ui.Avatar(c, displayName(env.From), nil).FontSize(16)

			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					if unread && !selected {
						ui.Box(c).Size(5, 5).Radius(3).Background(p.Accent)
					}
					name := ui.Text(c, displayName(env.From)).FontSize(13.5).SingleLine().Grow(1)
					if unread {
						name.Bold().TextColor(onSel(p.Text, p.SelectedText))
					} else {
						name.TextColor(onSel(p.Muted, p.SelectedMuted))
					}
					ui.Text(c, fmtListDate(env.Date)).FontSize(11.5).TextColor(secondary(1))
				})

				subj := ui.Text(c, env.Subject).FontSize(13.5).SingleLine()
				if unread {
					subj.Bold().TextColor(onSel(p.Text, p.SelectedText))
				} else {
					subj.TextColor(onSel(p.Muted, p.SelectedMuted))
				}

				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					snippet := env.Snippet
					if a.prefs.Density == "compact" {
						snippet = ""
					}
					if snippet != "" {
						ui.Text(c, snippet).FontSize(12).TextColor(secondary(0.85)).SingleLine().Grow(1)
					} else {
						ui.Spacer(c)
					}
					if env.HasAttachments {
						ui.Icon(c, iconClip).FontSize(13).TextColor(secondary(1))
					}
					if a.view.Kind != "folder" && m.AccountName != "" {
						chip := ui.Row(c).Padding(1, 7).Radius(8).Gap(4).AlignItems(ui.Center)
						if selected {
							chip.Background(p.ChipDeep)
						} else {
							chip.Background(p.Chip)
						}
						chip.Children(func() {
							ui.Box(c).Size(5, 5).Radius(3).Background(ui.Hex(m.Color))
							label := ui.Text(c, m.AccountName).FontSize(11).SingleLine().MaxWidth(80)
							if selected {
								label.TextColor(p.SelectedText)
							} else {
								label.TextColor(p.Muted)
							}
						})
					}
					star := ui.ButtonBase(c).Size(22, 22).Radius(6).Center().
						Label(map[bool]string{true: locales.MiscUnstar(), false: locales.MiscStar()}[env.Flags.Flagged])
					if env.Flags.Flagged {
						star.Children(func() { ui.Icon(c, iconStarFill).FontSize(14).TextColor(p.Star) })
					} else {
						alpha := float32(0.42)
						if row.Hovered() || selected {
							alpha = 0.95
						}
						star.Children(func() { ui.Icon(c, iconStar).FontSize(14).TextColor(secondary(alpha)) })
					}
					star.OnClick(func() { a.toggleStar(m) })
				})
			})
		})
	})
}

// openRow opens the message at a row, if the row holds one.
func (a *App) openRow(i int) {
	if m := a.rowOf(i); m != nil {
		a.openMessage(m)
	}
}

func (a *App) markAllRead() {
	byMailbox := map[string][]uint32{}
	for _, m := range a.msgs {
		if m.Env.Flags.Seen {
			continue
		}
		m.Env.Flags.Seen = true
		key := m.AccountID + "\x00" + m.Path
		byMailbox[key] = append(byMailbox[key], m.Env.UID)
	}
	a.syncListSelection()
	for key, uids := range byMailbox {
		parts := strings.SplitN(key, "\x00", 2)
		go func(accountID, path string, uids []uint32) {
			acct := store.Get(accountID)
			if acct == nil {
				return
			}
			seen := true
			_ = mailx.SetFlags(acct, path, uids, mailx.FlagPatch{Seen: &seen})
		}(parts[0], parts[1], uids)
	}
}
