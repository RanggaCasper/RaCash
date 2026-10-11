package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hibiken/asynq"
)

type subscription struct {
	name    string
	handler Handler
}

// Registry mengimplementasikan Bus: modul mendaftarkan subscriber di worker,
// lalu Mount memasang handler Asynq untuk tiap pasangan (event, subscriber).
type Registry struct {
	subs map[string][]subscription
}

func NewRegistry() *Registry { return &Registry{subs: map[string][]subscription{}} }

// Subscribe mendaftarkan handler. Nama subscriber unik per event, biasanya nama modul.
func (r *Registry) Subscribe(eventType, subscriber string, h Handler) {
	for _, s := range r.subs[eventType] {
		if s.name == subscriber {
			panic(fmt.Sprintf("event: subscriber %q untuk %q terdaftar dua kali", subscriber, eventType))
		}
	}
	r.subs[eventType] = append(r.subs[eventType], subscription{name: subscriber, handler: h})
}

// Subscribers mengembalikan nama subscriber suatu event (terurut, deterministik).
func (r *Registry) Subscribers(eventType string) []string {
	names := make([]string, 0, len(r.subs[eventType]))
	for _, s := range r.subs[eventType] {
		names = append(names, s.name)
	}
	sort.Strings(names)
	return names
}

// TaskType adalah tipe task Asynq untuk satu subscriber.
func TaskType(eventType, subscriber string) string {
	return "event:" + eventType + ":" + subscriber
}

// Mount memasang handler subscriber ke mux worker.
func (r *Registry) Mount(mux *asynq.ServeMux) {
	for typ, subs := range r.subs {
		for _, s := range subs {
			h := s.handler
			mux.HandleFunc(TaskType(typ, s.name), func(ctx context.Context, t *asynq.Task) error {
				var e Event
				if err := json.Unmarshal(t.Payload(), &e); err != nil {
					return fmt.Errorf("%w: payload event rusak: %v", asynq.SkipRetry, err)
				}
				return h(ctx, e)
			})
		}
	}
}

// encodeEnvelope menyimpan event utuh (metadata + payload) di kolom payload outbox,
// sehingga relay dapat meneruskannya apa adanya.
func encodeEnvelope(e Event) ([]byte, error) { return json.Marshal(e) }
