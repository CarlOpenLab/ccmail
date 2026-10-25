// Package mailx implements the IMAP access layer: pooled per-account
// connections, envelope sync with on-disk cache, full-message parsing
// and flag/move operations. Cache layout is stable across releases.
package mailx

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"ccmail/internal/store"
)

// Error is a user-facing mail failure with an HTTP status.
type Error struct {
	Msg    string
	Status int
}

func (e *Error) Error() string { return e.Msg }

func errf(status int, format string, args ...any) *Error {
	return &Error{Msg: fmt.Sprintf(format, args...), Status: status}
}

// ------------------------------- connections

type poolEntry struct {
	mu     sync.Mutex
	client *imapclient.Client
}

var (
	poolsMu sync.Mutex
	pools   = map[string]*poolEntry{}
)

func dial(server *store.Account) (*imapclient.Client, error) {
	if server.IMAP == nil {
		return nil, errf(400, "账号未配置 IMAP 服务器")
	}
	host := server.IMAP.Host
	port := server.IMAP.Port
	if port == 0 {
		port = 993
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	tlsCfg := &tls.Config{ServerName: host}

	var client *imapclient.Client
	var err error
	if server.IMAP.Secure {
		client, err = imapclient.DialTLS(addr, &imapclient.Options{TLSConfig: tlsCfg})
	} else {
		client, err = imapclient.DialStartTLS(addr, &imapclient.Options{TLSConfig: tlsCfg})
	}
	if err != nil {
		return nil, errf(502, "无法连接服务器：%s", explain(err))
	}
	if err := client.Login(server.Auth.User, server.Password()).Wait(); err != nil {
		client.Close()
		return nil, errf(400, "认证失败：请检查邮箱地址与密码/授权码是否正确")
	}
	return client, nil
}

func explain(err error) string {
	msg := err.Error()
	if netErr := (&net.OpError{}); errors.As(err, &netErr) || strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") {
		return "无法连接服务器：请检查服务器地址与端口，或本机网络"
	}
	if strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") {
		return "TLS/证书错误：" + truncateStr(msg, 100)
	}
	return truncateStr(msg, 140)
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// withClient runs fn on a kept-alive per-account connection, serialized.
func withClient(a *store.Account, fn func(c *imapclient.Client) error) error {
	if a.Type == "demo" {
		return errf(400, "演示账号没有 IMAP 连接")
	}
	poolsMu.Lock()
	p := pools[a.ID]
	if p == nil {
		p = &poolEntry{}
		pools[a.ID] = p
	}
	poolsMu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == nil {
		c, err := dial(a)
		if err != nil {
			return err
		}
		p.client = c
	}
	if err := fn(p.client); err != nil {
		// drop the connection; a later call will reconnect
		p.client.Close()
		p.client = nil
		return err
	}
	return nil
}

// DropConnection closes and forgets the pooled connection of an account.
func DropConnection(accountID string) {
	poolsMu.Lock()
	p := pools[accountID]
	delete(pools, accountID)
	poolsMu.Unlock()
	if p != nil {
		p.mu.Lock()
		if p.client != nil {
			p.client.Close()
		}
		p.mu.Unlock()
	}
}

// DialWatcher opens a dedicated connection for IDLE watching (outside the
// serialized op pool).
func DialWatcher(a *store.Account) (*imapclient.Client, error) {
	return dial(a)
}

// VerifyIMAP tests one IMAP login (used by the add-account wizard).
func VerifyIMAP(host string, port int, secure bool, user, password string) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	tlsCfg := &tls.Config{ServerName: host}
	var client *imapclient.Client
	var err error
	if secure {
		client, err = imapclient.DialTLS(addr, &imapclient.Options{TLSConfig: tlsCfg})
	} else {
		client, err = imapclient.DialStartTLS(addr, &imapclient.Options{TLSConfig: tlsCfg})
	}
	if err != nil {
		return errf(400, "无法连接服务器（%s）", explain(err))
	}
	defer client.Close()
	if err := client.Login(user, password).Wait(); err != nil {
		return errf(400, "认证失败：请检查邮箱地址与密码/授权码是否正确")
	}
	return nil
}

