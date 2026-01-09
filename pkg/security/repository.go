package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SecurityEventRepository handles persistence of security events to database
type SecurityEventRepository struct {
	db *pgxpool.Pool
}

// NewSecurityEventRepository creates a new repository for security events
func NewSecurityEventRepository(db *pgxpool.Pool) *SecurityEventRepository {
	return &SecurityEventRepository{db: db}
}

// PersistEvent inserts a security event into the database
func (r *SecurityEventRepository) PersistEvent(ctx context.Context, event SecurityEvent) error {
	query := `
		INSERT INTO security_events (
			event_type, service, environment, level,
			subject_type, subject_value, ip_address, user_agent,
			request_id, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	// Convert details to JSON
	var detailsJSON []byte
	if len(event.Details) > 0 {
		detailsJSON, _ = json.Marshal(event.Details)
	} else {
		detailsJSON = []byte("null") // Valid JSON null for empty details
	}

	// Handle IP address - use nil for empty strings
	var ipAddr interface{}
	if event.IP != "" {
		ipAddr = event.IP
	}

	_, err := r.db.Exec(ctx, query,
		string(event.Event),
		event.Service,
		event.Environment,
		event.Level,
		event.SubjectType,
		event.SubjectValue,
		ipAddr,
		event.UserAgent,
		event.RequestID,
		detailsJSON,
		event.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to persist security event: %w", err)
	}

	return nil
}

// CreatePersistFunc creates a persist function for the SecurityLogger
func (r *SecurityEventRepository) CreatePersistFunc() func(context.Context, SecurityEvent) error {
	return r.PersistEvent
}
