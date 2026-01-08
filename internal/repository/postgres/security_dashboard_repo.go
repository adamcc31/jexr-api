package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SecurityDashboardRepository implements the security dashboard data access
type SecurityDashboardRepository struct {
	db *pgxpool.Pool
}

// NewSecurityDashboardRepository creates a new security dashboard repository
func NewSecurityDashboardRepository(db *pgxpool.Pool) *SecurityDashboardRepository {
	return &SecurityDashboardRepository{db: db}
}

// GetStats returns aggregated dashboard statistics
func (r *SecurityDashboardRepository) GetStats(ctx context.Context) (*domain.SecurityDashboardStats, error) {
	stats := &domain.SecurityDashboardStats{
		EventsBySeverity: make(map[string]int64),
		EventsByType:     make(map[string]int64),
	}

	// Total events
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM security_events`).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// Events by severity (last 7 days)
	severityQuery := `
		SELECT COALESCE(severity::text, 'UNKNOWN'), COUNT(*) 
		FROM security_events 
		WHERE created_at > NOW() - INTERVAL '7 days'
		GROUP BY severity
	`
	rows, err := r.db.Query(ctx, severityQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query severity stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int64
		if err := rows.Scan(&severity, &count); err == nil {
			stats.EventsBySeverity[severity] = count
		}
	}

	// Events by type (last 7 days)
	typeQuery := `
		SELECT event_type, COUNT(*) 
		FROM security_events 
		WHERE created_at > NOW() - INTERVAL '7 days'
		GROUP BY event_type
		ORDER BY COUNT(*) DESC
		LIMIT 20
	`
	rows, err = r.db.Query(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err == nil {
			stats.EventsByType[eventType] = count
		}
	}

	// Failed logins in last 24h
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM security_events 
		WHERE event_type = 'login_failed' AND created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.FailedLogins24h)
	if err != nil {
		stats.FailedLogins24h = 0
	}

	// Blocked attempts in last 24h
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM security_events 
		WHERE event_type = 'login_blocked' AND created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.BlockedAttempts24h)
	if err != nil {
		stats.BlockedAttempts24h = 0
	}

	// Critical events in last 24h
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM security_events 
		WHERE severity = 'CRITICAL' AND created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.CriticalEvents24h)
	if err != nil {
		stats.CriticalEvents24h = 0
	}

	// Active break-glass sessions
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM break_glass_sessions 
		WHERE expires_at > NOW() AND revoked_at IS NULL
	`).Scan(&stats.ActiveBreakGlass)
	if err != nil {
		stats.ActiveBreakGlass = 0
	}

	// Top IPs
	topIPQuery := `
		SELECT ip_address::text, COUNT(*) as event_count,
		       SUM(CASE WHEN event_type = 'login_failed' THEN 1 ELSE 0 END) as failed_logins,
		       MAX(created_at) as last_seen,
		       MAX(severity::text) as highest_severity
		FROM security_events
		WHERE ip_address IS NOT NULL AND created_at > NOW() - INTERVAL '24 hours'
		GROUP BY ip_address
		ORDER BY event_count DESC
		LIMIT 10
	`
	rows, err = r.db.Query(ctx, topIPQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ip domain.IPSummary
			var lastSeen time.Time
			if err := rows.Scan(&ip.IP, &ip.EventCount, &ip.FailedLogins, &lastSeen, &ip.HighestSeverity); err == nil {
				ip.LastSeen = lastSeen.Format(time.RFC3339)
				stats.TopIPs = append(stats.TopIPs, ip)
			}
		}
	}

	// Get last anchor date
	err = r.db.QueryRow(ctx, `
		SELECT anchor_date FROM hash_anchors ORDER BY anchor_date DESC LIMIT 1
	`).Scan(&stats.LastAnchorDate)
	if err != nil {
		stats.LastAnchorDate = nil
	}

	// Determine integrity status
	var chainBreaks int64
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM hash_anchors WHERE verification_status = 'failed'
	`).Scan(&chainBreaks)
	if chainBreaks > 0 {
		stats.IntegrityStatus = "compromised"
	} else if stats.LastAnchorDate == nil {
		stats.IntegrityStatus = "degraded"
	} else {
		stats.IntegrityStatus = "intact"
	}

	return stats, nil
}

// ListEvents returns filtered security events
func (r *SecurityDashboardRepository) ListEvents(ctx context.Context, filter domain.SecurityEventFilter) ([]domain.SecurityEventView, int64, error) {
	baseQuery := `
		SELECT id, created_at, event_type, 
		       COALESCE(severity::text, 'UNKNOWN'), 
		       COALESCE(subject_type, ''), 
		       COALESCE(subject_value, ''),
		       COALESCE(ip_address::text, ''),
		       COALESCE(user_agent, ''),
		       COALESCE(request_id, ''),
		       COALESCE(details, '{}'::jsonb)
		FROM security_events
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM security_events WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Build WHERE clauses
	if filter.StartTime != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.StartTime)
		argIndex++
	}
	if filter.EndTime != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.EndTime)
		argIndex++
	}
	if len(filter.EventTypes) > 0 {
		baseQuery += fmt.Sprintf(" AND event_type = ANY($%d)", argIndex)
		countQuery += fmt.Sprintf(" AND event_type = ANY($%d)", argIndex)
		args = append(args, filter.EventTypes)
		argIndex++
	}
	if len(filter.Severities) > 0 {
		baseQuery += fmt.Sprintf(" AND severity::text = ANY($%d)", argIndex)
		countQuery += fmt.Sprintf(" AND severity::text = ANY($%d)", argIndex)
		args = append(args, filter.Severities)
		argIndex++
	}
	if filter.SearchIP != "" {
		baseQuery += fmt.Sprintf(" AND ip_address::text LIKE $%d", argIndex)
		countQuery += fmt.Sprintf(" AND ip_address::text LIKE $%d", argIndex)
		args = append(args, filter.SearchIP+"%")
		argIndex++
	}
	if filter.SearchUser != "" {
		baseQuery += fmt.Sprintf(" AND subject_value ILIKE $%d", argIndex)
		countQuery += fmt.Sprintf(" AND subject_value ILIKE $%d", argIndex)
		args = append(args, "%"+filter.SearchUser+"%")
		argIndex++
	}

	// Get total count
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	// Execute query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []domain.SecurityEventView
	for rows.Next() {
		var e domain.SecurityEventView
		var detailsJSON []byte
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &e.EventType, &e.Severity,
			&e.SubjectType, &e.SubjectValue, &e.IP, &e.UserAgent,
			&e.RequestID, &detailsJSON,
		); err != nil {
			continue
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &e.Details)
		}
		events = append(events, e)
	}

	return events, total, nil
}

