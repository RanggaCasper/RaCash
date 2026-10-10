// Package event: domain event, outbox writer, registry subscriber, dan relay
// outbox → Asynq (ARCHITECTURE §2.5).
//
// Alur: service menulis perubahan data + Outbox.Publish dalam SATU transaksi
// (uow). Relay di worker membaca outbox_events PENDING, lalu meng-enqueue satu
// task per subscriber dengan TaskID unik (dedupe). Subscriber wajib idempoten
// berdasarkan Event.ID.
package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
)

// Event adalah domain event. Type mengikuti pola "<aggregate>.<past_tense>",
// mis. "order.succeeded", dan wajib terdaftar di katalog ARCHITECTURE §2.5.
type Event struct {
	ID            uuid.UUID       `json:"id"`
	Module        string          `json:"module"`
	Type          string          `json:"type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	Payload       json.RawMessage `json:"payload"`
	OccurredAt    time.Time       `json:"occurred_at"`
}

// New membuat event dengan payload yang di-marshal ke JSON.
func New(module, typ, aggregateType string, aggregateID uuid.UUID, payload any, at time.Time) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:            id.New(),
		Module:        module,
		Type:          typ,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Payload:       raw,
		OccurredAt:    at.UTC(),
	}, nil
}

// Decode mengurai payload ke tipe tujuan.
func (e Event) Decode(out any) error { return json.Unmarshal(e.Payload, out) }

// Handler memproses event. Error membuat task di-retry oleh Asynq.
type Handler func(ctx context.Context, e Event) error

// Bus adalah kontrak yang diterima modul di RegisterSubscribers.
type Bus interface {
	Subscribe(eventType, subscriber string, h Handler)
}

// Publisher adalah kontrak yang dipakai service untuk menerbitkan event.
// Implementasinya (Outbox) wajib dipanggil di dalam transaksi uow.
type Publisher interface {
	Publish(ctx context.Context, events ...Event) error
}
