// Package hub fans app events out to in-process listeners. The native UI
// subscribes once and redraws its window for each event; accounts, sync
// workers and the UI all run in the same process, so no wire format or
// transport is involved.
package hub

import (
	"encoding/json"
	"sync"
)

// Event is one notification, with its payload still encoded so that a
// subscriber decodes only the events it cares about.
type Event struct {
	Name string
	Data json.RawMessage
}

// Decode unmarshals the payload into v.
func (e Event) Decode(v any) error {
	if len(e.Data) == 0 {
		return nil
	}
	return json.Unmarshal(e.Data, v)
}

var (
	mu     sync.Mutex
	nextID int
	subs   = make(map[int]func(Event), 4)
)

// Subscribe registers fn for every event broadcast after it returns, and
// returns a function that unregisters it. fn must not block: it runs on the
// goroutine that broadcast the event.
func Subscribe(fn func(Event)) (unsubscribe func()) {
	mu.Lock()
	nextID++
	id := nextID
	subs[id] = fn
	mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			delete(subs, id)
			mu.Unlock()
		})
	}
}

// Broadcast sends an event to every listener. A payload that cannot be
// encoded is dropped: event data is always best-effort.
func Broadcast(name string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	ev := Event{Name: name, Data: raw}

	mu.Lock()
	listeners := make([]func(Event), 0, len(subs))
	for _, fn := range subs {
		listeners = append(listeners, fn)
	}
	mu.Unlock()

	for _, fn := range listeners {
		fn(ev)
	}
}
