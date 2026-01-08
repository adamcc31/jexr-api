package domain

import (
	"context"
	"time"

	"go-recruitment-backend/pkg/security"
)

// SecurityDashboardStats contains aggregated statistics for the dashboard
type SecurityDashboardStats struct {
	TotalEvents        int64            `json:"totalEvents"`
	EventsBySeverity   map[string]int64 `json:"eventsBySeverity"`
	EventsByType       map[string]int64 `json:"eventsByType"`
	TopIPs             []IPSummary      `json:"topIps"`
	FailedLogins24h    int64            `json:"failedLogins24h"`
	BlockedAttempts24h int64            `json:"blockedAttempts24h"`
	CriticalEvents24h  int64            `json:"criticalEvents24h"`
	ActiveBreakGlass   int              `json:"activeBreakGlass"`
	IntegrityStatus    string           `json:"integrityStatus"` // intact, degraded, compromised
	LastAnchorDate     *time.Time       `json:"lastAnchorDate,omitempty"`
}

// IPSummary represents aggregated stats for an IP address
type IPSummary struct {
	IP              string `json:"ip"`
	EventCount      int64  `json:"eventCount"`
	FailedLogins    int64  `json:"failedLogins"`
	LastSeen        string `json:"lastSeen"`
	HighestSeverity string `json:"highestSeverity"`
}

