// Package smtpx sends mail via net/smtp with implicit-TLS or STARTTLS,
// builds the MIME message, and appends a copy to the Sent folder.
package smtpx

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"

	"ccmail/internal/mailx"
	"ccmail/internal/store"
)

// Recipient is one envelope recipient.
type Recipient struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Attachment is one upload from the compose sheet.
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	ContentB64  string `json:"contentBase64"`
}

// Outgoing describes a composed message.
type Outgoing struct {
	To          []Recipient  `json:"to"`
	Cc          []Recipient  `json:"cc"`
	Bcc         []Recipient  `json:"bcc"`
	Subject     string       `json:"subject"`
	HTML        string       `json:"html"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

func randomBoundary() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("=_ccmail-%x", b)
}

func encodeHeader(s string) string {
	// RFC 2047 Q-encode when non-ASCII is present
	needs := false
	for _, r := range s {
		if r > 127 {
			needs = true
			break
		}
	}
	if !needs {
		return s
	}
	return mime.QEncoding.Encode("utf-8", s)
}

func formatAddr(r Recipient) string {
	if r.Name != "" {
		return fmt.Sprintf("%s <%s>", encodeHeader(r.Name), r.Email)
	}
	return "<" + r.Email + ">"
}

func fold76(s string) string {
	var out strings.Builder
	lineLen := 0
	for i := 0; i < len(s); i++ {
		out.WriteByte(s[i])
		lineLen++
		if lineLen >= 76 && (i+1)%4 == 0 {
			out.WriteString("\r\n")
			lineLen = 0
		}
	}
	return out.String()
}

// BuildMIME renders the outgoing message as raw RFC822 bytes.
func BuildMIME(fromName, fromEmail string, out *Outgoing) []byte {
	var b bytes.Buffer
	boundary := randomBoundary()
	altBoundary := randomBoundary()

	writeHeader := func(k, v string) {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString("\r\n")
	}

	msgID := func() string {
		r := make([]byte, 10)
		_, _ = rand.Read(r)
		return fmt.Sprintf("<ccmail-%x@%s>", r, strings.SplitN(fromEmail, "@", 2)[1])
	}()

	writeHeader("From", formatAddr(Recipient{Name: fromName, Email: fromEmail}))
	if len(out.To) > 0 {
		writeHeader("To", strings.Join(mapRecipients(out.To, formatAddr), ", "))
	}
	if len(out.Cc) > 0 {
		writeHeader("Cc", strings.Join(mapRecipients(out.Cc, formatAddr), ", "))
	}
	writeHeader("Subject", encodeHeader(orDefault(out.Subject, "(无主题)")))
	writeHeader("Date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	writeHeader("Message-ID", msgID)
	writeHeader("MIME-Version", "1.0")
	writeHeader("X-Mailer", "ccmail")

	if len(out.Attachments) == 0 && out.HTML != "" {
		writeHeader("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", altBoundary))
		b.WriteString("\r\n")
		writePart(&b, altBoundary, "text/plain; charset=utf-8", quotedPrintable(orDefault(out.Text, "")))
		writePart(&b, altBoundary, "text/html; charset=utf-8", quotedPrintable(out.HTML))
		b.WriteString("--" + altBoundary + "--\r\n")
		return b.Bytes()
	}

	writeHeader("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", boundary))
	b.WriteString("\r\n")

	// body part
	if out.HTML != "" || out.Text != "" {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n\r\n", altBoundary))
		writePart(&b, altBoundary, "text/plain; charset=utf-8", quotedPrintable(orDefault(out.Text, "")))
		writePart(&b, altBoundary, "text/html; charset=utf-8", quotedPrintable(out.HTML))
	}

	for _, att := range out.Attachments {
		data, err := base64.StdEncoding.DecodeString(att.ContentB64)
		if err != nil || len(data) == 0 {
			continue
		}
		ct := att.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString(fmt.Sprintf("Content-Type: %s\r\n", ct))
		b.WriteString("Content-Disposition: attachment;\r\n\tfilename*=UTF-8''" + escapeFilename(att.Filename) + "\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		b.WriteString(fold76(base64.StdEncoding.EncodeToString(data)))
		b.WriteString("\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")
	return b.Bytes()
}

func escapeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r > 127 || strings.ContainsRune("!#$&'()*+,/:;=?@[]% ", r) {
			b.WriteString(fmt.Sprintf("%%%02X", r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func mapRecipients(list []Recipient, f func(Recipient) string) []string {
	out := make([]string, 0, len(list))
	for _, r := range list {
		out = append(out, f(r))
	}
	return out
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func writePart(b *bytes.Buffer, boundary, contentType, body string) {
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: " + contentType + "\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\r\n") {
		b.WriteString("\r\n")
	}
}

// quotedPrintable encodes body text conservatively (UTF-8 safe).
func quotedPrintable(s string) string {
	var out strings.Builder
	lineLen := 0
	encodeByte := func(v byte) {
		out.WriteString(fmt.Sprintf("=%02X", v))
		lineLen += 3
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\r':
			out.WriteString("\r\n")
			lineLen = 0
		case c == '\n':
			if i == 0 || s[i-1] != '\r' {
				out.WriteString("\r\n")
			}
			lineLen = 0
		case c == ' ':
			// trailing space would be eaten — encode when line end is near
			if lineLen >= 72 {
				out.WriteString("=\r\n")
				lineLen = 0
			}
			encodeByte(c)
		case c >= 33 && c <= 126 && c != '=':
			if lineLen >= 75 {
				out.WriteString("=\r\n")
				lineLen = 0
			}
			out.WriteByte(c)
			lineLen++
		default:
			if lineLen >= 72 {
				out.WriteString("=\r\n")
				lineLen = 0
			}
			encodeByte(c)
		}
	}
	return out.String()
}

// SendResult reports a successful send.
type SendResult struct {
	OK        bool   `json:"ok"`
	Demo      bool   `json:"demo,omitempty"`
	MessageID string `json:"messageId"`
}

// Send submits the message through the account's SMTP server.
func Send(a *store.Account, out *Outgoing) (*SendResult, error) {
	if len(out.To) == 0 {
		return nil, &mailx.Error{Msg: "请填写收件人", Status: 400}
	}
	raw := BuildMIME(a.Name, a.Email, out)

	if a.SMTP == nil || a.SMTP.Host == "" {
		return nil, &mailx.Error{Msg: "账号未配置 SMTP 服务器", Status: 400}
	}
	password := a.Password()
	if password == "" {
		return nil, &mailx.Error{Msg: "账号凭据读取失败", Status: 400}
	}

	host := a.SMTP.Host
	port := a.SMTP.Port
	if port == 0 {
		port = 465
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	var client *smtp.Client
	var err error
	if a.SMTP.Secure {
		var conn *tls.Conn
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err == nil {
			client, err = smtp.NewClient(conn, host)
		}
	} else {
		client, err = smtp.Dial(addr)
		if err == nil {
			err = client.StartTLS(&tls.Config{ServerName: host})
		}
	}
	if err != nil {
		return nil, &mailx.Error{Msg: "发送失败：无法连接 SMTP（" + err.Error() + "）", Status: 502}
	}
	defer client.Close()

	auth := smtp.PlainAuth("", a.Auth.User, password, host)
	if err := client.Auth(auth); err != nil {
		return nil, &mailx.Error{Msg: "发送失败：SMTP 认证失败（" + err.Error() + "）", Status: 502}
	}
	if err := client.Mail(a.Email); err != nil {
		return nil, &mailx.Error{Msg: "发送失败：" + err.Error(), Status: 502}
	}
	all := append(append(append([]Recipient{}, out.To...), out.Cc...), out.Bcc...)
	for _, r := range all {
		if err := client.Rcpt(r.Email); err != nil {
			return nil, &mailx.Error{Msg: "发送失败：" + err.Error(), Status: 502}
		}
	}
	w, err := client.Data()
	if err != nil {
		return nil, &mailx.Error{Msg: "发送失败：" + err.Error(), Status: 502}
	}
	if _, err := w.Write(raw); err != nil {
		return nil, &mailx.Error{Msg: "发送失败：" + err.Error(), Status: 502}
	}
	if err := w.Close(); err != nil {
		return nil, &mailx.Error{Msg: "发送失败：" + err.Error(), Status: 502}
	}
	_ = client.Quit()

	// best-effort: copy to the Sent folder
	if sent, _ := mailx.FindMailboxByKind(a, "sent"); sent != nil {
		_ = mailx.AppendToMailbox(a, sent.Path, raw)
	}
	return &SendResult{OK: true}, nil
}

// Verify tests SMTP connectivity + credentials without sending mail
// (used by the add-account wizard).
func Verify(host string, port int, secure bool, user, password string) error {
	if port == 0 {
		port = 465
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	var client *smtp.Client
	var err error
	if secure {
		var conn *tls.Conn
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err == nil {
			client, err = smtp.NewClient(conn, host)
		}
	} else {
		client, err = smtp.Dial(addr)
		if err == nil {
			err = client.StartTLS(&tls.Config{ServerName: host})
		}
	}
	if err != nil {
		return fmt.Errorf("无法连接 SMTP（%v）", err)
	}
	defer client.Close()
	if err := client.Auth(smtp.PlainAuth("", user, password, host)); err != nil {
		return fmt.Errorf("SMTP 认证失败（%v）", err)
	}
	if err := client.Noop(); err != nil {
		return fmt.Errorf("SMTP 验证失败（%v）", err)
	}
	return nil
}
