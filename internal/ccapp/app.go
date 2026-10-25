package ccapp

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appName is the name shown in the UI.
const appName = "ccmail"

// ApplyStoredTheme follows the saved appearance setting from startup.
func (a *App) ApplyStoredTheme() { applyThemePref(a.prefs.Theme) }

// BindWebView gives the reader the window's web view for HTML mail.
func (a *App) BindWebView(wv *mygo.WebView) {
	a.web = wv
	a.loadBody()
}

// CaptureWeb returns a PNG of the reader's web view, for screenshots.
func (a *App) CaptureWeb() ([]byte, error) {
	if a.web == nil {
		return nil, fmt.Errorf("no web view")
	}
	return a.web.CapturePage()
}

// Refresh loads accounts, folders and the first page of mail.
func (a *App) Refresh() {
	a.refreshAccounts()
	a.reloadList(true)
}

// paintWindow fills the window: a solid base, or one of the washes.
func (a *App) paintWindow(c *ui.Context, e ui.Element, p palette) {
	if g, ok := washGradient(a.prefs.Wash, p.Dark); ok {
		e.LinearGradient(g)
		return
	}
	e.Background(p.Window)
}

// shortcuts reads the app's keys, except while a sheet or dialog is open.
func (a *App) shortcuts(c *ui.Context) {
	if a.compose != nil || a.settings || a.wizard != nil {
		return
	}
	if c.Shortcut(0, ui.KeyJ) {
		a.moveSelection(1)
	}
	if c.Shortcut(0, ui.KeyK) {
		a.moveSelection(-1)
	}
	if c.Shortcut(0, ui.KeyN) {
		a.openCompose("new", nil)
	}
	if c.Shortcut(0, ui.KeyEnter) && a.listSel >= 0 {
		a.openRow(a.listSel)
	}
	if a.sel != nil {
		if c.Shortcut(0, ui.KeyR) {
			a.openCompose("reply", a.sel)
		}
		if c.Shortcut(0, ui.KeyA) {
			a.openCompose("replyAll", a.sel)
		}
		if c.Shortcut(0, ui.KeyF) {
			a.openCompose("forward", a.sel)
		}
		if c.Shortcut(0, ui.KeyS) {
			a.toggleStar(a.sel)
		}
		if c.Shortcut(0, ui.KeyE) {
			a.archiveMessage(a.sel)
		}
		if c.Shortcut(0, ui.KeyDelete) || c.Shortcut(0, ui.KeyBackspace) {
			a.deleteMessage(a.sel)
		}
	}
	if c.Shortcut(0, ui.KeyEscape) && a.sel != nil {
		a.sel = nil
		a.msg = nil
		a.msgErr = ""
		a.loadBody()
	}
	if c.Shortcut(ui.Ctrl, ui.KeyB) {
		a.sidebarOpen = !a.sidebarOpen
	}
	if c.Shortcut(0, ui.KeySlash) || c.Shortcut(ui.Cmd, ui.KeyK) {
		a.search.Focus()
	}
	if c.Shortcut(ui.Shift, ui.KeySlash) {
		a.openSettings(2)
	}
}

// deferAction runs fn after the frame's list is built, for menus that
// change the collection while their rows are being built.
func (a *App) deferAction(fn func()) { a.pending = append(a.pending, fn) }

// moveSelection walks the list, skipping date headings.
func (a *App) moveSelection(delta int) {
	row := a.listSel
	for {
		row += delta
		if row < 0 || row >= len(a.rows) {
			return
		}
		if a.rows[row].msg != nil {
			break
		}
	}
	a.listSel = row
	a.pendingScroll = row
	a.openMessage(a.rows[row].msg)
}
