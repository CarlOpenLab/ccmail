package mailx

import (
	"bytes"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"

	"ccmail/internal/hub"
	"ccmail/internal/store"
)

var wsRe = regexp.MustCompile(`\s+`)

// GetMessage returns the parsed full message, caching to disk.
func GetMessage(a *store.Account, mailbox string, uid uint32, markSeen bool) (*FullMessage, error) {
	if cached := loadMsg(a.ID, mailbox, uid); cached != nil {
		if markSeen {
			afterMessageOpened(a, mailbox, uid)
		}
		return cached, nil
	}

	var parsed *FullMessage
	err := withClient(a, func(c *imapclient.Client) error {
		set := imap.UIDSetNum(imap.UID(uid))
		options := &imap.FetchOptions{
			UID:         true,
			BodySection: []*imap.FetchItemBodySection{{Peek: true}},
		}
		cmd := c.Fetch(set, options)
		defer cmd.Close()
		var raw []byte
		for {
			msg := cmd.Next()
			if msg == nil {
				break
			}
			for {
				item := msg.Next()
				if item == nil {
					break
				}
				if bs, ok := item.(imapclient.FetchItemDataBodySection); ok && bs.Literal != nil {
					b, err := io.ReadAll(bs.Literal)
					if err != nil {
						return err
					}
					raw = b
				}
			}
		}
		if len(raw) == 0 {
			return errf(502, "邮件拉取失败")
		}
		m, err := parseRaw(raw, a, mailbox, uid)
		if err != nil {
			return err
		}
		parsed = m
		return nil
	})
	if err != nil {
		return nil, err
	}

	saveMsg(a.ID, mailbox, uid, parsed)

	// enrich the envelope cache with snippet / attachment info
	if doc := loadEnv(a.ID, mailbox); doc != nil {
		for _, e := range doc.Messages {
			if e.UID == uid {
				e.Snippet = parsed.Snippet
				e.HasAttachments = len(parsed.Attachments) > 0
			}
		}
		saveEnv(a.ID, mailbox, doc)
	}
	if markSeen {
		afterMessageOpened(a, mailbox, uid)
	}
	hub.Broadcast("message:updated", map[string]any{
		"accountId": a.ID, "path": mailbox, "uid": uid,
		"snippet": parsed.Snippet, "hasAttachments": len(parsed.Attachments) > 0,
	})
	return parsed, nil
}

func afterMessageOpened(a *store.Account, mailbox string, uid uint32) {
	if doc := loadEnv(a.ID, mailbox); doc != nil {
		for _, e := range doc.Messages {
			if e.UID == uid && !e.Flags.Seen {
				_ = SetFlags(a, mailbox, []uint32{uid}, FlagPatch{Seen: boolPtr(true)})
				return
			}
		}
	}
}

func boolPtr(b bool) *bool { return &b }