// GetAuthFailureHeatmap returns time-bucketed auth failure counts
func (r *SecurityDashboardRepository) GetAuthFailureHeatmap(ctx context.Context, startTime, endTime time.Time, bucketSize string) (*domain.HeatmapData, error) {
	interval := "1 hour"
	if bucketSize == "day" {
		interval = "1 day"
	}

	query := fmt.Sprintf(`
		SELECT 
			date_trunc('%s', created_at) as bucket,
			COUNT(*) as count,
			SUM(CASE WHEN severity::text = 'WARN' THEN 1 ELSE 0 END) as warn_count,
			SUM(CASE WHEN severity::text = 'HIGH' THEN 1 ELSE 0 END) as high_count,
			SUM(CASE WHEN severity::text = 'CRITICAL' THEN 1 ELSE 0 END) as critical_count
		FROM security_events
		WHERE event_type IN ('login_failed', 'login_blocked')
		  AND created_at >= $1 AND created_at <= $2
		GROUP BY bucket
		ORDER BY bucket ASC
	`, bucketSize)

	rows, err := r.db.Query(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query heatmap: %w", err)
	}
	defer rows.Close()

	heatmap := &domain.HeatmapData{
		BucketSize: interval,
	}

	for rows.Next() {
		var bucket domain.HeatmapBucket
		var warnCount, highCount, criticalCount int64
		if err := rows.Scan(&bucket.Timestamp, &bucket.Count, &warnCount, &highCount, &criticalCount); err != nil {
			continue
		}
		bucket.BySeverity = map[string]int64{
			"WARN":     warnCount,
			"HIGH":     highCount,
			"CRITICAL": criticalCount,
		}
		if bucket.Count > heatmap.MaxCount {
			heatmap.MaxCount = bucket.Count
		}
		heatmap.Buckets = append(heatmap.Buckets, bucket)
	}

	return heatmap, nil
}

