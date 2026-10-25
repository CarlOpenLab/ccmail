package ccapp

import (
	"sort"
	"strconv"
	"strings"

	"ccmail/locales"
	"github.com/egoist/mygo/ui"
)

var folderIcons = map[string]*ui.SVG{
	"inbox": iconInbox, "sent": iconSend, "drafts": iconDraft, "trash": iconTrash,
	"junk": iconJunk, "archive": iconArchive,
}

// sidebar is the source list: smart views, then every account's mailboxes.
func (a *App) sidebar(c *ui.Context, p palette, compact bool) {
	ui.Column(c).FillHeight().Gap(4).Children(func() {
		if compact {
			a.sidebarRail(c, p)
			return
		}
		ui.Column(c).Width(a.sidebarW).FillHeight().Gap(4).Children(func() {
			side := ui.Sidebar(c, &a.sidebarSel, func() {
				ui.SidebarSection(c, locales.SidebarSmartViews(), nil, func() {
					item := ui.SidebarItem(c, "unified", iconInbox, locales.MiscUnifiedInbox())
					if a.unreadTotal > 0 {
						item.Children(func() {
							bg, fg := p.badgeColors()
							ui.Badge(c, strconv.Itoa(a.unreadTotal)).Background(bg).TextColor(fg)
						})
					}
					ui.SidebarItem(c, "flagged", iconStar, locales.MiscStarred())
				})

				if len(a.accounts) == 0 {
					return
				}
				ui.SidebarSection(c, locales.SidebarMailboxes(), nil, func() {
					for _, acc := range a.accounts {
						open := a.sectionOpen(acc.ID)
						ui.SidebarSection(c, acc.Name, open, func() {
							for _, f := range visibleFolders(acc.Folders) {
								icon := folderIcons[f.Kind]
								if icon == nil {
									icon = iconFolder
								}
								item := ui.SidebarItem(c, "folder/"+acc.ID+"/"+f.Path, icon, f.Name)
								if f.Unread > 0 {
									item.Children(func() {
										bg, fg := p.badgeColors()
										ui.Badge(c, strconv.Itoa(f.Unread)).Background(bg).TextColor(fg)
									})
								}
							}
						})
					}
				})
			}).Grow(1).Label(locales.SidebarAriaLabel()).Background(ui.Transparent)

			if side.Changed() {
				a.sidebarChanged()
			}

			foot := ui.ButtonBase(c).Height(32).Radius(8).FillWidth().Center().Gap(6).
				Label(locales.SidebarAddAccount()).Background(p.Card).Border(1, p.Border)
			if foot.Hovered() {
				foot.Background(p.BtnHover)
			}
			foot.Children(func() {
				ui.Icon(c, iconPlus).FontSize(14).TextColor(p.Muted)
				ui.Text(c, locales.SidebarAddAccount()).FontSize(12.5).TextColor(p.Text)
			})
			foot.OnClick(func() { a.openWizard() })

			ui.Text(c, locales.SidebarCaption(appName)).FontSize(11).
				TextColor(p.Faint).TextAlign(ui.Center)
		})
	})
}

// sidebarRail is the collapsed sidebar. Items keep the expanded list's
// rhythm (30px rows, same icon column) so collapsing moves nothing sideways
// and only the labels go away.
func (a *App) sidebarRail(c *ui.Context, p palette) {
	const size = 30
	item := func(id string, icon *ui.SVG, label string) {
		active := a.sidebarSel == id
		b := ui.ButtonBase(c).Size(size, size).Radius(7).Center().Label(label).Tooltip(label)
		switch {
		case active:
			b.Background(p.ChipDeep)
		case b.Hovered():
			b.Background(p.Hover)
		}
		color := p.Muted
		if active {
			color = p.Text
		}
		b.Children(func() { ui.Icon(c, icon).FontSize(17).TextColor(color) })
		b.OnClick(func() {
			a.sidebarSel = id
			a.sidebarChanged()
		})
	}
	rule := func() {
		ui.Box(c).Width(20).Height(1).Background(p.Border).Margin(6, 0)
	}

	ui.Column(c).Width(52).FillHeight().AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
				// the expanded list starts under its section title: keep the
				// first icon on the same line when the labels go away
				ui.Box(c).Height(32)
				item("unified", iconInbox, locales.MiscUnifiedInbox())
				item("flagged", iconStar, locales.MiscStarred())
				for _, acc := range a.accounts {
					if len(acc.Folders) > 0 {
						rule()
					}
					for _, f := range visibleFolders(acc.Folders) {
						icon := folderIcons[f.Kind]
						if icon == nil {
							icon = iconFolder
						}
						item("folder/"+acc.ID+"/"+f.Path, icon, acc.Name+" · "+f.Name)
					}
				}
			})
		})

		add := ui.ButtonBase(c).Size(size, size).Radius(7).Center().
			Label(locales.SidebarAddAccount()).Tooltip(locales.SidebarAddAccount())
		if add.Hovered() {
			add.Background(p.Hover)
		}
		add.Children(func() { ui.Icon(c, iconPlus).FontSize(16).TextColor(p.Muted) })
		add.OnClick(func() { a.openWizard() })
	})
}

// sectionOpen keeps one account's fold state across frames.
func (a *App) sectionOpen(id string) *bool {
	if a.openSections == nil {
		a.openSections = map[string]*bool{}
	}
	if v, ok := a.openSections[id]; ok {
		return v
	}
	v := true
	a.openSections[id] = &v
	return &v
}

// visibleFolders lists a mail account's mailboxes in sidebar order.
func visibleFolders(all []Folder) []Folder {
	order := map[string]int{"inbox": 0, "drafts": 1, "sent": 2, "archive": 3, "junk": 4, "trash": 5}
	out := make([]Folder, 0, len(all))
	for _, f := range all {
		if f.Kind == "all" || f.Kind == "important" {
			continue
		}
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, iok := order[out[i].Kind]
		oj, jok := order[out[j].Kind]
		if iok != jok {
			return iok
		}
		if iok && oi != oj {
			return oi < oj
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// sidebarChanged turns the chosen source-list item into a view.
func (a *App) sidebarChanged() {
	a.sidebarOpen = false
	id := a.sidebarSel
	switch {
	case id == "unified":
		a.navigate(ViewSel{Kind: "unified", Name: locales.MiscUnifiedInbox()})
	case id == "flagged":
		a.navigate(ViewSel{Kind: "flagged", Name: locales.MiscStarred()})
	case strings.HasPrefix(id, "folder/"):
		parts := strings.SplitN(id, "/", 3)
		if len(parts) != 3 {
			return
		}
		accountID, path := parts[1], parts[2]
		name := path
		for _, acc := range a.accounts {
			if acc.ID != accountID {
				continue
			}
			for _, f := range acc.Folders {
				if f.Path == path {
					name = f.Name
				}
			}
		}
		a.navigate(ViewSel{Kind: "folder", AccountID: accountID, Path: path, Name: name})
	}
}
