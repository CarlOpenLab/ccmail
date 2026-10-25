package ccapp

import (
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// seed fills the app with the same sample data the -demo flag installs.
func seed(a *App) { a.LoadDemoData() }

func TestRenderScreens(t *testing.T) {
	out := os.Getenv("CCMAIL_SHOTS")
	if out == "" {
		out = "/tmp/ccmail-shots"
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	shot := func(name string, width, height int, dark bool, mut func(a *App)) {
		a := New()
		seed(a)
		if mut != nil {
			mut(a)
		}
		tt := ui.NewTester(a.View, width, height)
		tt.SetDark(dark)
		tt.SetTitleBar(ui.TitleBar{Height: 28, Left: 78, Right: 12})
		tt.Frame()
		img := tt.Image()
		f, err := os.Create(out + "/" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}

	shot("list-light", 1280, 800, false, nil)
	shot("list-dark", 1280, 800, true, nil)
	shot("compose", 1280, 800, false, func(a *App) {
		a.openCompose("reply", a.sel)
	})
	shot("settings", 1280, 800, false, func(a *App) {
		a.openSettings(0)
	})
	shot("settings-appearance", 1280, 800, false, func(a *App) {
		a.openSettings(1)
	})
	shot("settings-appearance-dark", 1280, 800, true, func(a *App) {
		a.openSettings(1)
	})
	shot("wizard", 1280, 800, false, func(a *App) {
		a.openWizard()
	})
	shot("sidebar-rail", 1280, 800, false, func(a *App) {
		a.prefs.SidebarShut = true
	})
	shot("moss-dark", 1280, 800, true, func(a *App) {
		a.prefs.Accent = "moss"
	})
	shot("narrow", 900, 560, false, nil)
	shot("phone-list", 820, 640, false, func(a *App) {
		a.sel = nil
		a.msg = nil
	})
	shot("phone-sidebar", 820, 640, true, func(a *App) {
		a.sel = nil
		a.msg = nil
		a.sidebarOpen = true
	})
	shot("empty", 1280, 800, false, func(a *App) {
		a.accounts = nil
		a.msgs = nil
		a.rows = nil
		a.sel = nil
		a.msg = nil
	})
}
