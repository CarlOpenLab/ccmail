package ccapp

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ccmail/internal/mailx"
	"ccmail/internal/store"
	"ccmail/locales"
)

// writeEnvCache plants an envelope cache for one mailbox, as a sync would
// leave it, so fetchList can page it without touching IMAP.
func writeEnvCache(t *testing.T, accountID, mailbox string, n int, flagged int) {
	t.Helper()
	envs := make([]*mailx.Envelope, 0, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		envs = append(envs, &mailx.Envelope{
			UID:     uint32(1000 - i),
			Subject: subjectsFor(i),
			From:    mailx.Addr{Name: "缓存发件人", Email: "cache@example.com"},
			Date:    now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
			Flags:   mailx.EnvelopeFlags{Seen: i > flagged, Flagged: i < flagged},
			Snippet: "snippet",
		})
	}
	doc := map[string]any{
		"path": mailbox, "uidValidity": "1", "uidNext": 2000,
		"messages": envs, "syncedAt": now.UnixMilli(),
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum([]byte(mailbox))
	file := filepath.Join(store.CacheDir, store.SafeDir(accountID), "env-"+hex.EncodeToString(sum[:])[:12]+".json")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func subjectsFor(i int) string {
	if i%10 == 0 {
		return "needle 第 " + itoa(i) + " 封"
	}
	return "第 " + itoa(i) + " 封缓存邮件"
}

func TestUnifiedFetchListPages(t *testing.T) {
	acct := store.Add(store.AddInput{Provider: "custom", Email: "pager@example.com", Password: "x", Name: "分页"})
	defer store.Remove(acct.ID)
	writeEnvCache(t, acct.ID, "INBOX", 120, 3)

	rows, total, err := fetchList(ViewSel{Kind: "unified"}, "", "", 1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if total != 120 {
		t.Fatalf("total = %d, want 120", total)
	}
	if len(rows) != mailx.PageSize {
		t.Fatalf("page 1 has %d rows, want %d", len(rows), mailx.PageSize)
	}
	if rows[0].Env.UID != 1000 {
		t.Fatalf("newest first, got UID %d", rows[0].Env.UID)
	}

	rows2, total2, err := fetchList(ViewSel{Kind: "unified"}, "", "", 2)
	if err != nil || total2 != 120 {
		t.Fatalf("page 2: rows=%d total=%d err=%v", len(rows2), total2, err)
	}
	if len(rows2) != mailx.PageSize {
		t.Fatalf("page 2 has %d rows, want %d", len(rows2), mailx.PageSize)
	}
	if rows2[0].Env.UID == rows[0].Env.UID {
		t.Fatal("page 2 repeats page 1")
	}

	rows3, _, err := fetchList(ViewSel{Kind: "unified"}, "", "", 3)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(rows3) != 120-2*mailx.PageSize {
		t.Fatalf("page 3 has %d rows, want %d", len(rows3), 120-2*mailx.PageSize)
	}
	rows4, _, _ := fetchList(ViewSel{Kind: "unified"}, "", "", 4)
	if len(rows4) != 0 {
		t.Fatalf("page past the end has %d rows", len(rows4))
	}
}

func TestUnifiedFetchListFilters(t *testing.T) {
	acct := store.Add(store.AddInput{Provider: "custom", Email: "filter@example.com", Password: "x", Name: "筛选"})
	defer store.Remove(acct.ID)
	writeEnvCache(t, acct.ID, "INBOX", 120, 3)

	_, total, err := fetchList(ViewSel{Kind: "flagged"}, "", "", 1)
	if err != nil {
		t.Fatalf("flagged view: %v", err)
	}
	if total != 3 {
		t.Fatalf("flagged total = %d, want 3", total)
	}

	_, total, err = fetchList(ViewSel{Kind: "unified"}, "needle", "", 1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 12 {
		t.Fatalf("search total = %d, want 12", total)
	}
}

func TestWantNextPage(t *testing.T) {
	a, tt := newTestApp(t)
	if a.wantNextPage() {
		t.Fatal("no more pages, should not want one")
	}
	a.more, a.total = true, 200
	tt.Frame()
	if !a.wantNextPage() {
		t.Fatal("the list shows its end and more pages exist")
	}
	a.loading = true
	if a.wantNextPage() {
		t.Fatal("a load in flight must not re-trigger")
	}
	a.loading = false
	a.loadErr = "boom"
	if a.wantNextPage() {
		t.Fatal("an error must not auto-retry forever")
	}
}

func TestLoadMoreFooterStates(t *testing.T) {
	a, tt := newTestApp(t)
	a.more, a.total = true, 200
	tt.Frame()
	if !tt.HasText(locales.ListLoadMore(len(a.msgs), a.total)) {
		t.Fatal("load-more button missing while a page remains")
	}

	// loading the next page swaps the button for a spinner
	a.pending = nil
	a.loading = true
	tt.Frame()
	if tt.HasText(locales.ListLoadMore(len(a.msgs), a.total)) {
		t.Fatal("button still shown while loading")
	}
	if !tt.HasText(locales.ListLoadingMore()) {
		t.Fatal("loading indicator missing")
	}
}

func TestAccountDotRenders(t *testing.T) {
	_, tt := newTestApp(t)
	tt.Frame()
	if !tt.HasText(locales.TitlebarSwitchAccount("工作邮箱")) {
		t.Fatal("account dot not rendered as a switch button")
	}
	if !tt.HasText(locales.TitlebarSwitchAccount("个人邮箱")) {
		t.Fatal("second account dot missing")
	}
}
