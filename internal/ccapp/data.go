package ccapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"ccmail/internal/hub"
	"ccmail/internal/mailx"
	"ccmail/internal/prefs"
	"ccmail/internal/providers"
	"ccmail/internal/smtpx"
	"ccmail/internal/store"
	"ccmail/internal/watchers"
)

// loadPrefs reads the stored preferences with the app's defaults.
func loadPrefs() Prefs {
	all := prefs.All()
	get := func(k, def string) string {
		if v, ok := all[k]; ok && v != "" {
			return v
		}
		return def
	}
	return Prefs{
		Theme:         get("theme", "auto"),
		Wash:          get("wash", "plain"),
		Density:       get("density", "cozy"),
		MarkRead:      get("markRead", "1") != "0",
		Notifications: get("notifications", "on") != "off",
		AutoImages:    get("autoImages", "0") == "1",
		QuoteReply:    get("quoteReply", "1") != "0",
		SidebarShut:   get("sidebarShut", "0") == "1",
	}
}

// defaultPrefs is the app's starting configuration, which the tests use so a
// developer's own data/prefs.json never changes a screenshot.
func defaultPrefs() Prefs {
	return Prefs{
		Theme:         "auto",
		Wash:          "plain",
		Density:       "cozy",
		MarkRead:      true,
		Notifications: true,
		QuoteReply:    true,
	}
}

func (a *App) savePrefs() {
	on := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	prefs.SetMany(map[string]string{
		"theme":         a.prefs.Theme,
		"wash":          a.prefs.Wash,
		"density":       a.prefs.Density,
		"markRead":      on(a.prefs.MarkRead),
		"notifications": map[bool]string{true: "on", false: "off"}[a.prefs.Notifications],
		"autoImages":    on(a.prefs.AutoImages),
		"quoteReply":    on(a.prefs.QuoteReply),
		"sidebarShut":   on(a.prefs.SidebarShut),
	})
}

