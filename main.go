// Gloss Mail desktop — a mygo rewrite: one Go binary serving the frontend
// from an embedded FS, the API over a custom "api" protocol, live updates
// over a mygo event, in a frameless window with page-drawn title bar.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"os"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"

	"glossmail/internal/api"
	"glossmail/internal/icon"
	"glossmail/internal/store"
	"glossmail/internal/watchers"
)

//go:embed all:public
var frontendFiles embed.FS

// Desktop is the small service bound to the page runtime.
type Desktop struct{}

// APIBase returns the absolute origin the frontend should prefix API
// requests with when running inside a mygo window.
func (Desktop) APIBase() string {
	if runtime.GOOS == "windows" {
		return "http://api.localhost"
	}
	return "api://localhost"
}

// Version reports the app version.
func (Desktop) Version() string { return "2.1.0" }

// Diag logs a diagnostic line from the page (used to verify the window's
// runtime wiring during development).
func (Desktop) Diag(msg string) {
	log.Printf("[window-diag] %s", msg)
}

var mainWindow *mygo.Window

func main() {
	serve := flag.String("serve", "", "also serve the app on this loopback address (e.g. 127.0.0.1:3731) for browser/debug mode")
	flag.Parse()

	store.EnsureDirs(".")
	store.Load()
	// remove leftover artifacts of the removed built-in demo account
	_ = os.Remove(filepath.Join(store.DataDir, "demo-state.json"))
	_ = os.RemoveAll(store.DemoFiles)

	mux := api.New()

	front, err := fs.Sub(frontendFiles, "public")
	if err != nil {
		log.Fatal(err)
	}
	mygo.SetFrontend(front)
	if err := mygo.Protocol.Handle("api", mux); err != nil {
		log.Fatalf("register api protocol: %v", err)
	}
	mygo.Bind(Desktop{})

	watchers.InitAll()

	if *serve != "" {
		go func() {
			// loopback mode serves BOTH the API and the embedded frontend
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"接口不存在"}`))
					return
				}
				p := strings.TrimPrefix(r.URL.Path, "/")
				if p == "" {
					p = "index.html"
				}
				data, err := frontendFiles.ReadFile("public/" + p)
				if err != nil {
					data, err = frontendFiles.ReadFile("public/index.html")
					if err != nil {
						http.NotFound(w, r)
						return
					}
					p = "index.html"
				}
				ctype := mime.TypeByExtension(filepath.Ext(p))
				if ctype == "" {
					ctype = "application/octet-stream"
				}
				w.Header().Set("Content-Type", ctype)
				_, _ = w.Write(data)
			})
			if err := http.ListenAndServe(*serve, mux); err != nil {
				log.Printf("loopback server: %v", err)
			}
		}()
	}

	appIcon := icon.Generate(256)
	trayIcon := icon.Generate(32)

	mygo.App.WhenReady(func() {
		mainWindow = mygo.NewWindow(mygo.WindowOptions{
			Title:     "Gloss Mail",
			URL:       "/",
			Width:     1280,
			Height:    840,
			MinWidth:  960,
			MinHeight: 620,
			Frameless: true,
			StateKey:  "main-window",
			BackgroundColor: "light-dark(#eceef4, #141418)",
		})
		if err := mainWindow.SetIcon(appIcon); err != nil {
			log.Printf("set window icon: %v", err)
		}
		setupTray(trayIcon)
	})

	if err := mygo.App.Run(); err != nil {
		fmt.Println("run:", err)
	}
}

func setupTray(trayIcon []byte) {
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示 Gloss Mail", Click: func(item *mygo.MenuItem, win *mygo.Window) {
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
		ID:     "gloss-main",
		Icon:   trayIcon,
		ToolTip: "Gloss Mail",
		Menu:   menu,
	}); err != nil {
		log.Printf("tray: %v", err)
	}
}
