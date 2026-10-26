package mailx

import (
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"ccmail/internal/store"
)

const (
	syncWindow = 800 // recent messages kept per mailbox
	pageSize   = 50
)

func addrOf(a imap.Address) Addr {
	email := a.Addr()
	name := a.Name
	if name == "" && email != "" {
		parts := strings.SplitN(email, "@", 2)
		name = parts[0]
	}
	return Addr{Name: name, Email: email}
}

func addrList(list []imap.Address) []Addr {
	out := make([]Addr, 0, len(list))
	for _, a := range list {
		out = append(out, addrOf(a))
	}
	return out
}

func mapEnvelope(b *imapclient.FetchMessageBuffer) *Envelope {
	env := &Envelope{
		UID:     uint32(b.UID),
		Subject: "(无主题)",
		Date:    b.InternalDate.Format(time.RFC3339),
		Flags:   EnvelopeFlags{Seen: hasFlag(b.Flags, "\\Seen"), Flagged: hasFlag(b.Flags, "\\Flagged"), Answered: hasFlag(b.Flags, "\\Answered"), Draft: hasFlag(b.Flags, "\\Draft")},
		Size:    b.RFC822Size,
	}
	if b.Envelope != nil {
		if b.Envelope.Subject != "" {
			env.Subject = b.Envelope.Subject
		}
		env.MessageID = b.Envelope.MessageID
		if len(b.Envelope.From) > 0 {
			env.From = addrOf(b.Envelope.From[0])
		}
		env.To = addrList(b.Envelope.To)
		if !b.InternalDate.IsZero() {
			env.Date = b.InternalDate.Format(time.RFC3339)
		} else if !b.Envelope.Date.IsZero() {
			env.Date = b.Envelope.Date.Format(time.RFC3339)
		}
	}
	if env.From.Email == "" && env.From.Name == "" {
		env.From = Addr{Name: "未知发件人"}
	}
	env.HasAttachments = bodyHasAttachments(b.BodyStructure)
	return env
}

func hasFlag(flags []imap.Flag, want string) bool {
	for _, f := range flags {
		if string(f) == want {
			return true
		}
	}
	return false
}

