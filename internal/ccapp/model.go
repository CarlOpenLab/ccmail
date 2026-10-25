package ccapp

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ccmail/internal/mailx"
	"ccmail/internal/watchers"
	"ccmail/locales"
)

// Folder is one mailbox of an account.
type Folder struct {
	AccountID string
	Path      string
	Name      string
	Kind      string
	Unread    int
	Total     int
}

// AccountVM is an account as the UI shows it.
type AccountVM struct {
	ID       string
	Name     string
	Email    string
	Color    string
	Provider string
	Status   string
	Detail   string
	Unread   int
	Folders  []Folder
}

// ViewSel is the mailbox shown in the list: the unified inbox, starred mail,
// or one folder of one account.
type ViewSel struct {
	Kind      string // "unified" | "flagged" | "folder"
	AccountID string
	Path      string
	Name      string
}

// MsgRef is a message in the list, with the mailbox it came from.
type MsgRef struct {
	AccountID   string
	AccountName string
	Color       string
	Path        string
	Env         *mailx.Envelope
}

// Key identifies the message across reloads and list rebuilds.
func (m *MsgRef) Key() string {
	return m.AccountID + "\x00" + m.Path + "\x00" + itoa(int(m.Env.UID))
}

// Prefs mirrors the stored local preferences.
type Prefs struct {
	Theme         string // auto | light | dark
	Language      string // auto | en | zh-Hans
	Wash          string // plain | rex | mist | meadow | dusk
	SidebarShut   bool
	Density       string // cozy | compact
	MarkRead      bool
	Notifications bool
	AutoImages    bool
	QuoteReply    bool
}

// App is the whole UI state. It is read and written on the main thread only:
// goroutines hand their results back through update.
type App struct {
	win *mygo.Window
	web *mygo.WebView

	// data
	accounts    []*AccountVM
	statuses    map[string]watchers.Status
	view        ViewSel
	collapsed   map[string]bool
	sidebarSel  string
	sidebarOpen bool

	msgs          []*MsgRef
	rows          []rowVM
	unreadTotal   int
	blockedImages int
	total         int
	page          int
	more          bool
	loading       bool
	loadErr       string
	query         string
	filter        int
	filterAt      time.Time

	sel         *MsgRef
	msg         *mailx.FullMessage
	msgLoading  bool
	msgErr      string
	allowImages bool

	prefs       Prefs
	settings    bool
	settingsTab int
	editAccount string
	editName    string
	editColor   string
	clearAsk    bool
	compose     *Compose
	wizard      *Wizard
	accountsT   int
	aboutAt     time.Time

	list          ui.ListState
	listSel       int
	pendingScroll int
	listKey       string
	search        ui.Handle
	msgBody       ui.ScrollState
	sidebarW      float32
	listW         float32

	toasts       []toastMsg
	openSections map[string]*bool

	pending []func()

	// search input timing
	queryAt  time.Time
	queryDue bool
}

// New builds the app state with the stored preferences loaded.
func New() *App {
	a := &App{
		statuses:  map[string]watchers.Status{},
		collapsed: map[string]bool{},
		sidebarW:  248,
		listW:     392,
		filter:    0,
	}
	a.prefs = loadPrefs()
	a.listSel = -1
	a.view = ViewSel{Kind: "unified", Name: locales.MiscUnifiedInbox()}
	a.accounts = loadAccounts()
	a.sidebarSel = "unified"
	return a
}

// BindWindow gives the app its window; background work draws through it.
func (a *App) BindWindow(win *mygo.Window) {
	a.win = win
}

// update runs fn on the main thread and draws a frame. Without a window, as
// in tests, it runs fn at once.
func (a *App) update(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

func (a *App) after() {
	if a.win != nil {
		a.win.Invalidate()
	}
}