// GetPrivilegedActionTimeline returns admin/privileged actions
func (r *SecurityDashboardRepository) GetPrivilegedActionTimeline(ctx context.Context, limit, offset int) ([]domain.PrivilegedActionView, int64, error) {
	// First get total count
	var total int64
	countQuery := `
		SELECT COUNT(*) FROM security_events 
		WHERE event_type IN (
			'role_modified', 'user_created', 'user_deleted', 'user_disabled',
			'config_changed', 'data_export_approved', 'breakglass_activated', 'breakglass_revoked'
		)
	`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count actions: %w", err)
	}

	query := `
		SELECT id, created_at, event_type, 
		       COALESCE(subject_type, ''),
		       COALESCE(subject_value, ''),
		       COALESCE(details, '{}'::jsonb)
		FROM security_events 
		WHERE event_type IN (
			'role_modified', 'user_created', 'user_deleted', 'user_disabled',
			'config_changed', 'data_export_approved', 'breakglass_activated', 'breakglass_revoked'
		)
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query actions: %w", err)
	}
	defer rows.Close()

	var actions []domain.PrivilegedActionView
	for rows.Next() {
		var a domain.PrivilegedActionView
		var detailsJSON []byte
		if err := rows.Scan(&a.ID, &a.Timestamp, &a.ActionType, &a.TargetType, &a.TargetID, &detailsJSON); err != nil {
			continue
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &a.Details)
			if actor, ok := a.Details["actor_id"].(string); ok {
				a.ActorID = actor
			}
			if justification, ok := a.Details["justification"].(string); ok {
				a.Justification = justification
			}
		}
		actions = append(actions, a)
	}

	return actions, total, nil
}

// CreateExportRequest creates a new export request
func (r *SecurityDashboardRepository) CreateExportRequest(ctx context.Context, userID string, req domain.CreateExportRequest) (*domain.ExportRequest, error) {
	query := `
		INSERT INTO export_requests (
			requested_by, filter_start_time, filter_end_time, 
			filter_event_types, filter_severity, filter_ip, filter_subject,
			justification
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	export := &domain.ExportRequest{
		RequestedBy:   userID,
		Filter:        req.Filter,
		Justification: req.Justification,
		Status:        "pending",
	}

	err := r.db.QueryRow(ctx, query,
		userID,
		req.Filter.StartTime,
		req.Filter.EndTime,
		req.Filter.EventTypes,
		req.Filter.Severities,
		req.Filter.SearchIP,
		req.Filter.SearchUser,
		req.Justification,
	).Scan(&export.ID, &export.RequestedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create export request: %w", err)
	}

	return export, nil
}

// GetExportRequest returns an export request by ID
func (r *SecurityDashboardRepository) GetExportRequest(ctx context.Context, exportID string) (*domain.ExportRequest, error) {
	query := `
		SELECT id, requested_by, created_at, justification, status,
		       approved_by, approved_at, download_count, download_expires_at
		FROM export_requests
		WHERE id = $1
	`

	export := &domain.ExportRequest{}
	err := r.db.QueryRow(ctx, query, exportID).Scan(
		&export.ID, &export.RequestedBy, &export.RequestedAt,
		&export.Justification, &export.Status,
		&export.ApprovedBy, &export.ApprovedAt,
		&export.DownloadCount, &export.DownloadExpires,
	)
	if err != nil {
		return nil, fmt.Errorf("export request not found: %w", err)
	}

	return export, nil
}