// parseRaw walks a raw RFC822 message with go-message/mail.
func parseRaw(raw []byte, a *store.Account, mailbox string, uid uint32) (*FullMessage, error) {
	entity, err := message.Read(bytes.NewReader(raw))
	if err != nil {
		return nil, errf(502, "邮件解析失败：%v", err)
	}
	mr := mail.NewReader(entity)

	full := &FullMessage{UID: uid, Headers: map[string]string{}}
	subject := mr.Header.Get("Subject")
	full.Subject = mimeDecode(subject)
	if full.Subject == "" {
		full.Subject = "(无主题)"
	}
	if froms, err := mr.Header.AddressList("From"); err == nil && len(froms) > 0 {
		full.From = Addr{Name: froms[0].Name, Email: froms[0].Address}
	}
	full.To = headerAddrs(mr.Header, "To")
	full.Cc = headerAddrs(mr.Header, "Cc")
	if date, err := mr.Header.Date(); err == nil {
		full.Date = date.Format(time.RFC3339)
	}
	for _, h := range []string{"Message-Id", "In-Reply-To", "References", "Date"} {
		if v := mr.Header.Get(h); v != "" {
			full.Headers[strings.ToLower(h)] = v
		}
	}

	var textPart, htmlPart string
	var attachments []AttachmentMeta
	attDir := attDir(a.ID, mailbox, uid)

	partIdx := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		disposition, dparams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		ctype, cparams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if ctype == "" {
			ctype = "text/plain"
		}
		filename := dparams["filename"]
		if filename == "" {
			filename = cparams["name"]
		}
		isAttachment := strings.EqualFold(disposition, "attachment") || filename != ""
		if !isAttachment && (ctype == "text/plain" || ctype == "text/html") {
			body, _ := io.ReadAll(io.LimitReader(part.Body, 5<<20))
			if ctype == "text/plain" && textPart == "" {
				textPart = mimeDecode(string(body))
			}
			if ctype == "text/html" && htmlPart == "" {
				htmlPart = string(body)
			}
			continue
		}
		// attachment-ish part: persist to disk
		body, _ := io.ReadAll(io.LimitReader(part.Body, 30<<20))
		if filename == "" {
			filename = "attachment-" + itoa(partIdx+1)
		}
		meta := AttachmentMeta{
			ID:          itoa(int(uid)) + "-" + itoa(partIdx),
			Filename:    filename,
			Size:        int64(len(body)),
			ContentType: ctype,
			Inline:      strings.EqualFold(disposition, "inline"),
			ContentID:   strings.Trim(part.Header.Get("Content-Id"), "<> "),
		}
		if len(body) > 0 {
			safe := sanitizeFilename(filename)
			file := filepath.Join(attDir, itoa(partIdx)+"-"+safe)
			if os.WriteFile(file, body, 0o600) == nil {
				rel, _ := filepath.Rel(store.CacheDir, file)
				meta.File = filepath.ToSlash(rel)
			}
		}
		attachments = append(attachments, meta)
		partIdx++
	}

	full.HTML = htmlPart
	full.Text = textPart
	snippetSrc := textPart
	if snippetSrc == "" && htmlPart != "" {
		snippetSrc = stripHTML(htmlPart)
	}
	full.Snippet = wsRe.ReplaceAllString(snippetSrc, " ")
	if len(full.Snippet) > 160 {
		full.Snippet = full.Snippet[:160]
	}
	full.Snippet = strings.TrimSpace(full.Snippet)
	full.Attachments = attachments
	if full.From.Email == "" && full.From.Name == "" {
		full.From = Addr{Name: "未知发件人"}
	}
	return full, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func headerAddrs(h mail.Header, key string) []Addr {
	list, err := h.AddressList(key)
	if err != nil {
		return nil
	}
	out := make([]Addr, 0, len(list))
	for _, a := range list {
		out = append(out, Addr{Name: a.Name, Email: a.Address})
	}
	return out
}

var mimeWordDec = &mime.WordDecoder{CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
	return input, nil
}}

// mimeDecode decodes RFC 2047 encoded words when present.
func mimeDecode(s string) string {
	if !strings.Contains(s, "=?") {
		return s
	}
	if dec, err := mimeWordDec.DecodeHeader(s); err == nil {
		return dec
	}
	return s
}

var styleTagRe = regexp.MustCompile(`(?s)<style.*?</style>|<[^>]+>`)

func stripHTML(s string) string {
	s = styleTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(name)
}

// ResolveAttachment finds an attachment file by its id.
func ResolveAttachment(a *store.Account, mailbox string, uid uint32, attID string) (filePath, filename, contentType string, err error) {
	var full *FullMessage
	if cached := loadMsg(a.ID, mailbox, uid); cached != nil {
		full = cached
	} else if fetched, e := GetMessage(a, mailbox, uid, false); e == nil {
		full = fetched
	}
	if full == nil {
		return "", "", "", errf(404, "附件不存在")
	}
	for _, att := range full.Attachments {
		if att.ID != attID {
			continue
		}
		if att.File != "" {
			p := filepath.Join(store.CacheDir, filepath.FromSlash(att.File))
			if _, serr := os.Stat(p); serr == nil {
				return p, att.Filename, att.ContentType, nil
			}
		}
		// re-fetch and re-extract
		if refetched, e := fetchForce(a, mailbox, uid); e == nil {
			for _, att2 := range refetched.Attachments {
				if att2.ID == attID && att2.File != "" {
					p := filepath.Join(store.CacheDir, filepath.FromSlash(att2.File))
					if _, serr := os.Stat(p); serr == nil {
						return p, att2.Filename, att2.ContentType, nil
					}
				}
			}
		}
	}
	return "", "", "", errf(404, "附件不存在")
}

// fetchForce re-parses a message ignoring the on-disk cache.
func fetchForce(a *store.Account, mailbox string, uid uint32) (*FullMessage, error) {
	_ = os.Remove(msgFile(a.ID, mailbox, uid))
	return GetMessage(a, mailbox, uid, false)
}
