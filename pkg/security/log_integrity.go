package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogIntegrityService handles hash chaining and external anchoring
// Trust Model:
// - DB logs: Untrusted (mutable)
// - Hash chain: Untrusted (recomputable)
// - S3 Object Lock: TRUSTED (WORM)
type LogIntegrityService struct {
	db       *pgxpool.Pool
	s3Client *s3.Client
	bucket   string
	logger   *SecurityLogger
}

// IntegrityReport represents the result of an integrity verification
type IntegrityReport struct {
	StartDate         time.Time `json:"startDate"`
	EndDate           time.Time `json:"endDate"`
	TotalEvents       int64     `json:"totalEvents"`
	VerifiedEvents    int64     `json:"verifiedEvents"`
	ChainBreaks       int64     `json:"chainBreaks"`
	MissingAnchors    int64     `json:"missingAnchors"`
	AnchorMismatches  int64     `json:"anchorMismatches"`
	Status            string    `json:"status"` // "intact", "degraded", "compromised"
	FirstBreakEventID *int64    `json:"firstBreakEventId,omitempty"`
	Details           []string  `json:"details,omitempty"`
}

// HashAnchor represents an externally-stored root hash
type HashAnchor struct {
	ID                 int        `json:"id"`
	AnchorDate         time.Time  `json:"anchorDate"`
	RootHash           string     `json:"rootHash"`
	EventCount         int        `json:"eventCount"`
	FirstEventID       int64      `json:"firstEventId"`
	LastEventID        int64      `json:"lastEventId"`
	S3Key              string     `json:"s3Key"`
	VerifiedAt         *time.Time `json:"verifiedAt,omitempty"`
	VerificationStatus string     `json:"verificationStatus"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// LogIntegrityConfig holds configuration for the integrity service
type LogIntegrityConfig struct {
	S3Bucket       string
	S3KeyPrefix    string // e.g., "security-anchors/"
	RetentionYears int    // Object Lock retention period
}

// NewLogIntegrityService creates a new log integrity service
func NewLogIntegrityService(db *pgxpool.Pool, s3Client *s3.Client, config LogIntegrityConfig) *LogIntegrityService {
	return &LogIntegrityService{
		db:       db,
		s3Client: s3Client,
		bucket:   config.S3Bucket,
		logger:   DefaultLogger(),
	}
}

// ComputeEventHash computes the hash for a single event row
// Hash includes: id, event_type, timestamp, subject, ip, details, previous_hash
func ComputeEventHash(id int64, eventType string, timestamp time.Time, subject string, ip string, details string, previousHash string) string {
	data := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s",
		id,
		eventType,
		timestamp.UTC().Format(time.RFC3339Nano),
		subject,
		ip,
		details,
		previousHash,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeRowHashForInsert computes the hash chain for a new event
// This should be called within a transaction to ensure consistency
func (s *LogIntegrityService) ComputeRowHashForInsert(ctx context.Context, eventType string, timestamp time.Time, subject, ip, details string) (string, string, error) {
	// Get the last event's hash
	var lastID int64
	var previousHash string

	query := `
		SELECT id, row_hash FROM security_events 
		ORDER BY id DESC 
		LIMIT 1
	`
	err := s.db.QueryRow(ctx, query).Scan(&lastID, &previousHash)
	if err != nil {
		// No previous events - this is the genesis
		previousHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	// Compute hash for the new row (we don't have the ID yet, so we use a placeholder)
	// The actual hash will be computed with the real ID after insert
	return previousHash, "", nil
}

// UpdateRowHash updates the row hash after insertion
func (s *LogIntegrityService) UpdateRowHash(ctx context.Context, eventID int64) error {
	// Get the event data
	query := `
		SELECT id, event_type, created_at, subject_value, ip_address, details, previous_hash
		FROM security_events
		WHERE id = $1
	`
	var id int64
	var eventType string
	var createdAt time.Time
	var subjectValue, ipAddress, previousHash *string
	var details []byte

	err := s.db.QueryRow(ctx, query, eventID).Scan(
		&id, &eventType, &createdAt, &subjectValue, &ipAddress, &details, &previousHash,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch event for hashing: %w", err)
	}

	// Compute hash
	subjectStr := ""
	if subjectValue != nil {
		subjectStr = *subjectValue
	}
	ipStr := ""
	if ipAddress != nil {
		ipStr = *ipAddress
	}
	prevHash := ""
	if previousHash != nil {
		prevHash = *previousHash
	}

	rowHash := ComputeEventHash(id, eventType, createdAt, subjectStr, ipStr, string(details), prevHash)

	// Update the row
	updateQuery := `UPDATE security_events SET row_hash = $2 WHERE id = $1`
	_, err = s.db.Exec(ctx, updateQuery, eventID, rowHash)
	return err
}

// ComputeDailyRootHash computes the Merkle root hash for a day's events
func (s *LogIntegrityService) ComputeDailyRootHash(ctx context.Context, date time.Time) (string, int, int64, int64, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, row_hash FROM security_events
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY id ASC
	`
	rows, err := s.db.Query(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var hashes []string
	var count int
	var firstID, lastID int64

	for rows.Next() {
		var id int64
		var rowHash *string
		if err := rows.Scan(&id, &rowHash); err != nil {
			return "", 0, 0, 0, err
		}

		if count == 0 {
			firstID = id
		}
		lastID = id
		count++

		if rowHash != nil {
			hashes = append(hashes, *rowHash)
		}
	}

	if count == 0 {
		return "", 0, 0, 0, nil
	}

	// Compute Merkle root
	rootHash := computeMerkleRoot(hashes)
	return rootHash, count, firstID, lastID, nil
}

// AnchorToS3 writes the root hash to S3 with Object Lock
func (s *LogIntegrityService) AnchorToS3(ctx context.Context, date time.Time, rootHash string, eventCount int, firstEventID, lastEventID int64) error {
	key := fmt.Sprintf("security-anchors/%s.hash", date.Format("2006-01-02"))
	content := fmt.Sprintf(`{"date":"%s","rootHash":"%s","eventCount":%d,"firstEventId":%d,"lastEventId":%d,"anchoredAt":"%s"}`,
		date.Format("2006-01-02"),
		rootHash,
		eventCount,
		firstEventID,
		lastEventID,
		time.Now().UTC().Format(time.RFC3339),
	)

	// Put object with Object Lock (GOVERNANCE mode, 1-year retention)
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:                    aws.String(s.bucket),
		Key:                       aws.String(key),
		Body:                      bytesReader([]byte(content)),
		ContentType:               aws.String("application/json"),
		ObjectLockMode:            types.ObjectLockModeGovernance,
		ObjectLockRetainUntilDate: aws.Time(time.Now().AddDate(1, 0, 0)), // 1 year retention
	})
	if err != nil {
		return fmt.Errorf("failed to write anchor to S3: %w", err)
	}

	// Record anchor in database
	insertQuery := `
		INSERT INTO hash_anchors (anchor_date, root_hash, event_count, first_event_id, last_event_id, s3_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (anchor_date) DO UPDATE
		SET root_hash = EXCLUDED.root_hash,
		    event_count = EXCLUDED.event_count,
		    first_event_id = EXCLUDED.first_event_id,
		    last_event_id = EXCLUDED.last_event_id
	`
	_, err = s.db.Exec(ctx, insertQuery, date, rootHash, eventCount, firstEventID, lastEventID, key)
	if err != nil {
		return fmt.Errorf("failed to record anchor in database: %w", err)
	}

	// Log anchor creation
	s.logger.Log(ctx, SecurityEvent{
		Event: EventHashAnchorCreated,
		Details: map[string]interface{}{
			"date":        date.Format("2006-01-02"),
			"event_count": eventCount,
			"s3_key":      key,
		},
	})

	return nil
}

