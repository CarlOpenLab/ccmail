// Package api exposes the HTTP surface: same routes and JSON shapes as the
// Node version, mounted on the mygo "api" custom protocol (and optionally a
// loopback listener for browser/debug mode).
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"glossmail/internal/hub"
	"glossmail/internal/mailx"
	"glossmail/internal/prefs"
	"glossmail/internal/providers"
	"glossmail/internal/smtpx"
	"glossmail/internal/store"
	"glossmail/internal/watchers"
)

// New builds the API mux.
func New() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "name": "Gloss Mail", "version": "2.0.0"})
	})
	mux.HandleFunc("GET /api/providers", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"providers": providers.All()})
	})
	mux.HandleFunc("GET /api/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"accounts": accountsWithStatus()})
	})
	mux.HandleFunc("GET /api/stream", hub.ServeSSE)

	// ---- accounts
	mux.HandleFunc("POST /api/accounts", addAccount)
	mux.HandleFunc("PATCH /api/accounts/{id}", patchAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}", deleteAccount)

	// ---- mail
	mux.HandleFunc("GET /api/accounts/{id}/mailboxes", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		boxes, err := mailx.ListMailboxes(a)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"mailboxes": boxes})
	})

	mux.HandleFunc("GET /api/accounts/{id}/messages", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		mailbox := r.URL.Query().Get("path")
		if mailbox == "" {
			http.Error(w, `{"error":"缺少邮箱路径"}`, 400)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		result, err := mailx.ListMessages(a, mailbox, mailx.Query{
			Page: page, Search: r.URL.Query().Get("search"), Filter: r.URL.Query().Get("filter"),
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, result)
		watchers.KickInboxSync(a)
	})

	mux.HandleFunc("GET /api/accounts/{id}/message", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		uid, _ := strconv.ParseUint(r.URL.Query().Get("uid"), 10, 32)
		if r.URL.Query().Get("path") == "" || uid == 0 {
			http.Error(w, `{"error":"缺少参数"}`, 400)
			return
		}
		markSeen := r.URL.Query().Get("mark") != "0"
		msg, err := mailx.GetMessage(a, r.URL.Query().Get("path"), uint32(uid), markSeen)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"message": msg})
	})

	mux.HandleFunc("PATCH /api/accounts/{id}/flags", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		var body struct {
			Path    string   `json:"path"`
			UIDs    []uint32 `json:"uids"`
			Seen    *bool    `json:"seen"`
			Flagged *bool    `json:"flagged"`
		}
		if !readBody(w, r, &body) {
			return
		}
		if err := mailx.SetFlags(a, body.Path, body.UIDs, mailx.FlagPatch{Seen: body.Seen, Flagged: body.Flagged}); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("POST /api/accounts/{id}/move", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		var body struct {
			Path string   `json:"path"`
			UIDs []uint32 `json:"uids"`
			Dest string   `json:"dest"`
		}
		if !readBody(w, r, &body) || body.Path == "" || body.Dest == "" {
			http.Error(w, `{"error":"缺少参数"}`, 400)
			return
		}
		if err := mailx.MoveMessages(a, body.Path, body.UIDs, body.Dest); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("POST /api/accounts/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		var body struct {
			Path string   `json:"path"`
			UIDs []uint32 `json:"uids"`
		}
		if !readBody(w, r, &body) {
			return
		}
		if err := mailx.DeleteMessages(a, body.Path, body.UIDs); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("POST /api/accounts/{id}/archive", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		var body struct {
			Path string   `json:"path"`
			UIDs []uint32 `json:"uids"`
		}
		if !readBody(w, r, &body) {
			return
		}
		if err := mailx.ArchiveMessages(a, body.Path, body.UIDs); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("GET /api/accounts/{id}/attachment", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		uid, _ := strconv.ParseUint(r.URL.Query().Get("uid"), 10, 32)
		file, filename, contentType, err := mailx.ResolveAttachment(a, r.URL.Query().Get("path"), uint32(uid), r.URL.Query().Get("att"))
		if err != nil {
			writeErr(w, err)
			return
		}
		f, oerr := os.Open(file)
		if oerr != nil {
			writeErr(w, &mailx.Error{Msg: "附件读取失败", Status: 502})
			return
		}
		defer f.Close()
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", urlEscape(filename)))
		io.Copy(w, f)
	})

	mux.HandleFunc("POST /api/accounts/{id}/prefetch", func(w http.ResponseWriter, r *http.Request) {
		a := needAccount(w, r)
		if a == nil {
			return
		}
		var body struct {
			Path string   `json:"path"`
			UIDs []uint32 `json:"uids"`
		}
		if readBody(w, r, &body) {
			go func() {
				for i, uid := range body.UIDs {
					if i >= 30 {
						break
					}
					_, _ = mailx.GetMessage(a, body.Path, uid, false)
				}
			}()
		}
		writeJSON(w, map[string]any{"ok": true, "queued": len(body.UIDs)})
	})

	mux.HandleFunc("GET /api/prefs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"prefs": prefs.All()})
	})
	mux.HandleFunc("PUT /api/prefs", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if !readBody(w, r, &body) {
			return
		}
		prefs.SetMany(body)
		hub.Broadcast("prefs:changed", map[string]any{"prefs": prefs.All()})
		writeJSON(w, map[string]any{"prefs": prefs.All()})
	})
	mux.HandleFunc("POST /api/cache/clear", func(w http.ResponseWriter, r *http.Request) {
		entries, _ := os.ReadDir(store.CacheDir)
		n := 0
		for _, e := range entries {
			if os.RemoveAll(filepath.Join(store.CacheDir, e.Name())) == nil {
				n++
			}
		}
		writeJSON(w, map[string]any{"ok": true, "removed": n})
	})

	mux.HandleFunc("GET /api/unified", unified)
	mux.HandleFunc("GET /api/contacts", contacts)
	mux.HandleFunc("POST /api/send", send)

	return mux
}

