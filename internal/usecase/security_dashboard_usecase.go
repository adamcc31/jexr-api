package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/security"
)

// SecurityDashboardUsecase implements the security dashboard business logic
type SecurityDashboardUsecase struct {
	repo             domain.SecurityDashboardRepository
	authService      *security.SecurityAuthService
	integrityService *security.LogIntegrityService
	logger           *security.SecurityLogger

	// Cache for stats (1 minute TTL)
	statsCache    *domain.SecurityDashboardStats
	statsCacheAt  time.Time
	statsCacheTTL time.Duration
	statsMutex    sync.RWMutex
}

// NewSecurityDashboardUsecase creates a new security dashboard usecase
func NewSecurityDashboardUsecase(
	repo domain.SecurityDashboardRepository,
	authService *security.SecurityAuthService,
	integrityService *security.LogIntegrityService,
) *SecurityDashboardUsecase {
	return &SecurityDashboardUsecase{
		repo:             repo,
		authService:      authService,
		integrityService: integrityService,
		logger:           security.DefaultLogger(),
		statsCacheTTL:    1 * time.Minute,
	}
}

// GetStats returns cached dashboard statistics
func (u *SecurityDashboardUsecase) GetStats(ctx context.Context) (*domain.SecurityDashboardStats, error) {
	// Check cache
	u.statsMutex.RLock()
	if u.statsCache != nil && time.Since(u.statsCacheAt) < u.statsCacheTTL {
		stats := u.statsCache
		u.statsMutex.RUnlock()
		return stats, nil
	}
	u.statsMutex.RUnlock()

	// Fetch fresh stats
	stats, err := u.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	u.statsMutex.Lock()
	u.statsCache = stats
	u.statsCacheAt = time.Now()
	u.statsMutex.Unlock()

	return stats, nil
}

// ListEvents returns filtered security events
func (u *SecurityDashboardUsecase) ListEvents(ctx context.Context, filter domain.SecurityEventFilter) ([]domain.SecurityEventView, int64, error) {
	// Apply defaults
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return u.repo.ListEvents(ctx, filter)
}

// GetAuthFailureHeatmap returns time-bucketed auth failure data
func (u *SecurityDashboardUsecase) GetAuthFailureHeatmap(ctx context.Context, startTime, endTime time.Time) (*domain.HeatmapData, error) {
	// Determine bucket size based on time range
	duration := endTime.Sub(startTime)
	bucketSize := "hour"
	if duration > 7*24*time.Hour {
		bucketSize = "day"
	}

	return u.repo.GetAuthFailureHeatmap(ctx, startTime, endTime, bucketSize)
}

// GetPrivilegedActionTimeline returns admin action timeline
func (u *SecurityDashboardUsecase) GetPrivilegedActionTimeline(ctx context.Context, page, pageSize int) ([]domain.PrivilegedActionView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	return u.repo.GetPrivilegedActionTimeline(ctx, pageSize, offset)
}

// RequestExport creates a new export request with validation
func (u *SecurityDashboardUsecase) RequestExport(ctx context.Context, userID string, req domain.CreateExportRequest) (*domain.ExportRequest, error) {
	// Validate justification length
	if len(req.Justification) < 20 {
		return nil, fmt.Errorf("justification must be at least 20 characters")
	}

	// Log export request
	u.logger.Log(ctx, security.SecurityEvent{
		Event:        security.EventDataExport,
		SubjectType:  "user_id",
		SubjectValue: security.HashValue(userID),
		Details: map[string]interface{}{
			"justification_preview": req.Justification[:min(50, len(req.Justification))],
		},
	})

	return u.repo.CreateExportRequest(ctx, userID, req)
}

// ApproveExport approves an export request
func (u *SecurityDashboardUsecase) ApproveExport(ctx context.Context, exportID, approverID string) error {
	// Get the export request first
	export, err := u.repo.GetExportRequest(ctx, exportID)
	if err != nil {
		return err
	}

	if export.Status != "pending" {
		return fmt.Errorf("export request is not pending")
	}

	// Approver cannot be the same as requester
	if export.RequestedBy == approverID {
		return fmt.Errorf("cannot approve own export request")
	}

	err = u.repo.ApproveExportRequest(ctx, exportID, approverID)
	if err != nil {
		return err
	}

	// Log approval
	u.logger.Log(ctx, security.SecurityEvent{
		Event:        security.EventDataExportApproved,
		SubjectType:  "export_request",
		SubjectValue: exportID,
		Details: map[string]interface{}{
			"approver_id":  security.HashValue(approverID),
			"requester_id": security.HashValue(export.RequestedBy),
		},
	})

	return nil
}

