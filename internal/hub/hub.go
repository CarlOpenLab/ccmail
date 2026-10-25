// Package hub broadcasts app events to every frontend: mygo windows via a
// typed event, and plain SSE for loopback/browser mode. Payload shape mirrors
// the Node SSE wire format: {"event": "...", "data": {...}}.
package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/egoist/mygo"
)

// Push is the single wire event carried to pages.
type Push struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

var pushEvent = mygo.NewEvent[Push]("push")

var (
	mu       sync.Mutex
	sseWriters = make(map[http.ResponseWriter]chan []byte, 8)
	nextID   int
)

// Broadcast sends an event to all connected surfaces.
func Broadcast(event string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = pushEvent.Broadcast(Push{Event: event, Data: raw})
	payload, _ := json.Marshal(Push{Event: event, Data: raw})

	mu.Lock()
	for w, ch := range sseWriters {
		select {
		case ch <- payload:
		default:
		}
		_ = w
	}
	mu.Unlock()
}

// ServeSSE implements GET /api/stream for loopback mode.
func ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, "retry: 3000\n\n")
	flusher.Flush()

	ch := make(chan []byte, 32)
	mu.Lock()
	sseWriters[w] = ch
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(sseWriters, w)
		mu.Unlock()
	}()

	hb := r.Context()
	for {
		select {
		case <-hb.Done():
			return
		case msg := <-ch:
			_, _ = w.Write(msg)
			fmt.Fprint(w, "\n\n")
			flusher.Flush()
		}
	}
}