// SecurityEventFilter defines filters for querying security events
type SecurityEventFilter struct {
	StartTime  *time.Time `json:"startTime,omitempty"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
	Severities []string   `json:"severities,omitempty"`
	SearchIP   string     `json:"searchIp,omitempty"`
	SearchUser string     `json:"searchUser,omitempty"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

// SecurityEventView represents a security event for display
type SecurityEventView struct {
	ID           int64                  `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"eventType"`
	Severity     string                 `json:"severity"`
	SubjectType  string                 `json:"subjectType,omitempty"`
	SubjectValue string                 `json:"subjectValue,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	UserAgent    string                 `json:"userAgent,omitempty"`
	RequestID    string                 `json:"requestId,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// HeatmapData represents time-bucketed event counts for visualization
type HeatmapData struct {
	Buckets    []HeatmapBucket `json:"buckets"`
	MaxCount   int64           `json:"maxCount"`
	BucketSize string          `json:"bucketSize"` // "hour", "day"
}

// HeatmapBucket represents a single time bucket in the heatmap
type HeatmapBucket struct {
	Timestamp  time.Time        `json:"timestamp"`
	Count      int64            `json:"count"`
	BySeverity map[string]int64 `json:"bySeverity,omitempty"`
}

// PrivilegedActionView represents an admin action for the timeline
type PrivilegedActionView struct {
	ID            int64                  `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	ActorID       string                 `json:"actorId"`
	ActorUsername string                 `json:"actorUsername,omitempty"`
	ActionType    string                 `json:"actionType"`
	TargetType    string                 `json:"targetType,omitempty"`
	TargetID      string                 `json:"targetId,omitempty"`
	Justification string                 `json:"justification,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// ExportRequest represents a data export request with approval workflow
type ExportRequest struct {
	ID              string              `json:"id"`
	RequestedBy     string              `json:"requestedBy"`
	RequestedAt     time.Time           `json:"requestedAt"`
	Filter          SecurityEventFilter `json:"filter"`
	Justification   string              `json:"justification"`
	Status          string              `json:"status"` // pending, approved, rejected, expired
	ApprovedBy      *string             `json:"approvedBy,omitempty"`
	ApprovedAt      *time.Time          `json:"approvedAt,omitempty"`
	RejectionReason *string             `json:"rejectionReason,omitempty"`
	DownloadCount   int                 `json:"downloadCount"`
	DownloadExpires *time.Time          `json:"downloadExpires,omitempty"`
}

// CreateExportRequest represents a request to create a data export
type CreateExportRequest struct {
	Filter        SecurityEventFilter `json:"filter" binding:"required"`
	Justification string              `json:"justification" binding:"required,min=20"`
}

// ApproveExportRequest represents a request to approve/reject an export
type ApproveExportRequest struct {
	ExportID        string `json:"exportId" binding:"required"`
	Approved        bool   `json:"approved"`
	RejectionReason string `json:"rejectionReason,omitempty"`
}

// BreakGlassRequest represents a request to activate break-glass
type BreakGlassRequest struct {
	Justification   string `json:"justification" binding:"required,min=50"`
	DurationMinutes int    `json:"durationMinutes" binding:"required,oneof=15 30 60"`
}

// BreakGlassResponse represents an active break-glass session
type BreakGlassResponse struct {
	SessionID     string    `json:"sessionId"`
	ActivatedAt   time.Time `json:"activatedAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
	RemainingMins int       `json:"remainingMinutes"`
}

// IntegrityVerificationRequest represents a request to verify log integrity
type IntegrityVerificationRequest struct {
	StartDate string `json:"startDate" binding:"required"` // YYYY-MM-DD
	EndDate   string `json:"endDate" binding:"required"`   // YYYY-MM-DD
}

// SecurityDashboardRepository defines data access for the security dashboard
type SecurityDashboardRepository interface {
	// Stats
	GetStats(ctx context.Context) (*SecurityDashboardStats, error)

	// Events
	ListEvents(ctx context.Context, filter SecurityEventFilter) ([]SecurityEventView, int64, error)
	GetAuthFailureHeatmap(ctx context.Context, startTime, endTime time.Time, bucketSize string) (*HeatmapData, error)
	GetPrivilegedActionTimeline(ctx context.Context, limit, offset int) ([]PrivilegedActionView, int64, error)

	// Export
	CreateExportRequest(ctx context.Context, userID string, req CreateExportRequest) (*ExportRequest, error)
	GetExportRequest(ctx context.Context, exportID string) (*ExportRequest, error)
	ListExportRequests(ctx context.Context, status string, limit, offset int) ([]ExportRequest, int64, error)
	ApproveExportRequest(ctx context.Context, exportID, approverID string) error
	RejectExportRequest(ctx context.Context, exportID, approverID, reason string) error
	IncrementDownloadCount(ctx context.Context, exportID string) error

	// Integrity
	GetLastAnchor(ctx context.Context) (*security.HashAnchor, error)
	ListAnchors(ctx context.Context, limit, offset int) ([]security.HashAnchor, int64, error)
}

// SecurityDashboardUsecase defines business logic for the security dashboard
type SecurityDashboardUsecase interface {
	// Stats
	GetStats(ctx context.Context) (*SecurityDashboardStats, error)

	// Events
	ListEvents(ctx context.Context, filter SecurityEventFilter) ([]SecurityEventView, int64, error)
	GetAuthFailureHeatmap(ctx context.Context, startTime, endTime time.Time) (*HeatmapData, error)
	GetPrivilegedActionTimeline(ctx context.Context, page, pageSize int) ([]PrivilegedActionView, int64, error)

	// Export workflow
	RequestExport(ctx context.Context, userID string, req CreateExportRequest) (*ExportRequest, error)
	ApproveExport(ctx context.Context, exportID, approverID string) error
	RejectExport(ctx context.Context, exportID, approverID, reason string) error
	GetExportData(ctx context.Context, exportID, userID string) ([]SecurityEventView, error)

	// Break-glass
	ActivateBreakGlass(ctx context.Context, userID string, req BreakGlassRequest) (*BreakGlassResponse, error)
	GetActiveBreakGlass(ctx context.Context, userID string) (*BreakGlassResponse, error)
	RevokeBreakGlass(ctx context.Context, sessionID, reason string) error

	// Integrity
	VerifyIntegrity(ctx context.Context, startDate, endDate time.Time) (*security.IntegrityReport, error)
	GetIntegrityStatus(ctx context.Context) (string, *time.Time, error)
}