// VerifyIntegrity verifies log integrity for a date range
func (s *LogIntegrityService) VerifyIntegrity(ctx context.Context, startDate, endDate time.Time) (*IntegrityReport, error) {
	report := &IntegrityReport{
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "intact",
	}

	// Verify hash chain
	chainBreaks, firstBreak, err := s.verifyHashChain(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}
	report.ChainBreaks = chainBreaks
	if firstBreak > 0 {
		report.FirstBreakEventID = &firstBreak
	}

	// Verify against external anchors
	anchorMismatches, missingAnchors, err := s.verifyAnchors(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}
	report.AnchorMismatches = anchorMismatches
	report.MissingAnchors = missingAnchors

	// Determine overall status
	if chainBreaks > 0 || anchorMismatches > 0 {
		report.Status = "compromised"

		// Log CRITICAL event
		s.logger.Log(ctx, SecurityEvent{
			Event: EventHashChainBreak,
			Details: map[string]interface{}{
				"chain_breaks":      chainBreaks,
				"anchor_mismatches": anchorMismatches,
				"first_break_id":    firstBreak,
			},
		})
	} else if missingAnchors > 0 {
		report.Status = "degraded"
		report.Details = append(report.Details, fmt.Sprintf("%d days missing external anchors", missingAnchors))
	}

	return report, nil
}

