package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Subject constants for NATS topics.
const (
	SubjectSampleResultCreated  = "sample_result.created"
	SubjectSampleResultReviewed = "sample_result.reviewed"
	SubjectSampleResultApproved = "sample_result.approved"
)

// ChangeEvent represents a data change published to the event bus.
type ChangeEvent struct {
	ID             string          `json:"id"`
	Timestamp      time.Time       `json:"timestamp"`
	Subject        string          `json:"subject"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	TableName      string          `json:"table_name"`
	RecordID       uuid.UUID       `json:"record_id"`
	Action         string          `json:"action"`
	ChangedBy      uuid.UUID       `json:"changed_by"`
	OldValues      json.RawMessage `json:"old_values,omitempty"`
	NewValues      json.RawMessage `json:"new_values,omitempty"`
	Reason         string          `json:"reason,omitempty"`
}

// Bus wraps a NATS connection for publishing events.
type Bus struct {
	conn *nats.Conn
}

// Connect creates a new event bus connected to NATS.
func Connect(url string) (*Bus, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			slog.Info("nats reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Bus{conn: nc}, nil
}

// Publish sends a change event to the given NATS subject.
func (b *Bus) Publish(event ChangeEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := b.conn.Publish(event.Subject, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

// Subscribe registers a handler for events matching the subject pattern.
// The subject can include wildcards (e.g., "sample_result.*").
func (b *Bus) Subscribe(subject string, handler func(ChangeEvent)) (*nats.Subscription, error) {
	return b.conn.Subscribe(subject, func(msg *nats.Msg) {
		var event ChangeEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			slog.Error("unmarshal event", "error", err, "subject", msg.Subject)
			return
		}
		handler(event)
	})
}

// Close drains and closes the NATS connection.
func (b *Bus) Close() {
	b.conn.Drain()
}