// MailboxInfo is one folder row for the sidebar.
type MailboxInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Delim    string `json:"delimiter"`
	Kind     string `json:"kind"`
	Parent   string `json:"parent"`
	Depth    int    `json:"depth"`
	Messages *int   `json:"messages"`
	Unread   *int   `json:"unread"`
}

var specialKinds = map[string]string{
	"\\Inbox": "inbox", "\\Sent": "sent", "\\Drafts": "drafts", "\\Trash": "trash",
	"\\Junk": "junk", "\\Spam": "junk", "\\All": "all", "\\Archive": "archive",
	"\\Flagged": "flagged", "\\Important": "important",
}

func kindFromAttrs(attrs []imap.MailboxAttr, name string) string {
	for _, a := range attrs {
		if k, ok := specialKinds[string(a)]; ok {
			return k
		}
	}
	l := strings.ToLower(name)
	switch {
	case l == "inbox" || strings.Contains(name, "收件箱"):
		return "inbox"
	case strings.HasPrefix(l, "sent") || strings.Contains(name, "已发送"):
		return "sent"
	case strings.HasPrefix(l, "draft") || strings.Contains(name, "草稿"):
		return "drafts"
	case strings.HasPrefix(l, "trash") || strings.Contains(name, "已删除") || strings.Contains(name, "废"):
		return "trash"
	case strings.HasPrefix(l, "junk") || l == "spam" || strings.Contains(name, "垃圾"):
		return "junk"
	case strings.HasPrefix(l, "archive") || strings.Contains(name, "已归档") || strings.Contains(name, "存档"):
		return "archive"
	}
	return "custom"
}

// ListMailboxes returns the folder tree with message/unread counts.
func ListMailboxes(a *store.Account) ([]MailboxInfo, error) {
	var out []MailboxInfo
	err := withClient(a, func(c *imapclient.Client) error {
		cmd := c.List("", "*", &imap.ListOptions{
			ReturnStatus: &imap.StatusOptions{NumMessages: true, NumUnseen: true},
		})
		datas, err := cmd.Collect()
		if err != nil {
			return errf(502, "读取邮箱列表失败：%s", explain(err))
		}
		for _, d := range datas {
			if d.Mailbox == "" {
				continue
			}
			nonSelectable := false
			for _, at := range d.Attrs {
				if string(at) == "\\Noselect" || string(at) == "\\NonExistent" {
					nonSelectable = true
				}
			}
			if nonSelectable {
				continue
			}
			delim := string(d.Delim)
			if delim == "" {
				delim = "/"
			}
			segs := strings.FieldsFunc(d.Mailbox, func(r rune) bool { return r == d.Delim })
			name := d.Mailbox
			if len(segs) > 0 {
				name = segs[len(segs)-1]
			}
			info := MailboxInfo{
				Path:  d.Mailbox,
				Name:  name,
				Delim: delim,
				Kind:  kindFromAttrs(d.Attrs, name),
				Depth: len(segs) - 1,
			}
			if len(segs) > 1 {
				info.Parent = strings.Join(segs[:len(segs)-1], delim)
			}
			if d.Status != nil {
				if d.Status.NumMessages != nil {
					n := int(*d.Status.NumMessages)
					info.Messages = &n
				}
				if d.Status.NumUnseen != nil {
					u := int(*d.Status.NumUnseen)
					info.Unread = &u
				}
			}
			out = append(out, info)
		}
		return nil
	})
	return out, err
}

// FindMailboxByKind finds the first folder of a canonical kind.
func FindMailboxByKind(a *store.Account, kind string) (*MailboxInfo, error) {
	boxes, err := ListMailboxes(a)
	if err != nil {
		return nil, err
	}
	for i := range boxes {
		if boxes[i].Kind == kind {
			return &boxes[i], nil
		}
	}
	return nil, nil
}
