package ccapp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"ccmail/internal/mailx"
)

func itoa(n int) string { return strconv.Itoa(n) }

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// parseDate reads the message's date header, in the formats IMAP servers
// send and the cache stores.
func parseDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func fmtListDate(s string) string {
	t, ok := parseDate(s)
	if !ok {
		return ""
	}
	now := time.Now()
	days := int(startOfDay(now).Sub(startOfDay(t)).Hours() / 24)
	hm := t.Format("15:04")
	switch {
	case days == 0:
		return hm
	case days == 1:
		return "昨天"
	case days > 0 && days < 7:
		return [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[int(t.Weekday())]
	case t.Year() == now.Year():
		return fmt.Sprintf("%d月%d日", int(t.Month()), t.Day())
	default:
		return fmt.Sprintf("%d/%d/%d", t.Year(), int(t.Month()), t.Day())
	}
}

func dateGroupLabel(s string) string {
	t, ok := parseDate(s)
	if !ok {
		return "更早"
	}
	days := int(startOfDay(time.Now()).Sub(startOfDay(t)).Hours() / 24)
	switch {
	case days <= 0:
		return "今天"
	case days == 1:
		return "昨天"
	case days < 7:
		return "本周"
	case days < 30:
		return "本月"
	default:
		return "更早"
	}
}

func fmtFullDate(s string) string {
	t, ok := parseDate(s)
	if !ok {
		return ""
	}
	days := int(startOfDay(time.Now()).Sub(startOfDay(t)).Hours() / 24)
	hm := t.Format("15:04")
	switch {
	case days == 0:
		return "今天 " + hm
	case days == 1:
		return "昨天 " + hm
	default:
		return fmt.Sprintf("%d年%d月%d日 %s", t.Year(), int(t.Month()), t.Day(), hm)
	}
}

func fmtSize(bytes int64) string {
	switch {
	case bytes <= 0:
		return ""
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%d KB", bytes/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
	}
}

func displayName(a mailx.Addr) string {
	if a.Name != "" {
		return a.Name
	}
	if a.Email != "" {
		if i := strings.IndexByte(a.Email, '@'); i > 0 {
			return a.Email[:i]
		}
		return a.Email
	}
	return "未知发件人"
}

// initials gives one or two letters for an avatar: one for CJK names, the
// first letters of the first two words otherwise.
func initials(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "?"
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || (r >= 0x3040 && r <= 0x30ff) {
			return string(r)
		}
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '.' || r == '_' || r == '-' || r == '@'
	})
	if len(parts) >= 2 {
		return strings.ToUpper(string([]rune(parts[0])[:1]) + string([]rune(parts[1])[:1]))
	}
	r := []rune(s)
	if len(r) > 2 {
		r = r[:2]
	}
	return strings.ToUpper(string(r))
}

// addrLine joins addresses for a header line.
func addrLine(list []mailx.Addr) string {
	parts := make([]string, 0, len(list))
	for _, a := range list {
		if a.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", a.Name, a.Email))
		} else if a.Email != "" {
			parts = append(parts, a.Email)
		}
	}
	return strings.Join(parts, "、")
}
