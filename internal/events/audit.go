package events

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditConsumer subscribes to all change events and writes them to the audit_log table.
type AuditConsumer struct {
	pool *pgxpool.Pool
	bus  *Bus
}

// NewAuditConsumer creates and starts the audit log consumer.
// It subscribes to all sample_result.* events.
func NewAuditConsumer(pool *pgxpool.Pool, bus *Bus) (*AuditConsumer, error) {
	c := &AuditConsumer{pool: pool, bus: bus}

	_, err := bus.Subscribe("sample_result.*", c.handle)
	if err != nil {
		return nil, err
	}

	slog.Info("audit consumer started", "subject", "sample_result.*")
	return c, nil
}

func (c *AuditConsumer) handle(event ChangeEvent) {
	id, err := uuid.NewV7()
	if err != nil {
		slog.Error("audit: generate uuid", "error", err)
		return
	}

	_, err = c.pool.Exec(context.Background(), `
		INSERT INTO audit_log (id, organization_id, table_name, record_id, action, old_values, new_values, changed_by, changed_at, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, event.OrganizationID, event.TableName, event.RecordID,
		event.Action, event.OldValues, event.NewValues,
		event.ChangedBy, event.Timestamp, event.Reason,
	)
	if err != nil {
		slog.Error("audit: insert", "error", err, "record_id", event.RecordID)
		return
	}

	slog.Info("audit recorded", "action", event.Action, "table", event.TableName, "record_id", event.RecordID)
}
