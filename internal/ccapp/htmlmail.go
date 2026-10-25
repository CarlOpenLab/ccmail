package ccapp

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"ccmail/internal/mailx"
)

// Elements that never belong in a message body.
var dropElements = map[atom.Atom]bool{
	atom.Script: true, atom.Iframe: true, atom.Object: true, atom.Embed: true,
	atom.Link: true, atom.Meta: true, atom.Base: true, atom.Form: true,
	atom.Frame: true, atom.Frameset: true, atom.Applet: true,
}

// blockElements are the ones a body must not keep inside another element.
var voidElements = map[atom.Atom]bool{
	atom.Area: true, atom.Base: true, atom.Br: true, atom.Col: true, atom.Embed: true,
	atom.Hr: true, atom.Img: true, atom.Input: true, atom.Link: true, atom.Meta: true,
	atom.Source: true, atom.Track: true, atom.Wbr: true,
}

// sanitizeEmail cleans a message body: it drops scripts, frames and forms,
// strips event handlers and dangerous URLs, resolves cid: images from the
// message's own attachments, and blocks remote images unless allowed.
func sanitizeEmail(raw string, m *mailx.FullMessage, allowRemote bool) (string, int, error) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", 0, err
	}
	body := findBody(doc)
	if body == nil {
		return "", 0, fmt.Errorf("邮件正文为空")
	}

	cid := cidIndex(m)

	blocked := 0
	var clean func(n *html.Node)
	clean = func(n *html.Node) {
		for child := n.FirstChild; child != nil; {
			next := child.NextSibling
			if child.Type == html.ElementNode && dropElements[child.DataAtom] {
				n.RemoveChild(child)
			} else {
				if child.Type == html.ElementNode {
					cleanElement(child, cid, allowRemote, &blocked)
				}
				clean(child)
			}
			child = next
		}
	}
	clean(body)

	var b strings.Builder
	for child := body.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&b, child); err != nil {
			return "", blocked, err
		}
	}
	return b.String(), blocked, nil
}

// cleanElement rewrites one element's attributes and remote images.
func cleanElement(el *html.Node, cid map[string]string, allowRemote bool, blocked *int) {
	attrs := el.Attr[:0]
	for _, at := range el.Attr {
		name := strings.ToLower(at.Key)
		val := strings.TrimSpace(at.Val)
		lower := strings.ToLower(val)
		switch {
		case strings.HasPrefix(name, "on"):
			continue
		case name == "srcdoc":
			continue
		case (name == "href" || name == "src" || name == "srcset") &&
			(strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:text/html")):
			continue
		}
		attrs = append(attrs, at)
	}
	el.Attr = attrs

	if el.DataAtom == atom.Img {
		src := attrValue(el, "src")
		if strings.HasPrefix(strings.ToLower(src), "cid:") {
			key := strings.Trim(strings.TrimSpace(src[4:]), "<>")
			if url, ok := cid[key]; ok {
				setAttr(el, "src", url)
				return
			}
		}
		if src == "" || strings.HasPrefix(strings.ToLower(src), "data:") {
			return
		}
		if allowRemote {
			return
		}
		(*blocked)++
		removeAttr(el, "src")
		removeAttr(el, "srcset")
		if attrValue(el, "width") == "" {
			setAttr(el, "width", "220")
		}
		if attrValue(el, "height") == "" {
			setAttr(el, "height", "90")
		}
		setAttr(el, "style", "width:220px;height:90px;background:rgba(120,120,140,0.14);border-radius:8px;display:block")
		setAttr(el, "alt", "已屏蔽远程图片")
	}

	if el.DataAtom == atom.A {
		href := attrValue(el, "href")
		if href != "" && !isAbsoluteURL(href) {
			setAttr(el, "href", "#"+href)
		}
		setAttr(el, "target", "_blank")
		setAttr(el, "rel", "noreferrer noopener")
	}
}

func isAbsoluteURL(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") ||
		strings.HasPrefix(l, "mailto:") || strings.HasPrefix(l, "#")
}

// cidIndex maps Content-ID values to data URLs of the message's inline parts.
func cidIndex(m *mailx.FullMessage) map[string]string {
	out := map[string]string{}
	if m == nil {
		return out
	}
	for _, att := range m.Attachments {
		id := strings.Trim(strings.TrimSpace(att.ContentID), "<>")
		if id == "" || att.File == "" || !strings.HasPrefix(att.ContentType, "image/") {
			continue
		}
		data, err := os.ReadFile(att.File)
		if err != nil || len(data) > 8<<20 {
			continue
		}
		out[id] = "data:" + att.ContentType + ";base64," + base64.StdEncoding.EncodeToString(data)
	}
	return out
}

func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == atom.Body {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBody(c); b != nil {
			return b
		}
	}
	return nil
}

func attrValue(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	for i := range n.Attr {
		if strings.EqualFold(n.Attr[i].Key, key) {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func removeAttr(n *html.Node, key string) {
	out := n.Attr[:0]
	for _, a := range n.Attr {
		if !strings.EqualFold(a.Key, key) {
			out = append(out, a)
		}
	}
	n.Attr = out
}

// renderMailHTML wraps a sanitized body in the reader's document: the app's
// typography, a readable measure, and the appearance's colors.
func renderMailHTML(m *mailx.FullMessage, allowRemote bool, accent ui.Color) (string, int, error) {
	body, blocked, err := sanitizeEmail(m.HTML, m, allowRemote)
	if err != nil {
		return "", blocked, err
	}
	return mailDocument(body, isDarkNow(), accent), blocked, nil
}

func mailDocument(body string, dark bool, accent ui.Color) string {
	fg, bg, quotes, codeBg := "#1d1d21", "#ffffff", "#67677a", "#f4f5f8"
	if dark {
		fg, bg, quotes, codeBg = "#ececf1", "#1d1d22", "#a0a0ac", "#26262d"
		accent = ui.Hex("#c9cbd2")
	}
	link := hexOf(accent)
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="color-scheme" content="` +
		map[bool]string{true: "dark", false: "light"}[dark] + `"><style>
:root { color-scheme: ` + map[bool]string{true: "dark", false: "light"}[dark] + `; }
html, body { margin: 0; padding: 0; background: ` + bg + `; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", "PingFang SC",
    "Hiragino Sans GB", "Microsoft YaHei UI", sans-serif;
  color: ` + fg + `; font-size: 14.5px; line-height: 1.75; word-break: break-word;
  -webkit-font-smoothing: antialiased;
}
.ccmail-pad { padding: 24px 30px 36px; max-width: 680px; margin: 0 auto; }
img { max-width: 100% !important; height: auto !important; }
table { max-width: 100% !important; }
a { color: ` + link + `; text-decoration: underline; text-underline-offset: 2px; }
blockquote { margin: 1em 0; padding: 2px 14px; border-left: 3px solid ` + quotes + `; color: ` + quotes + `; }
pre { overflow-x: auto; background: ` + codeBg + `; padding: 12px 14px; border-radius: 10px; font-size: 13px; }
</style></head><body><div class="ccmail-pad">` + body + `</div></body></html>`
}

// isDarkNow reports the appearance in force, for pages the app composes.
func isDarkNow() (dark bool) {
	defer func() {
		if recover() != nil {
			dark = false
		}
	}()
	return mygo.Theme.IsDark()
}

// emptyReaderHTML is the placeholder page for a reader with no HTML body.
func emptyReaderHTML() string {
	return `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;background:transparent}
</style></head><body></body></html>`
}
