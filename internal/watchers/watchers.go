// Package watchers keeps per-account status and pushes live updates: a
// dedicated IMAP connection per real account runs IDLE on the INBOX and
// triggers a sync whenever EXISTS changes; new mail raises a desktop
// notification and an event to the frontend.
package watchers

import (
	"fmt"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"ccmail/internal/hub"
	"ccmail/internal/mailx"
	"ccmail/internal/prefs"
	"ccmail/internal/store"
	"ccmail/locales"
)

// Status is one account's connection state.
type Status struct {
	Status      string    `json:"status"`
	Detail      string    `json:"detail"`
	Unread      int       `json:"unread"`
	Total       int       `json:"total"`
	LastSync    time.Time `json:"-"`
	hasReported bool
}

var (
	mu       sync.Mutex
	statuses = map[string]*Status{}
	stopped  = map[string]bool{}
)

// Statuses copies the current status map.
func Statuses() map[string]Status {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]Status, len(statuses))
	for id, s := range statuses {
		out[id] = *s
	}
	return out
}

func setStatus(id, status, detail string) {
	mu.Lock()
	if statuses[id] == nil {
		statuses[id] = &Status{}
	}
	statuses[id].Status = status
	statuses[id].Detail = detail
	mu.Unlock()
	hub.Broadcast("account:status", map[string]any{
		"accountId": id, "status": status, "detail": detail,
	})
}

// KickInboxSync triggers a background INBOX sync + count broadcast.
func KickInboxSync(a *store.Account) {
	if a == nil || a.Type == "demo" {
		return
	}
	go syncCounts(a)
}

// syncCounts syncs the INBOX, broadcasts counts and notifies on new mail.
func syncCounts(a *store.Account) {
	setStatus(a.ID, "connecting", "")
	if _, err := mailx.ListMessages(a, "INBOX", mailx.Query{Page: 1}); err != nil {
		setStatus(a.ID, "offline", err.Error())
		return
	}
	setStatus(a.ID, "online", "")

	rows := mailx.CachedEnvelopes(a.ID, "INBOX")
	if rows == nil {
		return
	}
	unread, total := 0, len(rows)
	for _, m := range rows {
		if !m.Flags.Seen {
			unread++
		}
	}

	mu.Lock()
	s := statuses[a.ID]
	if s == nil {
		s = &Status{}
		statuses[a.ID] = s
	}
	prevUnread, hadReported := s.Unread, s.hasReported
	increase := hadReported && unread > prevUnread
	s.Unread, s.Total, s.hasReported = unread, total, true
	s.LastSync = time.Now()
	mu.Unlock()

	hub.Broadcast("mail:counts", map[string]any{
		"accountId": a.ID, "path": "INBOX", "unread": unread, "total": total,
		"lastSync": time.Now().Format("15:04:05"),
	})
	if increase {
		notifyNewMail(a, rows)
	}
}

// notifyNewMail raises an OS notification for the newest unseen message.
func notifyNewMail(a *store.Account, rows []*mailx.Envelope) {
	if !prefs.NotificationsEnabled() || !mygo.NotificationsSupported() {
		return
	}
	body := locales.NotifyNewMail(a.Name)
	for _, m := range rows {
		if !m.Flags.Seen {
			subject := m.Subject
			if len(subject) > 42 {
				subject = subject[:42] + "…"
			}
			body = fmt.Sprintf("%s：%s", m.From.Name, subject)
			break
		}
	}
	n := mygo.NewNotification(mygo.NotificationOptions{
		ID:    "ccmail-new-mail-" + a.ID,
		Title: "ccmail",
		Body:  body,
		Group: "ccmail",
	})
	n.OnClick(func() {
		mygo.App.Focus()
	})
	if err := n.Show(); err != nil {
		// notifications may be denied; the in-app toast still fires
		_ = err
	}
}

// Start begins watching an account (idempotent); one goroutine per account.
func Start(a *store.Account) {
	if a == nil || a.Type == "demo" {
		return
	}
	mu.Lock()
	if _, exists := statuses[a.ID]; exists && !stopped[a.ID] {
		mu.Unlock()
		return
	}
	stopped[a.ID] = false
	statuses[a.ID] = &Status{Status: "connecting"}
	mu.Unlock()
	go watchLoop(a)
}

// Stop halts the watcher of an account.
func Stop(id string) {
	mu.Lock()
	stopped[id] = true
	delete(statuses, id)
	isStopped := stopped[id]
	mu.Unlock()
	_ = isStopped
}

func isStopped(id string) bool {
	mu.Lock()
	defer mu.Unlock()
	return stopped[id]
}

// watchLoop maintains one IDLE connection on the INBOX, reconnecting with
// backoff; falls back to polling when the server lacks IDLE.
func watchLoop(a *store.Account) {
	backoff := 15 * time.Second
	for {
		if isStopped(a.ID) || store.Get(a.ID) == nil {
			return
		}
		client, err := mailx.DialWatcher(a)
		if err != nil {
			setStatus(a.ID, "offline", err.Error())
			if !sleepOrStopped(backoff, a.ID) {
				return
			}
			backoff = min(backoff*2, 5*time.Minute)
			continue
		}
		backoff = 15 * time.Second
		setStatus(a.ID, "online", "")

		sel, err := client.Select("INBOX", nil).Wait()
		if err != nil {
			client.Close()
			if !sleepOrStopped(backoff, a.ID) {
				return
			}
			continue
		}
		lastCount := sel.NumMessages
		KickInboxSync(a)

		idleCmd, idleErr := client.Idle()
		if idleErr != nil {
			// server without IDLE: poll instead
			client.Close()
			if !sleepOrStopped(60*time.Second, a.ID) {
				return
			}
			syncCounts(accountRef(a))
			continue
		}

		// re-issue IDLE well before the common 29-minute server cap
		deadline := time.Now().Add(25 * time.Minute)
		ticker := time.NewTicker(10 * time.Second)
	idleWatch:
		for time.Now().Before(deadline) {
			select {
			case <-ticker.C:
				if isStopped(a.ID) {
					break idleWatch
				}
				if mb := client.Mailbox(); mb != nil && mb.NumMessages != lastCount {
					lastCount = mb.NumMessages
					KickInboxSync(accountRef(a))
				}
			}
		}
		ticker.Stop()
		_ = idleCmd.Close()
		_ = idleCmd.Wait()
		client.Close()
		if isStopped(a.ID) {
			return
		}
	}
}

func accountRef(a *store.Account) *store.Account {
	if cur := store.Get(a.ID); cur != nil {
		return cur
	}
	return a
}

func sleepOrStopped(d time.Duration, id string) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	<-t.C
	return !isStopped(id)
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// InitAll starts watchers for every stored real account.
func InitAll() {
	for _, a := range store.All() {
		Start(a)
	}
}