// bodyHasAttachments walks BODYSTRUCTURE looking for attachment-ish parts.
func bodyHasAttachments(bs imap.BodyStructure) bool {
	found := false
	if bs == nil {
		return false
	}
	bs.Walk(func(path []int, part imap.BodyStructure) bool {
		if sp, ok := part.(*imap.BodyStructureSinglePart); ok {
			media := sp.MediaType() // e.g. "text/plain"
			if strings.HasPrefix(media, "multipart/") {
				return true
			}
			if sp.Filename() != "" {
				found = true
				return false
			}
			if d := sp.Disposition(); d != nil && strings.EqualFold(d.Value, "attachment") {
				found = true
				return false
			}
			if !strings.HasPrefix(media, "text/") && media != "message/rfc822" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// syncAndGetEnvelopes syncs one mailbox into the disk cache and returns it.
func syncAndGetEnvelopes(a *store.Account, mailbox string) (*envDoc, error) {
	var result *envDoc
	err := withClient(a, func(c *imapclient.Client) error {
		sel, err := c.Select(mailbox, nil).Wait()
		if err != nil {
			return errf(404, "邮箱不存在：%s", mailbox)
		}
		uidValidity := uint64ToString(uint64(sel.UIDValidity))
		uidNext := uint32(sel.UIDNext)

		cache := loadEnv(a.ID, mailbox)
		var messages []*Envelope
		if cache == nil || cache.UIDValidity != uidValidity {
			from := uint32(1)
			if uidNext > syncWindow {
				from = uidNext - syncWindow
			}
			msgs, err := fetchRange(c, from, uidNext-1)
			if err != nil {
				return err
			}
			messages = msgs
		} else {
			messages = cache.Messages
			if uidNext > cache.UIDNext {
				fresh, err := fetchRange(c, cache.UIDNext, uidNext-1)
				if err != nil {
					return err
				}
				messages = append(messages, fresh...)
			}
			refreshFlags(c, messages)
		}

		// dedupe by uid, sort by date desc
		seen := map[uint32]bool{}
		deduped := messages[:0]
		for _, m := range messages {
			if m == nil || m.UID == 0 || seen[m.UID] {
				continue
			}
			seen[m.UID] = true
			deduped = append(deduped, m)
		}
		sort.SliceStable(deduped, func(i, j int) bool {
			return deduped[i].Date > deduped[j].Date
		})

		doc := &envDoc{Path: mailbox, UIDValidity: uidValidity, UIDNext: uidNext, Messages: deduped, SyncedAt: time.Now().UnixMilli()}
		saveEnv(a.ID, mailbox, doc)
		result = doc
		return nil
	})
	return result, err
}

func uint64ToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func fetchRange(c *imapclient.Client, fromUID, toUID uint32) ([]*Envelope, error) {
	if toUID < fromUID || toUID == 0 {
		return nil, nil
	}
	// AddRange, not UIDSetNum: UIDSetNum(a, b) is the two single UIDs a and
	// b, not the range a:b.
	set := imap.UIDSet{}
	set.AddRange(imap.UID(fromUID), imap.UID(toUID))
	options := &imap.FetchOptions{
		UID: true, Envelope: true, Flags: true, InternalDate: true, RFC822Size: true,
		BodyStructure: &imap.FetchItemBodyStructure{Extended: false},
	}
	cmd := c.Fetch(set, options)
	buffers, err := cmd.Collect()
	if err != nil {
		return nil, errf(502, "拉取邮件列表失败：%s", explain(err))
	}
	out := make([]*Envelope, 0, len(buffers))
	for _, b := range buffers {
		out = append(out, mapEnvelope(b))
	}
	return out, nil
}

func refreshFlags(c *imapclient.Client, messages []*Envelope) {
	if len(messages) == 0 {
		return
	}
	minUID, maxUID := messages[0].UID, messages[0].UID
	for _, m := range messages {
		if m.UID < minUID {
			minUID = m.UID
		}
		if m.UID > maxUID {
			maxUID = m.UID
		}
	}
	set := imap.UIDSet{}
	set.AddRange(imap.UID(minUID), imap.UID(maxUID))
	cmd := c.Fetch(set, &imap.FetchOptions{UID: true, Flags: true})
	buffers, err := cmd.Collect()
	if err != nil {
		return
	}
	byUID := map[uint32]*Envelope{}
	for _, m := range messages {
		byUID[m.UID] = m
	}
	for _, b := range buffers {
		m, ok := byUID[uint32(b.UID)]
		if !ok {
			continue
		}
		m.Flags.Seen = hasFlag(b.Flags, "\\Seen")
		m.Flags.Flagged = hasFlag(b.Flags, "\\Flagged")
		m.Flags.Answered = hasFlag(b.Flags, "\\Answered")
		m.Flags.Draft = hasFlag(b.Flags, "\\Draft")
	}
}

// ListResult is one page of the message list.
type ListResult struct {
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Messages []*Envelope `json:"messages"`
}

// Query filters a list request.
type Query struct {
	Page   int
	Search string
	Filter string
}

// ListMessages returns a filtered page for an account + mailbox.
func ListMessages(a *store.Account, mailbox string, q Query) (*ListResult, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	doc, err := syncAndGetEnvelopes(a, mailbox)
	if err != nil {
		return nil, err
	}
	rows := applyQuery(doc.Messages, q)
	return pageOf(rows, page), nil
}

// CachedEnvelopes returns the on-disk envelope cache without any network
// activity (used by the unified inbox aggregator).
func CachedEnvelopes(accountID, mailbox string) []*Envelope {
	if doc := loadEnv(accountID, mailbox); doc != nil {
		return doc.Messages
	}
	return nil
}

func pageOf(rows []*Envelope, page int) *ListResult {
	start := (page - 1) * pageSize
	if start > len(rows) {
		start = len(rows)
	}
	end := start + pageSize
	if end > len(rows) {
		end = len(rows)
	}
	return &ListResult{
		Total: len(rows), Page: page, PageSize: pageSize,
		Messages: rows[start:end],
	}
}

func applyQuery(rows []*Envelope, q Query) []*Envelope {
	out := make([]*Envelope, 0, len(rows))
	search := strings.ToLower(q.Search)
	for _, m := range rows {
		switch q.Filter {
		case "unread":
			if m.Flags.Seen {
				continue
			}
		case "flagged":
			if !m.Flags.Flagged {
				continue
			}
		case "attachments":
			if !m.HasAttachments {
				continue
			}
		}
		if search != "" {
			subject := strings.ToLower(m.Subject)
			fromName := strings.ToLower(m.From.Name)
			fromEmail := strings.ToLower(m.From.Email)
			snippet := strings.ToLower(m.Snippet)
			if !strings.Contains(subject, search) && !strings.Contains(fromName, search) &&
				!strings.Contains(fromEmail, search) && !strings.Contains(snippet, search) {
				continue
			}
		}
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}
