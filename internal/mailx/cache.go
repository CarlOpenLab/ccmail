package mailx

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"ccmail/internal/store"
)

// Envelope is one message-list row.
type Envelope struct {
	UID            uint32        `json:"uid"`
	MessageID      string        `json:"messageId,omitempty"`
	Subject        string        `json:"subject"`
	From           Addr          `json:"from"`
	To             []Addr        `json:"to"`
	Date           string        `json:"date"`
	Flags          EnvelopeFlags `json:"flags"`
	Size           int64         `json:"size"`
	HasAttachments bool          `json:"hasAttachments"`
	Snippet        string        `json:"snippet"`
}

// EnvelopeFlags carries the IMAP flags we display.
type EnvelopeFlags struct {
	Seen     bool `json:"seen"`
	Flagged  bool `json:"flagged"`
	Answered bool `json:"answered"`
	Draft    bool `json:"draft"`
}

// Addr is a wire-format email address.
type Addr struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AttachmentMeta describes one parsed attachment.
type AttachmentMeta struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	Inline      bool   `json:"inline"`
	ContentID   string `json:"contentId,omitempty"`
	File        string `json:"file,omitempty"`
}

// FullMessage is the parsed body served to the reader pane.
type FullMessage struct {
	UID         uint32            `json:"uid"`
	Subject     string            `json:"subject"`
	From        Addr              `json:"from"`
	To          []Addr            `json:"to"`
	Cc          []Addr            `json:"cc"`
	Date        string            `json:"date"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	Snippet     string            `json:"snippet"`
	Attachments []AttachmentMeta  `json:"attachments"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// envDoc is the cached envelope list of one mailbox.
type envDoc struct {
	Path        string      `json:"path"`
	UIDValidity string      `json:"uidValidity"`
	UIDNext     uint32      `json:"uidNext"`
	Messages    []*Envelope `json:"messages"`
	SyncedAt    int64       `json:"syncedAt"`
}

func hashPath(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

func accountDir(accountID string) string {
	dir := filepath.Join(store.CacheDir, store.SafeDir(accountID))
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func envFile(accountID, mailbox string) string {
	return filepath.Join(accountDir(accountID), "env-"+hashPath(mailbox)+".json")
}

func msgFile(accountID, mailbox string, uid uint32) string {
	return filepath.Join(accountDir(accountID), "msg-"+hashPath(mailbox)+"-"+strconv.Itoa(int(uid))+".json")
}

func attDir(accountID, mailbox string, uid uint32) string {
	dir := filepath.Join(accountDir(accountID), "att-"+hashPath(mailbox)+"-"+strconv.Itoa(int(uid)))
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func readJSON(path string, v any) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

func writeJSON(path string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

func loadEnv(accountID, mailbox string) *envDoc {
	var doc envDoc
	if readJSON(envFile(accountID, mailbox), &doc) {
		return &doc
	}
	return nil
}

func saveEnv(accountID, mailbox string, doc *envDoc) {
	writeJSON(envFile(accountID, mailbox), doc)
}

func loadMsg(accountID, mailbox string, uid uint32) *FullMessage {
	var m FullMessage
	if readJSON(msgFile(accountID, mailbox, uid), &m) {
		return &m
	}
	return nil
}

func saveMsg(accountID, mailbox string, uid uint32, m *FullMessage) {
	writeJSON(msgFile(accountID, mailbox, uid), m)
}
