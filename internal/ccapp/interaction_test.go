package ccapp

import (
	"testing"
	"time"

	"ccmail/locales"
	"github.com/egoist/mygo/ui"
)

func newTestApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a := New()
	seed(a)
	a.sel = nil
	a.msg = nil
	a.listSel = -1
	tt := ui.NewTester(a.View, 1280, 800)
	tt.SetTitleBar(ui.TitleBar{Height: 28, Left: 78, Right: 12})
	tt.Frame()
	return a, tt
}

func TestClickOpensMessage(t *testing.T) {
	a, tt := newTestApp(t)
	if err := tt.Click("Bob Smith"); err != nil {
		t.Fatalf("click row: %v", err)
	}
	tt.Frame()
	if a.sel == nil {
		t.Fatal("no message selected")
	}
	if got := a.sel.Env.Subject; got != "Weekend hiking plan" {
		t.Fatalf("opened %q", got)
	}
	if !tt.HasText("Weekend hiking plan") {
		t.Fatal("reader did not show the subject")
	}
}

func TestKeyboardMovesSelection(t *testing.T) {
	a, tt := newTestApp(t)
	tt.Key(0, ui.KeyJ)
	tt.Frame()
	if a.sel == nil || a.sel.Env.From.Name != "张伟" {
		t.Fatalf("J selected %+v", a.sel)
	}
	tt.Key(0, ui.KeyJ)
	tt.Frame()
	if a.sel == nil || a.sel.Env.From.Name != "Alice Chen" {
		t.Fatalf("second J selected %+v", a.sel)
	}
	tt.Key(0, ui.KeyK)
	tt.Frame()
	if a.sel == nil || a.sel.Env.From.Name != "张伟" {
		t.Fatalf("K selected %+v", a.sel)
	}
}

func TestSearchShortcutFocusesField(t *testing.T) {
	a, tt := newTestApp(t)
	tt.Key(0, ui.KeySlash)
	tt.Frame()
	if !tt.Focused(locales.TitlebarSearch()) {
		t.Fatal("search field did not take the focus")
	}
	if a.query != "" {
		t.Fatalf("query = %q", a.query)
	}
}

func TestComposeOpensAndRequiresRecipient(t *testing.T) {
	a, tt := newTestApp(t)
	if err := tt.Click(locales.TitlebarCompose()); err != nil {
		t.Fatalf("click compose: %v", err)
	}
	tt.Frame()
	if a.compose == nil {
		t.Fatal("compose sheet did not open")
	}
	a.sendCompose()
	if a.compose == nil || a.compose.Err == "" {
		t.Fatalf("expected a recipient error, compose=%+v", a.compose)
	}
	// the sheet stays open with the error shown
	tt.Frame()
	if !tt.HasText(locales.ComposeNeedRecipient()) {
		t.Fatal("error not shown in the sheet")
	}
}

func TestSettingsAndWizardOpen(t *testing.T) {
	a, tt := newTestApp(t)
	if err := tt.Click(locales.MiscSettings()); err != nil {
		t.Fatalf("click settings: %v", err)
	}
	tt.Frame()
	if !a.settings {
		t.Fatal("settings did not open")
	}
	if !tt.HasText(locales.SettingsMountedAccounts()) {
		t.Fatal("accounts tab not shown")
	}
	if err := tt.Click(locales.SettingsAddAccount()); err != nil {
		t.Fatalf("click add account: %v", err)
	}
	tt.Frame()
	if a.wizard == nil {
		t.Fatal("wizard did not open")
	}
	if !tt.HasText(locales.WizardPickProvider()) {
		t.Fatal("provider step not shown")
	}
}

func TestSidebarCollapseToggle(t *testing.T) {
	a, tt := newTestApp(t)
	before, ok := tt.Find(locales.ListAriaLabel())
	if !ok {
		t.Fatal("list not found")
	}
	if err := tt.Click(locales.TitlebarCollapseSidebar()); err != nil {
		t.Fatalf("click collapse: %v", err)
	}
	tt.Frame()
	if !a.prefs.SidebarShut {
		t.Fatal("sidebar did not collapse")
	}

	// the width eases to the rail rather than snapping
	tt.Advance(80 * time.Millisecond)
	tt.Frame()
	mid, ok := tt.Find(locales.ListAriaLabel())
	if !ok {
		t.Fatal("list vanished mid-transition")
	}
	if mid.X > before.X-20 || mid.X < 80 {
		t.Fatalf("card did not ease with the collapse: before=%.1f mid=%.1f", before.X, mid.X)
	}

	tt.Advance(300 * time.Millisecond)
	tt.Frame()
	end, _ := tt.Find(locales.ListAriaLabel())
	if end.X > 90 {
		t.Fatalf("rail width leaves the list at %.1f, want ~74", end.X)
	}
	if tt.HasText(locales.SidebarSmartViews()) {
		t.Fatal("collapsed rail still shows section labels")
	}

	if err := tt.Click(locales.TitlebarExpandSidebar()); err != nil {
		t.Fatalf("click expand: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	tt.Frame()
	back, _ := tt.Find(locales.ListAriaLabel())
	if a.prefs.SidebarShut || back.X < 200 {
		t.Fatalf("sidebar did not expand: shut=%v x=%.1f", a.prefs.SidebarShut, back.X)
	}
	if !tt.HasText(locales.SidebarSmartViews()) {
		t.Fatal("expanded sidebar lost its section labels")
	}
}