// applyThemePref follows the user's appearance setting.
func applyThemePref(theme string) {
	switch theme {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

// loadAccounts builds the sidebar's account list from the store.
func loadAccounts() []*AccountVM {
	all := store.All()
	out := make([]*AccountVM, 0, len(all))
	for _, s := range all {
		out = append(out, &AccountVM{
			ID:       s.ID,
			Name:     s.Name,
			Email:    s.Email,
			Color:    s.Color,
			Provider: s.Provider,
		})
	}
	return out
}

// refreshAccounts reloads accounts and their connection status.
func (a *App) refreshAccounts() {
	accounts := loadAccounts()
	statuses := watchers.Statuses()
	a.update(func() {
		// keep the folders already loaded, they refresh on their own
		boxes := map[string][]Folder{}
		for _, acc := range a.accounts {
			boxes[acc.ID] = acc.Folders
		}
		for _, acc := range accounts {
			acc.Folders = boxes[acc.ID]
			if st, ok := statuses[acc.ID]; ok {
				acc.Status, acc.Detail, acc.Unread = st.Status, st.Detail, st.Unread
			}
		}
		a.accounts = accounts
		a.statuses = statuses
		a.recomputeUnread()
	})
	a.refreshFolders()
}

// refreshFolders lists every account's mailboxes in the background.
func (a *App) refreshFolders() {
	for _, vm := range a.accounts {
		id := vm.ID
		go func() {
			acct := store.Get(id)
			if acct == nil {
				return
			}
			infos, err := mailx.ListMailboxes(acct)
			if err != nil {
				return
			}
			folders := make([]Folder, 0, len(infos))
			for _, info := range infos {
				f := Folder{AccountID: id, Path: info.Path, Name: info.Name, Kind: info.Kind}
				if info.Unread != nil {
					f.Unread = *info.Unread
				}
				if info.Messages != nil {
					f.Total = *info.Messages
				}
				folders = append(folders, f)
			}
			a.update(func() {
				for _, acc := range a.accounts {
					if acc.ID == id {
						acc.Folders = folders
					}
				}
				a.recomputeUnread()
			})
		}()
	}
}

func (a *App) recomputeUnread() {
	total := 0
	for _, acc := range a.accounts {
		unread := acc.Unread
		if unread == 0 {
			for _, f := range acc.Folders {
				if f.Kind == "inbox" {
					unread = f.Unread
				}
			}
		}
		total += unread
	}
	a.unreadTotal = total
}

func (a *App) accountVM(id string) *AccountVM {
	for _, acc := range a.accounts {
		if acc.ID == id {
			return acc
		}
	}
	return nil
}

// ---------------------------------------------------------------- the list

func filterName(i int) string {
	return [...]string{"", "unread", "flagged", "attachments"}[i]
}

// reloadList loads the current view in the background. reset starts at page
// one; otherwise the next page is appended.
func (a *App) reloadList(reset bool) {
	if reset {
		a.page = 1
		if a.query != "" || a.filter != 0 {
			// a fresh search keeps the selected message only if it re-appears
		}
	}
	view, query, filter, page := a.view, a.query, filterName(a.filter), a.page
	a.loading = true
	a.loadErr = ""

	go func() {
		rows, total, err := fetchList(view, query, filter, page)
		a.update(func() {
			if view != a.view || query != a.query || filterName(a.filter) != filter || page != a.page {
				return // a newer request won
			}
			a.loading = false
			if err != nil {
				a.loadErr = err.Error()
				if reset {
					a.msgs = nil
					a.total = 0
				}
				return
			}
			if reset {
				a.msgs = rows
			} else {
				a.msgs = append(a.msgs, rows...)
			}
			a.total = total
			a.more = len(a.msgs) < total
			a.rebuildRows()
			a.syncListSelection()
		})
	}()
}

// syncListSelection points the native list at the selected message.
func (a *App) syncListSelection() {
	a.listSel = -1
	if a.sel != nil {
		key := a.sel.Key()
		for i, r := range a.rows {
			if r.msg != nil && r.msg.Key() == key {
				a.listSel = i
				break
			}
		}
	}
	if a.listSel >= 0 {
		a.listKey = a.sel.Key()
	}
	a.list.Selected = &a.listSel
}

func fetchList(view ViewSel, query, filter string, page int) ([]*MsgRef, int, error) {
	if view.Kind == "folder" {
		acct := store.Get(view.AccountID)
		if acct == nil {
			return nil, 0, fmt.Errorf("账号不存在")
		}
		res, err := mailx.ListMessages(acct, view.Path, mailx.Query{Page: page, Search: query, Filter: filter})
		if err != nil {
			return nil, 0, err
		}
		acc := accountVMFromStore(acct)
		out := make([]*MsgRef, 0, len(res.Messages))
		for _, env := range res.Messages {
			out = append(out, &MsgRef{AccountID: acc.ID, AccountName: acc.Name, Color: acc.Color, Path: view.Path, Env: env})
		}
		return out, res.Total, nil
	}

	// unified / starred: aggregate cached INBOX envelopes
	var out []*MsgRef
	for _, acct := range store.All() {
		rows := mailx.CachedEnvelopes(acct.ID, "INBOX")
		if rows == nil {
			watchers.KickInboxSync(acct)
			continue
		}
		vm := accountVMFromStore(acct)
		for _, env := range rows {
			if view.Kind == "flagged" && !env.Flags.Flagged {
				continue
			}
			if !envelopeMatches(env, query, filter) {
				continue
			}
			out = append(out, &MsgRef{AccountID: vm.ID, AccountName: vm.Name, Color: vm.Color, Path: "INBOX", Env: env})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ti, oki := parseDate(out[i].Env.Date)
		tj, okj := parseDate(out[j].Env.Date)
		if oki && okj {
			return ti.After(tj)
		}
		return out[i].Env.Date > out[j].Env.Date
	})
	if len(out) > 150 {
		out = out[:150]
	}
	return out, len(out), nil
}

func accountVMFromStore(s *store.Account) *AccountVM {
	return &AccountVM{ID: s.ID, Name: s.Name, Email: s.Email, Color: s.Color, Provider: s.Provider}
}

func envelopeMatches(env *mailx.Envelope, query, filter string) bool {
	switch filter {
	case "unread":
		if env.Flags.Seen {
			return false
		}
	case "flagged":
		if !env.Flags.Flagged {
			return false
		}
	case "attachments":
		if !env.HasAttachments {
			return false
		}
	}
	if query != "" {
		q := strings.ToLower(query)
		if !strings.Contains(strings.ToLower(env.Subject), q) &&
			!strings.Contains(strings.ToLower(env.From.Name), q) &&
			!strings.Contains(strings.ToLower(env.From.Email), q) &&
			!strings.Contains(strings.ToLower(env.Snippet), q) {
			return false
		}
	}
	return true
}

// navigate shows a view and loads its first page.
func (a *App) navigate(v ViewSel) {
	a.view = v
	a.sel = nil
	a.msg = nil
	a.msgErr = ""
	a.listSel = -1
	a.loading = true
	a.msgs = nil
	a.total = 0
	a.more = false
	a.setSidebarFor(v)
	a.reloadList(true)
	a.loadBody()
}

func (a *App) setSidebarFor(v ViewSel) {
	switch {
	case v.Kind == "folder":
		a.sidebarSel = "folder/" + v.AccountID + "/" + v.Path
	default:
		a.sidebarSel = v.Kind
	}
}

// openMessage shows a message in the reader.
func (a *App) openMessage(ref *MsgRef) {
	a.sel = ref
	a.msg = nil
	a.msgErr = ""
	a.msgLoading = true
	a.allowImages = a.prefs.AutoImages
	a.loadBody()

	accountID, path, uid := ref.AccountID, ref.Path, ref.Env.UID
	if a.demo && store.Get(accountID) == nil {
		// sample mail never touches IMAP; every row opens the demo body
		a.msgLoading = false
		a.msg = demoMessage(ref, time.Now())
		a.sel.Env.Flags.Seen = true
		a.loadBody()
		return
	}
	go func() {
		acct := store.Get(accountID)
		if acct == nil {
			a.update(func() {
				a.msgLoading = false
				a.msgErr = "账号不存在"
				a.loadBody()
			})
			return
		}
		full, err := mailx.GetMessage(acct, path, uid, a.prefs.MarkRead)
		a.update(func() {
			if a.sel == nil || a.sel.Key() != ref.Key() {
				return
			}
			a.msgLoading = false
			if err != nil {
				a.msgErr = err.Error()
			} else {
				a.msg = full
				ref.Env.Flags.Seen = true
			}
			a.loadBody()
		})
	}()
}

func (a *App) markSeen(ref *MsgRef) {
	ref.Env.Flags.Seen = true
	a.syncListSelection()
}

// loadBody hands the sanitized body to the reader, native text or the web
// view made for HTML mail.
func (a *App) loadBody() {
	if a.web == nil {
		return
	}
	if a.sel == nil || a.msg == nil {
		a.blockedImages = 0
		a.web.Page().LoadHTML(emptyReaderHTML(), "")
		return
	}
	if a.msg.HTML == "" {
		a.blockedImages = 0
		a.web.Page().LoadHTML(emptyReaderHTML(), "")
		return
	}
	accent := paletteFor(isDarkNow()).Accent
	doc, blocked, err := renderMailHTML(a.msg, a.allowImages, accent)
	a.blockedImages = blocked
	if err != nil {
		a.web.Page().LoadHTML(emptyReaderHTML(), "")
		return
	}
	a.web.Page().LoadHTML(doc, "")
}

func (a *App) reloadImages() {
	a.allowImages = true
	a.loadBody()
}

// ------------------------------------------------------------- message ops

func (a *App) toggleStar(ref *MsgRef) {
	next := !ref.Env.Flags.Flagged
	ref.Env.Flags.Flagged = next
	if a.sel != nil && a.sel.Key() == ref.Key() && a.msg != nil {
		// the reader reads its flags from the envelope
	}
	accountID, path, uid := ref.AccountID, ref.Path, ref.Env.UID
	go func() {
		if acct := store.Get(accountID); acct != nil {
			_ = mailx.SetFlags(acct, path, []uint32{uid}, mailx.FlagPatch{Flagged: &next})
		}
		a.update(func() {
			hub.Broadcast("flags:changed", map[string]any{"accountId": accountID, "path": path, "uid": uid})
			if a.view.Kind == "flagged" && !next {
				a.reloadList(true)
			}
		})
	}()
}

func (a *App) markRead(ref *MsgRef, seen bool) {
	ref.Env.Flags.Seen = seen
	accountID, path, uid := ref.AccountID, ref.Path, ref.Env.UID
	go func() {
		if acct := store.Get(accountID); acct != nil {
			_ = mailx.SetFlags(acct, path, []uint32{uid}, mailx.FlagPatch{Seen: &seen})
		}
		a.update(func() {
			hub.Broadcast("flags:changed", map[string]any{"accountId": accountID, "path": path, "uid": uid})
		})
	}()
}

func (a *App) archiveMessage(ref *MsgRef) {
	accountID, path, uid := ref.AccountID, ref.Path, ref.Env.UID
	a.removeRow(ref)
	go func() {
		acct := store.Get(accountID)
		if acct == nil {
			return
		}
		if err := mailx.ArchiveMessages(acct, path, []uint32{uid}); err != nil {
			a.update(func() { a.flash("归档失败："+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash("已归档", "success") })
	}()
}

func (a *App) deleteMessage(ref *MsgRef) {
	accountID, path, uid := ref.AccountID, ref.Path, ref.Env.UID
	a.removeRow(ref)
	go func() {
		acct := store.Get(accountID)
		if acct == nil {
			return
		}
		if err := mailx.DeleteMessages(acct, path, []uint32{uid}); err != nil {
			a.update(func() { a.flash("删除失败："+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash("已移到废纸篓", "success") })
	}()
}

// moveMessage files a message into another mailbox.
func (a *App) moveMessage(ref *MsgRef, dest string) {
	accountID, path, uid := ref.AccountID, ref.Path, ref.Env.UID
	a.removeRow(ref)
	go func() {
		acct := store.Get(accountID)
		if acct == nil {
			return
		}
		if err := mailx.MoveMessages(acct, path, []uint32{uid}, dest); err != nil {
			a.update(func() { a.flash("移动失败："+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash("已移动", "success") })
	}()
}

// removeRow drops a message from the list and clears the reader if it showed.
func (a *App) removeRow(ref *MsgRef) {
	key := ref.Key()
	out := a.msgs[:0]
	for _, m := range a.msgs {
		if m.Key() != key {
			out = append(out, m)
		}
	}
	a.msgs = out
	a.total--
	if a.total < 0 {
		a.total = 0
	}
	if a.sel != nil && a.sel.Key() == key {
		a.sel = nil
		a.msg = nil
		a.msgErr = ""
		a.loadBody()
	}
	a.rebuildRows()
	a.syncListSelection()
}

// ----------------------------------------------------------------- accounts

// addAccount verifies the credentials, stores the account and starts its
// watchers. It runs off the main thread from the wizard.
func (a *App) addAccount(in AddAccountInput) (string, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(email, "@") {
		return "", fmt.Errorf("请填写有效的邮箱地址")
	}
	if in.Password == "" {
		return "", fmt.Errorf("请填写密码或授权码")
	}

	providerID := in.Provider
	var preset *providers.Provider
	if p := providers.ByID(providerID); p != nil {
		preset = p
	} else if g := providers.Guess(email); g != nil {
		preset, providerID = g, g.ID
	} else {
		preset, providerID = providers.ByID("custom"), "custom"
	}

	imapHost := strings.TrimSpace(in.IMAPHost)
	if imapHost == "" {
		imapHost = preset.IMAP.Host
	}
	smtpHost := strings.TrimSpace(in.SMTPHost)
	if smtpHost == "" {
		smtpHost = preset.SMTP.Host
	}
	if imapHost == "" {
		return "", fmt.Errorf("缺少 IMAP 服务器地址")
	}
	if smtpHost == "" {
		return "", fmt.Errorf("缺少 SMTP 服务器地址")
	}
	imapPort := in.IMAPPort
	if imapPort == 0 {
		imapPort = preset.IMAP.Port
	}
	smtpPort := in.SMTPPort
	if smtpPort == 0 {
		smtpPort = preset.SMTP.Port
	}
	authUser := in.AuthUser
	if authUser == "" {
		authUser = email
	}

	if err := mailx.VerifyIMAP(imapHost, imapPort, preset.IMAP.Secure, authUser, in.Password); err != nil {
		return "", err
	}
	warning := ""
	if err := smtpx.Verify(smtpHost, smtpPort, preset.SMTP.Secure, authUser, in.Password); err != nil {
		warning = "SMTP 验证未通过（可能仍可收信）：" + err.Error()
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = email
	}
	account := store.Add(store.AddInput{
		Provider: providerID,
		Email:    email,
		Password: in.Password,
		Name:     name,
		Color:    in.Color,
		AuthUser: authUser,
		IMAP:     providers.Server{Host: imapHost, Port: imapPort, Secure: preset.IMAP.Secure},
		SMTP:     providers.Server{Host: smtpHost, Port: smtpPort, Secure: preset.SMTP.Secure, RequireTLS: preset.SMTP.RequireTLS},
	})
	watchers.Start(account)
	watchers.KickInboxSync(account)
	a.refreshAccounts()
	return warning, nil
}

// clearCache drops cached bodies and attachments; mail on the server stays.
func (a *App) clearCache() {
	entries, err := os.ReadDir(store.CacheDir)
	if err != nil {
		a.flash("清除失败："+err.Error(), "error")
		return
	}
	n := 0
	for _, e := range entries {
		if os.RemoveAll(filepath.Join(store.CacheDir, e.Name())) == nil {
			n++
		}
	}
	a.flash(fmt.Sprintf("已清除缓存（%d 项）", n), "success")
}

func (a *App) removeAccount(id string) {
	watchers.Stop(id)
	mailx.DropConnection(id)
	store.Remove(id)
	a.refreshAccounts()
}

func (a *App) patchAccount(id, name, color string) {
	var nameP, colorP *string
	if name != "" {
		nameP = &name
	}
	if color != "" {
		colorP = &color
	}
	store.Patch(id, nameP, colorP)
	a.refreshAccounts()
}

// --------------------------------------------------------------- subscribe

// SubscribeToWorkers redraws the window for sync events, coalescing bursts
// into one reload.
func (a *App) SubscribeToWorkers() {
	var mu sync.Mutex
	var pending *time.Timer
	schedule := func() {
		mu.Lock()
		defer mu.Unlock()
		if pending != nil {
			return
		}
		pending = time.AfterFunc(500*time.Millisecond, func() {
			mu.Lock()
			pending = nil
			mu.Unlock()
			a.update(func() {
				statuses := watchers.Statuses()
				for _, acc := range a.accounts {
					if st, ok := statuses[acc.ID]; ok {
						acc.Status, acc.Detail, acc.Unread = st.Status, st.Detail, st.Unread
					}
				}
				a.statuses = statuses
				a.recomputeUnread()
				if a.view.Kind != "folder" {
					a.reloadList(true)
				}
			})
		})
	}
	hub.Subscribe(func(ev hub.Event) {
		switch ev.Name {
		case "account:status", "mail:counts", "accounts:changed", "flags:changed", "message:updated":
			schedule()
		}
	})
}
