package ccapp

import (
	"errors"
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
	"ccmail/locales"
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
		Language:      get("language", "auto"),
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
		Language:      "auto",
		Wash:          "plain",
		Density:       "cozy",
		MarkRead:      true,
		Notifications: true,
		QuoteReply:    true,
	}
}

// StartupLocale reports the saved language for mygo's locale wiring at boot.
func StartupLocale() string { return prefs.Get("language", "auto") }

func (a *App) savePrefs() {
	on := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	prefs.SetMany(map[string]string{
		"theme":         a.prefs.Theme,
		"language":      a.prefs.Language,
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
				// the cache may have shifted between pages; never show a row twice
				seen := make(map[string]bool, len(a.msgs)+len(rows))
				for _, m := range a.msgs {
					seen[m.Key()] = true
				}
				for _, m := range rows {
					if !seen[m.Key()] {
						seen[m.Key()] = true
						a.msgs = append(a.msgs, m)
					}
				}
			}
			a.total = total
			a.more = len(a.msgs) < total && len(rows) > 0
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
	a.list.Selected = &a.listSel
}

func fetchList(view ViewSel, query, filter string, page int) ([]*MsgRef, int, error) {
	if view.Kind == "folder" {
		acct := store.Get(view.AccountID)
		if acct == nil {
			return nil, 0, errors.New(locales.ComposeAccountMissing())
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

	// unified / starred: aggregate cached INBOX envelopes, paged like a
	// folder view so older mail stays reachable
	var all []*MsgRef
	for _, acct := range store.All() {
		rows := mailx.CachedEnvelopes(acct.ID, "INBOX")
		if rows == nil {
			watchers.KickInboxSync(acct)
			continue
		}
		if view.Kind == "flagged" {
			rows = mailx.ApplyQuery(rows, mailx.Query{Filter: "flagged"})
		} else if query != "" || filter != "" {
			rows = mailx.ApplyQuery(rows, mailx.Query{Search: query, Filter: filter})
		}
		vm := accountVMFromStore(acct)
		for _, env := range rows {
			all = append(all, &MsgRef{AccountID: vm.ID, AccountName: vm.Name, Color: vm.Color, Path: "INBOX", Env: env})
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		ti, oki := parseDate(all[i].Env.Date)
		tj, okj := parseDate(all[j].Env.Date)
		if oki && okj {
			return ti.After(tj)
		}
		return all[i].Env.Date > all[j].Env.Date
	})
	total := len(all)
	start := (page - 1) * mailx.PageSize
	if start > total {
		start = total
	}
	end := start + mailx.PageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func accountVMFromStore(s *store.Account) *AccountVM {
	return &AccountVM{ID: s.ID, Name: s.Name, Email: s.Email, Color: s.Color, Provider: s.Provider}
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
	go func() {
		acct := store.Get(accountID)
		if acct == nil {
			a.update(func() {
				a.msgLoading = false
				a.msgErr = locales.ComposeAccountMissing()
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
			a.update(func() { a.flash(locales.DataArchiveFailed()+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash(locales.DataArchived(), "success") })
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
			a.update(func() { a.flash(locales.DataDeleteFailed()+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash(locales.DataMovedToTrash(), "success") })
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
			a.update(func() { a.flash(locales.DataMoveFailed()+err.Error(), "error") })
			return
		}
		a.update(func() { a.flash(locales.DataMoved(), "success") })
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
		return "", errors.New(locales.ComposeBadEmail())
	}
	if in.Password == "" {
		return "", errors.New(locales.ComposeNeedPassword())
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
		return "", errors.New(locales.DataNeedImapHost())
	}
	if smtpHost == "" {
		return "", errors.New(locales.DataNeedSmtpHost())
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
		warning = locales.DataSmtpUnverified() + err.Error()
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
		a.flash(locales.DataClearFailed()+err.Error(), "error")
		return
	}
	n := 0
	for _, e := range entries {
		if os.RemoveAll(filepath.Join(store.CacheDir, e.Name())) == nil {
			n++
		}
	}
	a.flash(locales.DataCacheCleared(n), "success")
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
// into one reload. A bare account-status flip only recolors the sidebar; it
// must not throw the user back to the list's first page.
func (a *App) SubscribeToWorkers() {
	var mu sync.Mutex
	var pending *time.Timer
	var names map[string]bool
	schedule := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		if names == nil {
			names = map[string]bool{}
		}
		names[name] = true
		if pending != nil {
			return
		}
		pending = time.AfterFunc(500*time.Millisecond, func() {
			mu.Lock()
			got := names
			names = nil
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
				mailChanged := false
				for n := range got {
					if n != "account:status" {
						mailChanged = true
					}
				}
				if mailChanged && a.view.Kind != "folder" {
					a.reloadList(true)
				}
			})
		})
	}
	hub.Subscribe(func(ev hub.Event) {
		switch ev.Name {
		case "account:status", "mail:counts", "accounts:changed", "flags:changed", "message:updated":
			schedule(ev.Name)
		}
	})
}