// ---- handlers

func addAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Color    string `json:"color"`
		AuthUser string `json:"authUser"`
		IMAP     struct {
			Host   string `json:"host"`
			Port   int    `json:"port"`
			Secure *bool  `json:"secure"`
		} `json:"imap"`
		SMTP struct {
			Host       string `json:"host"`
			Port       int    `json:"port"`
			Secure     *bool  `json:"secure"`
			RequireTLS bool   `json:"requireTLS"`
		} `json:"smtp"`
	}
	if !readBody(w, r, &body) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if !strings.Contains(email, "@") {
		http.Error(w, `{"error":"请填写有效的邮箱地址"}`, 400)
		return
	}
	if body.Password == "" {
		http.Error(w, `{"error":"请填写密码或授权码"}`, 400)
		return
	}

	providerID := body.Provider
	var preset *providers.Provider
	if p := providers.ByID(providerID); p != nil {
		preset = p
	} else if g := providers.Guess(email); g != nil {
		preset = g
		providerID = g.ID
	} else {
		preset = providers.ByID("custom")
		providerID = "custom"
	}

	imapHost := strings.TrimSpace(body.IMAP.Host)
	if imapHost == "" {
		imapHost = preset.IMAP.Host
	}
	smtpHost := strings.TrimSpace(body.SMTP.Host)
	if smtpHost == "" {
		smtpHost = preset.SMTP.Host
	}
	if imapHost == "" {
		http.Error(w, `{"error":"缺少 IMAP 服务器地址"}`, 400)
		return
	}
	if smtpHost == "" {
		http.Error(w, `{"error":"缺少 SMTP 服务器地址"}`, 400)
		return
	}
	imapPort := body.IMAP.Port
	if imapPort == 0 {
		imapPort = preset.IMAP.Port
	}
	imapSecure := preset.IMAP.Secure
	if body.IMAP.Secure != nil {
		imapSecure = *body.IMAP.Secure
	}
	smtpPort := body.SMTP.Port
	if smtpPort == 0 {
		smtpPort = preset.SMTP.Port
	}
	smtpSecure := preset.SMTP.Secure
	if body.SMTP.Secure != nil {
		smtpSecure = *body.SMTP.Secure
	}
	smtpRequireTLS := body.SMTP.RequireTLS || preset.SMTP.RequireTLS

	if err := mailx.VerifyIMAP(imapHost, imapPort, imapSecure, orDefault(body.AuthUser, email), body.Password); err != nil {
		writeErr(w, err)
		return
	}

	// best-effort SMTP check (a failure still allows mounting: the account
	// may be receive-only while we warn in the wizard)
	smtpWarning := ""
	if verr := smtpx.Verify(smtpHost, smtpPort, smtpSecure, orDefault(body.AuthUser, email), body.Password); verr != nil {
		smtpWarning = "SMTP 验证未通过（可能仍可收信）：" + verr.Error()
	}

	account := store.Add(store.AddInput{
		Provider: providerID, Email: email, Password: body.Password, Name: body.Name, Color: body.Color,
		AuthUser: orDefault(body.AuthUser, email),
		IMAP:     providers.Server{Host: imapHost, Port: imapPort, Secure: imapSecure},
		SMTP:     providers.Server{Host: smtpHost, Port: smtpPort, Secure: smtpSecure, RequireTLS: smtpRequireTLS},
	})
	watchers.Start(account)
	watchers.KickInboxSync(account)
	hub.Broadcast("accounts:changed", map[string]any{})
	writeJSON(w, map[string]any{
		"account":     account.Public(map[string]any{"status": "connecting"}),
		"smtpWarning": smtpWarning,
	})
}

func patchAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if !readBody(w, r, &body) {
		return
	}
	updated := store.Patch(r.PathValue("id"), body.Name, body.Color)
	if updated == nil {
		http.Error(w, `{"error":"账号不存在"}`, 404)
		return
	}
	hub.Broadcast("accounts:changed", map[string]any{})
	writeJSON(w, map[string]any{"account": updated.Public(nil)})
}

func deleteAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	watchers.Stop(id)
	mailx.DropConnection(id)
	if !store.Remove(id) {
		http.Error(w, `{"error":"账号不存在"}`, 404)
		return
	}
	hub.Broadcast("accounts:changed", map[string]any{})
	writeJSON(w, map[string]any{"ok": true})
}

func unified(w http.ResponseWriter, r *http.Request) {
	kind := orDefault(r.URL.Query().Get("kind"), "inbox")
	q := mailx.Query{Search: r.URL.Query().Get("search"), Filter: r.URL.Query().Get("filter")}

	type item struct {
		*mailx.Envelope
		AccountID    string `json:"accountId"`
		AccountName  string `json:"accountName"`
		AccountColor string `json:"accountColor"`
		Path         string `json:"path"`
	}
	var items []item
	for _, a := range store.All() {
		var rows []*mailx.Envelope
		if cached := mailx.CachedEnvelopes(a.ID, "INBOX"); cached != nil {
			rows = cached
		} else {
			watchers.KickInboxSync(a)
			continue
		}
		for _, m := range rows {
			if kind == "flagged" && !m.Flags.Flagged {
				continue
			}
			if q.Search != "" || q.Filter != "" {
				if filtered := filterOne(m, q); filtered == nil {
					continue
				}
			}
			if len(items) < 150 {
				items = append(items, item{Envelope: m, AccountID: a.ID, AccountName: a.Name, AccountColor: a.Color, Path: "INBOX"})
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Date > items[j].Date })
	if items == nil {
		items = []item{}
	}
	writeJSON(w, map[string]any{"total": len(items), "messages": items})
}

func filterOne(m *mailx.Envelope, q mailx.Query) *mailx.Envelope {
	switch q.Filter {
	case "unread":
		if m.Flags.Seen {
			return nil
		}
	case "flagged":
		if !m.Flags.Flagged {
			return nil
		}
	case "attachments":
		if !m.HasAttachments {
			return nil
		}
	}
	if q.Search != "" {
		s := strings.ToLower(q.Search)
		if !strings.Contains(strings.ToLower(m.Subject), s) &&
			!strings.Contains(strings.ToLower(m.From.Name), s) &&
			!strings.Contains(strings.ToLower(m.From.Email), s) &&
			!strings.Contains(strings.ToLower(m.Snippet), s) {
			return nil
		}
	}
	return m
}

func contacts(w http.ResponseWriter, r *http.Request) {
	freq := map[string]*contact{}
	add := func(a mailx.Addr) {
		if a.Email == "" {
			return
		}
		key := strings.ToLower(a.Email)
		c := freq[key]
		if c == nil {
			name := a.Name
			if name == "" {
				name = strings.SplitN(a.Email, "@", 2)[0]
			}
			c = &contact{Name: name, Email: a.Email}
			freq[key] = c
		}
		c.Count++
		if a.Name != "" && c.Count < 3 {
			c.Name = a.Name
		}
	}
	for _, a := range store.All() {
		if cached := mailx.CachedEnvelopes(a.ID, "INBOX"); cached != nil {
			for _, m := range cached {
				add(m.From)
				for _, t := range m.To {
					add(t)
				}
			}
		}
	}
	list := make([]*contact, 0, len(freq))
	for _, c := range freq {
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })
	if len(list) > 80 {
		list = list[:80]
	}
	writeJSON(w, map[string]any{"contacts": list})
}

type contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Count int    `json:"-"`
}

func send(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AccountID   string             `json:"accountId"`
		To          []smtpx.Recipient  `json:"to"`
		Cc          []smtpx.Recipient  `json:"cc"`
		Bcc         []smtpx.Recipient  `json:"bcc"`
		Subject     string             `json:"subject"`
		HTML        string             `json:"html"`
		Text        string             `json:"text"`
		Attachments []smtpx.Attachment `json:"attachments"`
	}
	if !readBody(w, r, &body) {
		return
	}
	a := store.Get(body.AccountID)
	if a == nil {
		http.Error(w, `{"error":"账号不存在"}`, 404)
		return
	}
	result, err := smtpx.Send(a, &smtpx.Outgoing{
		To: body.To, Cc: body.Cc, Bcc: body.Bcc, Subject: body.Subject,
		HTML: body.HTML, Text: body.Text, Attachments: body.Attachments,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, result)
}

// ---- helpers

func needAccount(w http.ResponseWriter, r *http.Request) *store.Account {
	a := store.Get(r.PathValue("id"))
	if a == nil {
		http.Error(w, `{"error":"账号不存在"}`, 404)
		return nil
	}
	return a
}

func accountsWithStatus() []map[string]any {
	statuses := watchers.Statuses()
	out := []map[string]any{}
	for _, a := range store.All() {
		extra := map[string]any{"status": "offline"}
		if s, ok := statuses[a.ID]; ok {
			extra["status"] = s.Status
			extra["detail"] = s.Detail
			extra["unread"] = s.Unread
			extra["total"] = s.Total
			if !s.LastSync.IsZero() {
				extra["lastSync"] = s.LastSync.Format("15:04:05")
			}
		}
		out = append(out, a.Public(extra))
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := 502
	if me, ok := err.(*mailx.Error); ok {
		status = me.Status
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func readBody(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 40<<20))
	if err != nil {
		http.Error(w, `{"error":"读取请求失败"}`, 400)
		return false
	}
	if len(raw) == 0 {
		return true
	}
	if err := json.Unmarshal(raw, v); err != nil {
		http.Error(w, `{"error":"请求格式错误"}`, 400)
		return false
	}
	return true
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r > 127 || r == '%' || r == '"' || r <= 32 {
			b.WriteString(fmt.Sprintf("%%%02X", r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

