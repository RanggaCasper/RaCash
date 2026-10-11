package event

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
)

const (
	StatusPending   = "PENDING"
	StatusPublished = "PUBLISHED"
	StatusFailed    = "FAILED"
)

var ErrNotInTx = errors.New("event: Publish harus dipanggil di dalam transaksi uow")

type outboxRecord struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Module        string
	EventType     string
	AggregateType string
	AggregateID   uuid.UUID `gorm:"type:uuid"`
	Payload       datatypes.JSON
	Status        string
	Attempts      int16
	LastError     *string
	AvailableAt   time.Time
	PublishedAt   *time.Time
	CreatedAt     time.Time
}

func (outboxRecord) TableName() string { return "outbox_events" }

// Outbox menulis event ke tabel outbox_events memakai transaksi aktif.
type Outbox struct{ db uow.DBProvider }

func NewOutbox(db uow.DBProvider) *Outbox { return &Outbox{db: db} }

// Publish menyimpan event di transaksi yang sama dengan perubahan data.
func (o *Outbox) Publish(ctx context.Context, events ...Event) error {
	if len(events) == 0 {
		return nil
	}
	if !uow.InTx(ctx) {
		return ErrNotInTx
	}
	recs := make([]outboxRecord, 0, len(events))
	for _, e := range events {
		envelope, err := encodeEnvelope(e)
		if err != nil {
			return err
		}
		recs = append(recs, outboxRecord{
			ID:            e.ID,
			Module:        e.Module,
			EventType:     e.Type,
			AggregateType: e.AggregateType,
			AggregateID:   e.AggregateID,
			Payload:       envelope,
			Status:        StatusPending,
			AvailableAt:   e.OccurredAt,
			CreatedAt:     e.OccurredAt,
		})
	}
	return o.db.DB(ctx).Create(&recs).Error
}