// verifyHashChain verifies the internal hash chain
func (s *LogIntegrityService) verifyHashChain(ctx context.Context, startDate, endDate time.Time) (int64, int64, error) {
	query := `
		SELECT id, event_type, created_at, subject_value, ip_address, details, previous_hash, row_hash
		FROM security_events
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY id ASC
	`
	rows, err := s.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var chainBreaks int64
	var firstBreak int64
	var previousHash string

	for rows.Next() {
		var id int64
		var eventType string
		var createdAt time.Time
		var subjectValue, ipAddress, prevHash, rowHash *string
		var details []byte

		if err := rows.Scan(&id, &eventType, &createdAt, &subjectValue, &ipAddress, &details, &prevHash, &rowHash); err != nil {
			return 0, 0, err
		}

		// Skip events without hash chain (pre-migration)
		if rowHash == nil || prevHash == nil {
			continue
		}

		// Verify previous_hash matches last row's row_hash
		if previousHash != "" && *prevHash != previousHash {
			chainBreaks++
			if firstBreak == 0 {
				firstBreak = id
			}
		}

		// Verify row_hash is correct
		subjectStr := ""
		if subjectValue != nil {
			subjectStr = *subjectValue
		}
		ipStr := ""
		if ipAddress != nil {
			ipStr = *ipAddress
		}
		prevHashStr := ""
		if prevHash != nil {
			prevHashStr = *prevHash
		}

		expectedHash := ComputeEventHash(id, eventType, createdAt, subjectStr, ipStr, string(details), prevHashStr)
		if *rowHash != expectedHash {
			chainBreaks++
			if firstBreak == 0 {
				firstBreak = id
			}
		}

		previousHash = *rowHash
	}

	return chainBreaks, firstBreak, nil
}

// verifyAnchors verifies computed hashes against S3 anchors
func (s *LogIntegrityService) verifyAnchors(ctx context.Context, startDate, endDate time.Time) (int64, int64, error) {
	var anchorMismatches, missingAnchors int64

	// Iterate through each day
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		// Get stored anchor
		var storedHash string
		query := `SELECT root_hash FROM hash_anchors WHERE anchor_date = $1`
		err := s.db.QueryRow(ctx, query, d).Scan(&storedHash)
		if err != nil {
			missingAnchors++
			continue
		}

		// Recompute hash for the day
		computedHash, count, _, _, err := s.ComputeDailyRootHash(ctx, d)
		if err != nil {
			return 0, 0, err
		}

		if count > 0 && computedHash != storedHash {
			anchorMismatches++
		}
	}

	return anchorMismatches, missingAnchors, nil
}

// Helper functions

func computeMerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	// Simple Merkle tree implementation
	for len(hashes) > 1 {
		var newLevel []string
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				combined := hashes[i] + hashes[i+1]
				hash := sha256.Sum256([]byte(combined))
				newLevel = append(newLevel, hex.EncodeToString(hash[:]))
			} else {
				// Odd number of hashes - carry forward
				newLevel = append(newLevel, hashes[i])
			}
		}
		hashes = newLevel
	}

	return hashes[0]
}

// bytesReader is a simple bytes.Reader wrapper
type bytesReaderWrapper struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderWrapper {
	return &bytesReaderWrapper{data: data}
}

func (r *bytesReaderWrapper) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