// ListExportRequests lists export requests by status
func (r *SecurityDashboardRepository) ListExportRequests(ctx context.Context, status string, limit, offset int) ([]domain.ExportRequest, int64, error) {
	var total int64
	countQuery := `SELECT COUNT(*) FROM export_requests WHERE status = $1`
	r.db.QueryRow(ctx, countQuery, status).Scan(&total)

	query := `
		SELECT id, requested_by, created_at, justification, status,
		       approved_by, approved_at, download_count
		FROM export_requests
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var exports []domain.ExportRequest
	for rows.Next() {
		var e domain.ExportRequest
		rows.Scan(&e.ID, &e.RequestedBy, &e.RequestedAt, &e.Justification,
			&e.Status, &e.ApprovedBy, &e.ApprovedAt, &e.DownloadCount)
		exports = append(exports, e)
	}

	return exports, total, nil
}

// ApproveExportRequest approves an export request
func (r *SecurityDashboardRepository) ApproveExportRequest(ctx context.Context, exportID, approverID string) error {
	query := `
		UPDATE export_requests 
		SET status = 'approved', 
		    approved_by = $2, 
		    approved_at = NOW(),
		    download_expires_at = NOW() + INTERVAL '24 hours'
		WHERE id = $1 AND status = 'pending'
	`
	result, err := r.db.Exec(ctx, query, exportID, approverID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("export request not found or already processed")
	}
	return nil
}

// RejectExportRequest rejects an export request
func (r *SecurityDashboardRepository) RejectExportRequest(ctx context.Context, exportID, approverID, reason string) error {
	query := `
		UPDATE export_requests 
		SET status = 'rejected', 
		    approved_by = $2, 
		    approved_at = NOW(),
		    rejection_reason = $3
		WHERE id = $1 AND status = 'pending'
	`
	result, err := r.db.Exec(ctx, query, exportID, approverID, reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("export request not found or already processed")
	}
	return nil
}

// IncrementDownloadCount increments the download count for an export
func (r *SecurityDashboardRepository) IncrementDownloadCount(ctx context.Context, exportID string) error {
	query := `UPDATE export_requests SET download_count = download_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, exportID)
	return err
}

// GetLastAnchor returns the most recent hash anchor
func (r *SecurityDashboardRepository) GetLastAnchor(ctx context.Context) (*security.HashAnchor, error) {
	query := `
		SELECT id, anchor_date, root_hash, event_count, first_event_id, last_event_id,
		       s3_key, verified_at, verification_status, created_at
		FROM hash_anchors
		ORDER BY anchor_date DESC
		LIMIT 1
	`
	anchor := &security.HashAnchor{}
	err := r.db.QueryRow(ctx, query).Scan(
		&anchor.ID, &anchor.AnchorDate, &anchor.RootHash, &anchor.EventCount,
		&anchor.FirstEventID, &anchor.LastEventID, &anchor.S3Key,
		&anchor.VerifiedAt, &anchor.VerificationStatus, &anchor.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return anchor, nil
}

// ListAnchors lists hash anchors with pagination
func (r *SecurityDashboardRepository) ListAnchors(ctx context.Context, limit, offset int) ([]security.HashAnchor, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM hash_anchors`).Scan(&total)

	query := `
		SELECT id, anchor_date, root_hash, event_count, verification_status, created_at
		FROM hash_anchors
		ORDER BY anchor_date DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var anchors []security.HashAnchor
	for rows.Next() {
		var a security.HashAnchor
		rows.Scan(&a.ID, &a.AnchorDate, &a.RootHash, &a.EventCount, &a.VerificationStatus, &a.CreatedAt)
		anchors = append(anchors, a)
	}

	return anchors, total, nil
}
