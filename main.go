// ccmail is a quiet, focused mail client for any IMAP/SMTP account, drawn
// with MyGo's native UI: no webview, no front-end build, one Go binary.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"ccmail/internal/ccapp"
	"ccmail/internal/icon"
	"ccmail/internal/store"
	"ccmail/internal/version"
	"ccmail/internal/watchers"
)

var mainWindow *mygo.Window

func main() {
	flag.Parse()

	store.EnsureDirs(".")
	store.Load()
	// clear leftovers of the removed built-in demo account
	_ = os.Remove(filepath.Join(store.DataDir, "demo-state.json"))
	_ = os.RemoveAll(store.DemoFiles)

	app := ccapp.New()
	app.SubscribeToWorkers()
	app.ApplyStoredTheme()

	watchers.InitAll()

	mygo.App.WhenReady(func() {
		mainWindow = mygo.NewWindow(mygo.WindowOptions{
			Title:           version.Name,
			Width:           1280,
			Height:          840,
			MinWidth:        900,
			MinHeight:       560,
			TitleBarStyle:   mygo.TitleBarHidden,
			StateKey:        "main-window",
			BackgroundColor: "light-dark(#eef0f4, #0f1013)",
			Content:         ui.View(app.View),
		})
		app.BindWindow(mainWindow)
		if err := mainWindow.SetIcon(icon.Generate(256)); err != nil {
			log.Printf("set window icon: %v", err)
		}

		web, err := mainWindow.NewWebView(mygo.WebViewOptions{
			BackgroundColor: "light-dark(#ffffff, #17181c)",
		})
		if err != nil {
			log.Printf("new web view: %v", err)
		} else {
			app.BindWebView(web)
		}

		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleWindowMenu},
		}))

		app.Refresh()
		setupTray(icon.Generate(32))
	})

	if err := mygo.App.Run(); err != nil {
		fmt.Println("run:", err)
	}
}

func setupTray(trayIcon []byte) {
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示 ccmail", Click: func(item *mygo.MenuItem, win *mygo.Window) {
			if mainWindow != nil {
				mainWindow.Show()
			}
			mygo.App.Focus()
		}},
		mygo.Separator(),
		{Label: "退出", Click: func(item *mygo.MenuItem, win *mygo.Window) {
			mygo.App.Exit(0)
		}},
	})
	if _, err := mygo.NewTray(mygo.TrayOptions{
		ID:      "ccmail-main",
		Icon:    trayIcon,
		ToolTip: version.Name,
		Menu:    menu,
	}); err != nil {
		log.Printf("tray: %v", err)
	}
}
