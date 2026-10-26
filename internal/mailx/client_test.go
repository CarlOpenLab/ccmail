package mailx

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// literal adapts a byte slice to imap.LiteralReader for appends.
type literal struct{ *bytes.Reader }

func (l literal) Size() int64 { return int64(l.Reader.Len()) }

// memIMAP serves an in-memory IMAP server with a seeded INBOX and returns a
// client logged in to it. The server runs without TLS (InsecureAuth), so the
// test drives the client code paths without certificates.
func memIMAP(t *testing.T, messages []string) *imapclient.Client {
	t.Helper()
	mem := imapmemserver.New()
	user := imapmemserver.NewUser("user@example.com", "pw")
	if err := user.Create("INBOX", nil); err != nil {
		t.Fatalf("create INBOX: %v", err)
	}
	for _, raw := range messages {
		if _, err := user.Append("INBOX", literal{bytes.NewReader([]byte(raw))}, &imap.AppendOptions{}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	mem.AddUser(user)

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), &imapserver.GreetingData{}, nil
		},
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go srv.Serve(ln) //nolint:errcheck // closed via Cleanup
	t.Cleanup(func() {
		srv.Close()
		ln.Close()
	})

	c, err := imapclient.DialInsecure(ln.Addr().String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	if err := c.Login("user@example.com", "pw").Wait(); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("select: %v", err)
	}
	return c
}

func rawMessage(subject, from string, daysAgo int) string {
	date := time.Now().AddDate(0, 0, -daysAgo).UTC().Format(time.RFC1123Z)
	return "From: " + from + "\r\nTo: user@example.com\r\nSubject: " + subject +
		"\r\nDate: " + date + "\r\nMessage-ID: <" + subject + "@example.com>\r\n" +
		"\r\nbody of " + subject + "\r\n"
}

// The client panics on a pointer NumSet ("imap: invalid NumSet type"), which
// crashed the app the moment the inbox synced. go-imap resolves UID vs
// sequence sets from the value type, so every call site must pass the set
// itself.
func TestFetchRangeAndRefreshFlags(t *testing.T) {
	c := memIMAP(t, []string{
		rawMessage("one", "a@example.com", 2),
		rawMessage("two", "b@example.com", 1),
		rawMessage("three", "c@example.com", 0),
	})

	envs, err := fetchRange(c, 1, 100)
	if err != nil {
		t.Fatalf("fetchRange: %v", err)
	}
	if len(envs) != 3 {
		t.Fatalf("fetched %d envelopes, want 3", len(envs))
	}
	subjects := map[string]bool{}
	for _, e := range envs {
		subjects[e.Subject] = true
		if e.From.Email == "" || e.Date == "" {
			t.Errorf("envelope %q missing from/date: %+v", e.Subject, e.From)
		}
	}
	for _, want := range []string{"one", "two", "three"} {
		if !subjects[want] {
			t.Errorf("subject %q missing from %v", want, subjects)
		}
	}

	// refreshFlags is the second FETCH call site over a UID range.
	refreshFlags(c, envs)
	if envs[0].Flags.Seen {
		t.Error("unexpected seen flag after refresh")
	}

	// an empty or inverted range must not issue a command at all
	empty, err := fetchRange(c, 100, 1)
	if err != nil || len(empty) != 0 {
		t.Fatalf("fetchRange(100,1) = %v, %v", empty, err)
	}
}

// Store over a UID set goes through the same NumSet resolution; the flag
// actions (star / archive / delete) all use it.
func TestStoreOverUIDSet(t *testing.T) {
	c := memIMAP(t, []string{rawMessage("one", "a@example.com", 0)})
	envs, err := fetchRange(c, 1, 100)
	if err != nil || len(envs) != 1 {
		t.Fatalf("seed fetch: %v, %d", err, len(envs))
	}

	set := imap.UIDSet{}
	set.AddNum(imap.UID(envs[0].UID))
	cmd := c.Store(set, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Flags: []imap.Flag{imap.FlagFlagged}, Silent: true}, nil)
	if err := cmd.Close(); err != nil {
		t.Fatalf("store: %v", err)
	}
	refreshFlags(c, envs)
	if !envs[0].Flags.Flagged {
		t.Error("flagged flag did not stick")
	}
}

var _ io.Reader = literal{}
