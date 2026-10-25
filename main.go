// ccmail is a quiet, focused mail client for any IMAP/SMTP account, drawn
// with MyGo's native UI: no webview, no front-end build, one Go binary.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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
	demo := flag.Bool("demo", false, "preview the interface with sample mail; nothing is saved")
	capture := flag.String("capture", "", "write window/reader screenshots to this directory, then exit")
	flag.Parse()

	store.EnsureDirs(".")
	store.Load()
	// clear leftovers of the removed built-in demo account
	_ = os.Remove(filepath.Join(store.DataDir, "demo-state.json"))
	_ = os.RemoveAll(store.DemoFiles)

	app := ccapp.New()
	if *demo {
		app.LoadDemoData()
	}
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

		if !*demo {
			app.Refresh()
		}
		setupTray(icon.Generate(32))

		if *capture != "" {
			go func() {
				time.Sleep(2500 * time.Millisecond)
				if data, err := mainWindow.CapturePage(); err == nil {
					_ = os.WriteFile(filepath.Join(*capture, "window.png"), data, 0o644)
				}
				if data, err := app.CaptureWeb(); err == nil {
					_ = os.WriteFile(filepath.Join(*capture, "reader.png"), data, 0o644)
				}
				mygo.App.Exit(0)
			}()
		}
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
