package ccapp

import (
	"testing"

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
	if !tt.Focused("搜索邮件") {
		t.Fatal("search field did not take the focus")
	}
	if a.query != "" {
		t.Fatalf("query = %q", a.query)
	}
}

func TestComposeOpensAndRequiresRecipient(t *testing.T) {
	a, tt := newTestApp(t)
	if err := tt.Click("写邮件"); err != nil {
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
	if !tt.HasText("请填写收件人") {
		t.Fatal("error not shown in the sheet")
	}
}

func TestSettingsAndWizardOpen(t *testing.T) {
	a, tt := newTestApp(t)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click settings: %v", err)
	}
	tt.Frame()
	if !a.settings {
		t.Fatal("settings did not open")
	}
	if !tt.HasText("已挂载的账号") {
		t.Fatal("accounts tab not shown")
	}
	if err := tt.Click("添加账号"); err != nil {
		t.Fatalf("click add account: %v", err)
	}
	tt.Frame()
	if a.wizard == nil {
		t.Fatal("wizard did not open")
	}
	if !tt.HasText("选择邮件服务商") {
		t.Fatal("provider step not shown")
	}
}
