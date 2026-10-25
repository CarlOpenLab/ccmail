package ccapp

import (
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

var folderIcons = map[string]*ui.SVG{
	"inbox": iconInbox, "sent": iconSend, "drafts": iconDraft, "trash": iconTrash,
	"junk": iconJunk, "archive": iconArchive,
}

// sidebar is the source list: smart views, then every account's mailboxes.
func (a *App) sidebar(c *ui.Context, p palette) {
	ui.Column(c).Width(a.sidebarW).Shrink(0).Gap(4).Children(func() {
		side := ui.Sidebar(c, &a.sidebarSel, func() {
			ui.SidebarSection(c, "智能视图", nil, func() {
				item := ui.SidebarItem(c, "unified", iconInbox, "全部收件箱")
				if a.unreadTotal > 0 {
					item.Children(func() {
						bg, fg := p.badgeColors()
						ui.Badge(c, strconv.Itoa(a.unreadTotal)).Background(bg).TextColor(fg)
					})
				}
				ui.SidebarItem(c, "flagged", iconStar, "星标邮件")
			})

			if len(a.accounts) == 0 {
				return
			}
			ui.SidebarSection(c, "邮箱", nil, func() {
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
		}).Grow(1).Label("侧栏").Background(ui.Transparent)

		if side.Changed() {
			a.sidebarChanged()
		}

		foot := ui.ButtonBase(c).Height(32).Radius(8).FillWidth().Center().Gap(6).Label("添加邮箱账号")
		if foot.Hovered() {
			foot.Background(p.Chip)
		}
		foot.Background(p.Chip.Alpha(0.5))
		foot.Children(func() {
			ui.Icon(c, iconPlus).FontSize(14).TextColor(p.Muted)
			ui.Text(c, "添加邮箱账号").FontSize(12.5).TextColor(p.Muted)
		})
		foot.OnClick(func() { a.openWizard() })

		ui.Textf(c, "%s · IMAP / SMTP 多账号", appName).FontSize(11).
			TextColor(p.Faint).TextAlign(ui.Center)
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
		a.navigate(ViewSel{Kind: "unified", Name: "全部收件箱"})
	case id == "flagged":
		a.navigate(ViewSel{Kind: "flagged", Name: "星标邮件"})
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
