// Package prefs stores small user preferences (data/prefs.json) shared by
// the Go backend (e.g. desktop notifications) and the frontend.
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"glossmail/internal/store"
)

var (
	mu   sync.Mutex
	data map[string]string
)

func file() string { return filepath.Join(store.DataDir, "prefs.json") }

func load() {
	if data != nil {
		return
	}
	data = map[string]string{}
	if raw, err := os.ReadFile(file()); err == nil {
		_ = json.Unmarshal(raw, &data)
	}
}

// All returns a copy of every preference.
func All() map[string]string {
	mu.Lock()
	defer mu.Unlock()
	load()
	out := make(map[string]string, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out
}

// SetMany merges values and persists.
func SetMany(values map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	load()
	for k, v := range values {
		if k == "" {
			continue
		}
		data[k] = v
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	tmp := file() + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, file())
	}
}

// Get returns one preference or the default.
func Get(key, def string) string {
	mu.Lock()
	defer mu.Unlock()
	load()
	if v, ok := data[key]; ok && v != "" {
		return v
	}
	return def
}

// NotificationsEnabled reports whether desktop notifications are on.
func NotificationsEnabled() bool { return Get("notifications", "on") != "off" }
