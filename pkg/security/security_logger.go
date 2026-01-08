package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EventType represents the type of security event
type EventType string

const (
	EventLoginFailed        EventType = "login_failed"
	EventLoginBlocked       EventType = "login_blocked"
	EventLoginSuccess       EventType = "login_success"
	EventRateLimitTriggered EventType = "rate_limit_triggered"
	EventUnauthorizedAccess EventType = "unauthorized_access"
	EventBlockCreated       EventType = "block_created"
	EventBlockRemoved       EventType = "block_removed"
	EventValidationFailed   EventType = "validation_failed"
)

// SecurityEvent represents a security-related event to be logged
type SecurityEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	Service      string                 `json:"service"`
	Environment  string                 `json:"env"`
	Level        string                 `json:"level"`
	Event        EventType              `json:"event"`
	SubjectType  string                 `json:"subject_type,omitempty"`  // "email", "ip", "user_id"
	SubjectValue string                 `json:"subject_value,omitempty"` // Masked or hashed for PII
	IP           string                 `json:"ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// SecurityLogger provides structured logging for security events
type SecurityLogger struct {
	zapLogger   *zap.Logger
	serviceName string
	environment string
	// Optional: DB persistence function
	persistFunc func(ctx context.Context, event SecurityEvent) error
}

var (
	defaultLogger *SecurityLogger
)

// InitSecurityLogger initializes the security logger with Zap
func InitSecurityLogger(serviceName, environment string) *SecurityLogger {
	// Create production-ready Zap config
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"

	// Set output to stdout for Railway/container environments
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		// Fallback to a basic logger if config fails
		logger, _ = zap.NewProduction()
	}

	sl := &SecurityLogger{
		zapLogger:   logger,
		serviceName: serviceName,
		environment: environment,
	}

	defaultLogger = sl
	return sl
}

// DefaultLogger returns the default security logger instance
func DefaultLogger() *SecurityLogger {
	if defaultLogger == nil {
		// Create a basic logger if not initialized
		return InitSecurityLogger("j-expert-backend", getEnvironment())
	}
	return defaultLogger
}

// SetPersistFunc sets the function to persist events to database
func (sl *SecurityLogger) SetPersistFunc(f func(ctx context.Context, event SecurityEvent) error) {
	sl.persistFunc = f
}

// Log logs a security event
func (sl *SecurityLogger) Log(ctx context.Context, event SecurityEvent) {
	// Fill in defaults
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	event.Service = sl.serviceName
	event.Environment = sl.environment

	// Determine log level based on event type
	level := zapcore.WarnLevel
	switch event.Event {
	case EventLoginSuccess, EventBlockRemoved:
		level = zapcore.InfoLevel
	case EventLoginFailed, EventRateLimitTriggered, EventValidationFailed:
		level = zapcore.WarnLevel
	case EventLoginBlocked, EventBlockCreated, EventUnauthorizedAccess:
		level = zapcore.ErrorLevel
	}
	event.Level = level.String()

	// Build Zap fields
	fields := []zap.Field{
		zap.String("service", event.Service),
		zap.String("env", event.Environment),
		zap.String("event", string(event.Event)),
	}
	if event.SubjectType != "" {
		fields = append(fields, zap.String("subject_type", event.SubjectType))
	}
	if event.SubjectValue != "" {
		fields = append(fields, zap.String("subject_value", event.SubjectValue))
	}
	if event.IP != "" {
		fields = append(fields, zap.String("ip", event.IP))
	}
	if event.UserAgent != "" {
		fields = append(fields, zap.String("user_agent", event.UserAgent))
	}
	if event.RequestID != "" {
		fields = append(fields, zap.String("request_id", event.RequestID))
	}
	if len(event.Details) > 0 {
		detailsJSON, _ := json.Marshal(event.Details)
		fields = append(fields, zap.String("details", string(detailsJSON)))
	}

	// Log to Zap
	sl.zapLogger.Log(level, string(event.Event), fields...)

	// Persist to DB if configured
	if sl.persistFunc != nil {
		go func(e SecurityEvent) {
			// Use Background context because request context might be canceled
			// Ideally we should use a timeout context here
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := sl.persistFunc(ctx, e); err != nil {
				sl.zapLogger.Error("Failed to persist security event", zap.Error(err))
			}
		}(event)
	}
}

// LogLoginFailed logs a failed login attempt
func (sl *SecurityLogger) LogLoginFailed(ctx context.Context, email, ip, userAgent, requestID, reason string) {
	sl.Log(ctx, SecurityEvent{
		Event:        EventLoginFailed,
		SubjectType:  "email",
		SubjectValue: MaskEmail(email),
		IP:           ip,
		UserAgent:    userAgent,
		RequestID:    requestID,
		Details:      map[string]interface{}{"reason": reason},
	})
}

// LogLoginBlocked logs when a login is blocked due to too many attempts
func (sl *SecurityLogger) LogLoginBlocked(ctx context.Context, email, ip, userAgent, requestID string) {
	sl.Log(ctx, SecurityEvent{
		Event:        EventLoginBlocked,
		SubjectType:  "email",
		SubjectValue: MaskEmail(email),
		IP:           ip,
		UserAgent:    userAgent,
		RequestID:    requestID,
		Details:      map[string]interface{}{"reason": "too_many_failed_attempts"},
	})
}

// LogRateLimitTriggered logs when rate limiting is triggered
func (sl *SecurityLogger) LogRateLimitTriggered(ctx context.Context, ip, userAgent, requestID, endpoint string) {
	sl.Log(ctx, SecurityEvent{
		Event:        EventRateLimitTriggered,
		SubjectType:  "ip",
		SubjectValue: ip,
		IP:           ip,
		UserAgent:    userAgent,
		RequestID:    requestID,
		Details:      map[string]interface{}{"endpoint": endpoint},
	})
}

// LogBlockCreated logs when a block is created
func (sl *SecurityLogger) LogBlockCreated(ctx context.Context, subjectType, subjectValue, ip, requestID string, durationMinutes int) {
	sl.Log(ctx, SecurityEvent{
		Event:        EventBlockCreated,
		SubjectType:  subjectType,
		SubjectValue: maskValue(subjectType, subjectValue),
		IP:           ip,
		RequestID:    requestID,
		Details:      map[string]interface{}{"duration_minutes": durationMinutes},
	})
}

// Sync flushes any buffered log entries
func (sl *SecurityLogger) Sync() error {
	return sl.zapLogger.Sync()
}

// --- Helper Functions ---

// MaskEmail masks an email for logging (e.g., "j***@example.com")
func MaskEmail(email string) string {
	if len(email) < 3 {
		return "***"
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex <= 1 {
		return "***" + email[1:]
	}
	return string(email[0]) + "***" + email[atIndex:]
}

// HashValue creates a SHA256 hash of a value (for logging without PII)
func HashValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:8]) // First 16 chars of hex
}

// maskValue masks a value based on its type
func maskValue(subjectType, value string) string {
	switch subjectType {
	case "email":
		return MaskEmail(value)
	case "ip":
		return value // IPs are not PII in security context
	default:
		return HashValue(value)
	}
}

// getEnvironment determines the current environment
func getEnvironment() string {
	env := os.Getenv("GIN_MODE")
	if env == "release" {
		return "production"
	}
	return "development"
}
