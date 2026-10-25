package ccapp

import (
	"strings"
	"testing"

	"ccmail/internal/mailx"
)

func TestSanitizeEmailDropsActiveContent(t *testing.T) {
	raw := `<p onclick="x()">hello<script>alert(1)</script><a href="javascript:evil()">go</a>` +
		`<iframe src="https://x"></iframe><img src="https://tracker.example/p.gif" onerror="y()">`
	out, blocked, err := sanitizeEmail(raw, &mailx.FullMessage{}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"<script", "onclick", "javascript:", "<iframe", "onerror"} {
		if strings.Contains(strings.ToLower(out), bad) {
			t.Errorf("sanitized body still contains %q: %s", bad, out)
		}
	}
	if blocked != 1 {
		t.Errorf("blocked = %d, want 1", blocked)
	}
	if strings.Contains(out, "tracker.example") {
		t.Errorf("remote image source survived: %s", out)
	}
}

func TestSanitizeEmailAllowsRemoteImagesOnRequest(t *testing.T) {
	out, blocked, err := sanitizeEmail(`<img src="https://cdn.example/a.png" alt="a">`, &mailx.FullMessage{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if blocked != 0 || !strings.Contains(out, "https://cdn.example/a.png") {
		t.Fatalf("blocked=%d out=%s", blocked, out)
	}
}

func TestSanitizeEmailKeepsLinksSafe(t *testing.T) {
	out, _, err := sanitizeEmail(`<a href="https://example.com/x">x</a><a href="/rel">r</a>`, &mailx.FullMessage{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `rel="noreferrer noopener"`) || !strings.Contains(out, `target="_blank"`) {
		t.Errorf("links not hardened: %s", out)
	}
	if !strings.Contains(out, `href="#/rel"`) {
		t.Errorf("relative link not neutralized: %s", out)
	}
}
