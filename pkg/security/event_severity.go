package security

// Severity represents the severity level of a security event
// This is derived from EventType, NOT user-provided
type Severity string

const (
	SeverityINFO     Severity = "INFO"
	SeverityMEDIUM   Severity = "MEDIUM"
	SeverityWARN     Severity = "WARN"
	SeverityHIGH     Severity = "HIGH"
	SeverityCRITICAL Severity = "CRITICAL"
)

// Extended event types for security dashboard
const (
	// Administrative events
	EventPasswordReset      EventType = "password_reset"
	EventPasswordChange     EventType = "password_change"
	EventRoleModified       EventType = "role_modified"
	EventUserCreated        EventType = "user_created"
	EventUserDeleted        EventType = "user_deleted"
	EventUserDisabled       EventType = "user_disabled"
	EventConfigChanged      EventType = "config_changed"
	EventDataExport         EventType = "data_export"
	EventDataExportApproved EventType = "data_export_approved"
	EventDataExportRejected EventType = "data_export_rejected"

	// Error and anomaly events
	EventServerError     EventType = "server_error"
	EventSuspiciousInput EventType = "suspicious_input"
	EventCSRFViolation   EventType = "csrf_violation"

	// Break-glass events
	EventBreakglassActivated EventType = "breakglass_activated"
	EventBreakglassExpired   EventType = "breakglass_expired"
	EventBreakglassRevoked   EventType = "breakglass_revoked"

	// Log integrity events
	EventHashAnchorCreated EventType = "hash_anchor_created"
	EventHashChainBreak    EventType = "hash_chain_break"

	// Security dashboard auth events
	EventSecDashboardLogin       EventType = "sec_dashboard_login"
	EventSecDashboardLoginFailed EventType = "sec_dashboard_login_failed"
	EventSecDashboardLogout      EventType = "sec_dashboard_logout"
	EventIPDenied                EventType = "ip_denied"
)

// EventSeverityMap defines the hard-coded severity for each event type
// Severity is DERIVED from EventType, not user-provided
// This enables proper SOC prioritization
var EventSeverityMap = map[EventType]Severity{
	// INFO - Normal operations
	EventLoginSuccess:       SeverityINFO,
	EventBlockRemoved:       SeverityINFO,
	EventSecDashboardLogin:  SeverityINFO,
	EventSecDashboardLogout: SeverityINFO,
	EventHashAnchorCreated:  SeverityINFO,
	EventBreakglassExpired:  SeverityINFO,

	// MEDIUM - Notable but not urgent
	EventPasswordReset:  SeverityMEDIUM,
	EventPasswordChange: SeverityMEDIUM,
	EventDataExport:     SeverityMEDIUM,
	EventServerError:    SeverityMEDIUM,

	// WARN - Potential issues, monitor
	EventLoginFailed:             SeverityWARN,
	EventRateLimitTriggered:      SeverityWARN,
	EventValidationFailed:        SeverityWARN,
	EventSecDashboardLoginFailed: SeverityWARN,

	// HIGH - Active threats or significant changes
	EventLoginBlocked:       SeverityHIGH,
	EventBlockCreated:       SeverityHIGH,
	EventUnauthorizedAccess: SeverityHIGH,
	EventSuspiciousInput:    SeverityHIGH,
	EventCSRFViolation:      SeverityHIGH,
	EventRoleModified:       SeverityHIGH,
	EventUserCreated:        SeverityHIGH,
	EventUserDeleted:        SeverityHIGH,
	EventUserDisabled:       SeverityHIGH,
	EventConfigChanged:      SeverityHIGH,
	EventDataExportApproved: SeverityHIGH,
	EventDataExportRejected: SeverityHIGH,
	EventIPDenied:           SeverityHIGH,
	EventBreakglassRevoked:  SeverityHIGH,

	// CRITICAL - Immediate attention required
	EventBreakglassActivated: SeverityCRITICAL,
	EventHashChainBreak:      SeverityCRITICAL,
}

// GetSeverity returns the severity for an event type
// If the event type is not mapped, defaults to MEDIUM
func GetSeverity(eventType EventType) Severity {
	if severity, ok := EventSeverityMap[eventType]; ok {
		return severity
	}
	return SeverityMEDIUM
}

// IsCritical returns true if the event requires immediate attention
func IsCritical(eventType EventType) bool {
	return GetSeverity(eventType) == SeverityCRITICAL
}

// IsHighOrAbove returns true if the event is HIGH or CRITICAL severity
func IsHighOrAbove(eventType EventType) bool {
	severity := GetSeverity(eventType)
	return severity == SeverityHIGH || severity == SeverityCRITICAL
}