// RejectExport rejects an export request
func (u *SecurityDashboardUsecase) RejectExport(ctx context.Context, exportID, approverID, reason string) error {
	if len(reason) < 10 {
		return fmt.Errorf("rejection reason must be at least 10 characters")
	}

	err := u.repo.RejectExportRequest(ctx, exportID, approverID, reason)
	if err != nil {
		return err
	}

	// Log rejection
	u.logger.Log(ctx, security.SecurityEvent{
		Event:        security.EventDataExportRejected,
		SubjectType:  "export_request",
		SubjectValue: exportID,
		Details: map[string]interface{}{
			"approver_id": security.HashValue(approverID),
			"reason":      reason,
		},
	})

	return nil
}

// GetExportData retrieves export data for download
func (u *SecurityDashboardUsecase) GetExportData(ctx context.Context, exportID, userID string) ([]domain.SecurityEventView, error) {
	// Verify export is approved and not expired
	export, err := u.repo.GetExportRequest(ctx, exportID)
	if err != nil {
		return nil, fmt.Errorf("export request not found")
	}

	if export.Status != "approved" {
		return nil, fmt.Errorf("export request not approved")
	}

	if export.DownloadExpires != nil && export.DownloadExpires.Before(time.Now()) {
		return nil, fmt.Errorf("export download has expired")
	}

	// Verify requester or approver is downloading
	if export.RequestedBy != userID {
		if export.ApprovedBy == nil || *export.ApprovedBy != userID {
			return nil, fmt.Errorf("not authorized to download this export")
		}
	}

	// Increment download count
	u.repo.IncrementDownloadCount(ctx, exportID)

	// Fetch the events based on filter
	events, _, err := u.repo.ListEvents(ctx, export.Filter)
	return events, err
}

// ActivateBreakGlass activates a time-limited DEVELOPER_ROOT session
func (u *SecurityDashboardUsecase) ActivateBreakGlass(ctx context.Context, userID string, req domain.BreakGlassRequest) (*domain.BreakGlassResponse, error) {
	// Validate duration
	validDurations := map[int]bool{15: true, 30: true, 60: true}
	if !validDurations[req.DurationMinutes] {
		return nil, fmt.Errorf("invalid duration: must be 15, 30, or 60 minutes")
	}

	// Check for existing active session
	existing, active, err := u.authService.CheckBreakGlassActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, fmt.Errorf("break-glass session already active, expires at %s", existing.ExpiresAt.Format(time.RFC3339))
	}

	// Activate
	session, err := u.authService.ActivateBreakGlass(ctx, userID, req.Justification, req.DurationMinutes)
	if err != nil {
		return nil, err
	}

	return &domain.BreakGlassResponse{
		SessionID:     session.ID,
		ActivatedAt:   session.ActivatedAt,
		ExpiresAt:     session.ExpiresAt,
		RemainingMins: int(session.ExpiresAt.Sub(time.Now()).Minutes()),
	}, nil
}

// GetActiveBreakGlass returns the current active break-glass session
func (u *SecurityDashboardUsecase) GetActiveBreakGlass(ctx context.Context, userID string) (*domain.BreakGlassResponse, error) {
	session, active, err := u.authService.CheckBreakGlassActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, nil
	}

	return &domain.BreakGlassResponse{
		SessionID:     session.ID,
		ActivatedAt:   session.ActivatedAt,
		ExpiresAt:     session.ExpiresAt,
		RemainingMins: int(session.ExpiresAt.Sub(time.Now()).Minutes()),
	}, nil
}

// RevokeBreakGlass revokes an active break-glass session
func (u *SecurityDashboardUsecase) RevokeBreakGlass(ctx context.Context, sessionID, reason string) error {
	if len(reason) < 10 {
		return fmt.Errorf("revocation reason must be at least 10 characters")
	}
	return u.authService.RevokeBreakGlass(ctx, sessionID, reason)
}

// VerifyIntegrity performs a full integrity check
func (u *SecurityDashboardUsecase) VerifyIntegrity(ctx context.Context, startDate, endDate time.Time) (*security.IntegrityReport, error) {
	if u.integrityService == nil {
		return nil, fmt.Errorf("integrity service not configured")
	}
	return u.integrityService.VerifyIntegrity(ctx, startDate, endDate)
}

// GetIntegrityStatus returns current integrity status
func (u *SecurityDashboardUsecase) GetIntegrityStatus(ctx context.Context) (string, *time.Time, error) {
	anchor, err := u.repo.GetLastAnchor(ctx)
	if err != nil {
		return "degraded", nil, nil // No anchors yet
	}

	status := "intact"
	if anchor.VerificationStatus == "failed" {
		status = "compromised"
	}

	return status, &anchor.AnchorDate, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
