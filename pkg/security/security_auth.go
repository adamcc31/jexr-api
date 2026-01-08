package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// SecurityRole represents the role of a security dashboard user
type SecurityRole string

const (
	RoleSecurityObserver SecurityRole = "SECURITY_OBSERVER"
	RoleSecurityAnalyst  SecurityRole = "SECURITY_ANALYST"
	RoleSecurityAdmin    SecurityRole = "SECURITY_ADMIN"
)

// SecurityUser represents a security dashboard operator
type SecurityUser struct {
	ID                  string       `json:"id"`
	Username            string       `json:"username"`
	Email               string       `json:"email"`
	PasswordHash        string       `json:"-"`
	Role                SecurityRole `json:"role"`
	TOTPSecret          string       `json:"-"`
	TOTPEnabled         bool         `json:"totpEnabled"`
	IsActive            bool         `json:"isActive"`
	LastLoginAt         *time.Time   `json:"lastLoginAt,omitempty"`
	LastLoginIP         string       `json:"lastLoginIP,omitempty"`
	FailedLoginAttempts int          `json:"-"`
	LockedUntil         *time.Time   `json:"-"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}

// SecuritySession represents an active security dashboard session
type SecuritySession struct {
	ID             string     `json:"id"`
	SecurityUserID string     `json:"securityUserId"`
	TokenHash      string     `json:"-"`
	IPAddress      string     `json:"ipAddress"`
	UserAgent      string     `json:"userAgent"`
	CreatedAt      time.Time  `json:"createdAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	RevokedReason  string     `json:"revokedReason,omitempty"`
}

// BreakGlassSession represents a time-limited DEVELOPER_ROOT elevation
type BreakGlassSession struct {
	ID             string     `json:"id"`
	SecurityUserID string     `json:"securityUserId"`
	Justification  string     `json:"justification"`
	ActivatedAt    time.Time  `json:"activatedAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	RevokedReason  string     `json:"revokedReason,omitempty"`
}

// AllowedIPRange represents an IP range allowed to access the security dashboard
type AllowedIPRange struct {
	ID          int       `json:"id"`
	CIDR        string    `json:"cidr"`
	Description string    `json:"description"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SecurityAuthService handles authentication for the security dashboard
type SecurityAuthService struct {
	db           *pgxpool.Pool
	logger       *SecurityLogger
	sessionTTL   time.Duration
	maxAttempts  int
	lockDuration time.Duration
}

// SecurityAuthConfig holds configuration for the security auth service
type SecurityAuthConfig struct {
	SessionTTL   time.Duration // Default: 30 minutes
	MaxAttempts  int           // Default: 5
	LockDuration time.Duration // Default: 15 minutes
}

// DefaultSecurityAuthConfig returns sensible defaults
func DefaultSecurityAuthConfig() SecurityAuthConfig {
	return SecurityAuthConfig{
		SessionTTL:   30 * time.Minute,
		MaxAttempts:  5,
		LockDuration: 15 * time.Minute,
	}
}

// NewSecurityAuthService creates a new security auth service
func NewSecurityAuthService(db *pgxpool.Pool, config SecurityAuthConfig) *SecurityAuthService {
	return &SecurityAuthService{
		db:           db,
		logger:       DefaultLogger(),
		sessionTTL:   config.SessionTTL,
		maxAttempts:  config.MaxAttempts,
		lockDuration: config.LockDuration,
	}
}

// ValidateIP checks if the given IP is in the allowed ranges
// This is the PRIMARY security control
func (s *SecurityAuthService) ValidateIP(ctx context.Context, ipStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	query := `
		SELECT cidr FROM allowed_ip_ranges 
		WHERE is_active = true
	`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to query allowed IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cidrStr string
		if err := rows.Scan(&cidrStr); err != nil {
			continue
		}

		_, network, err := net.ParseCIDR(cidrStr)
		if err != nil {
			continue
		}

		if network.Contains(ip) {
			return true, nil
		}
	}

	// Log IP denial
	s.logger.Log(ctx, SecurityEvent{
		Event:        EventIPDenied,
		IP:           ipStr,
		SubjectType:  "ip",
		SubjectValue: ipStr,
		Details:      map[string]interface{}{"reason": "not_in_allowlist"},
	})

	return false, nil
}

// Authenticate validates username/password and returns the user if successful
func (s *SecurityAuthService) Authenticate(ctx context.Context, username, password, ip, userAgent string) (*SecurityUser, error) {
	// First validate IP
	allowed, err := s.ValidateIP(ctx, ip)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("access denied: IP not in allowlist")
	}

	// Load user
	user, err := s.getUserByUsername(ctx, username)
	if err != nil {
		s.logFailedLogin(ctx, username, ip, userAgent, "user_not_found")
		return nil, errors.New("invalid credentials")
	}

	// Check if locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		s.logFailedLogin(ctx, username, ip, userAgent, "account_locked")
		return nil, fmt.Errorf("account locked until %s", user.LockedUntil.Format(time.RFC3339))
	}

	// Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.incrementFailedAttempts(ctx, user.ID, ip, userAgent)
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

// ValidateTOTP validates the TOTP code for a user
func (s *SecurityAuthService) ValidateTOTP(ctx context.Context, user *SecurityUser, code string) (bool, error) {
	if !user.TOTPEnabled || user.TOTPSecret == "" {
		return false, errors.New("TOTP not enabled for this user")
	}

	valid := totp.Validate(code, user.TOTPSecret)
	if !valid {
		s.logFailedLogin(ctx, user.Username, "", "", "invalid_totp")
		return false, nil
	}

	return true, nil
}

// CreateSession creates a new session for an authenticated user
func (s *SecurityAuthService) CreateSession(ctx context.Context, userID, ip, userAgent string) (*SecuritySession, string, error) {
	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := hashToken(token)

	expiresAt := time.Now().Add(s.sessionTTL)

	query := `
		INSERT INTO security_sessions (security_user_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	session := &SecuritySession{
		SecurityUserID: userID,
		TokenHash:      tokenHash,
		IPAddress:      ip,
		UserAgent:      userAgent,
		ExpiresAt:      expiresAt,
	}

	err := s.db.QueryRow(ctx, query, userID, tokenHash, ip, userAgent, expiresAt).
		Scan(&session.ID, &session.CreatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	s.updateLastLogin(ctx, userID, ip)

	// Log successful login
	s.logger.Log(ctx, SecurityEvent{
		Event:        EventSecDashboardLogin,
		SubjectType:  "user_id",
		SubjectValue: HashValue(userID),
		IP:           ip,
		UserAgent:    userAgent,
	})

	return session, token, nil
}

// ValidateSession validates a session token and returns the session if valid
func (s *SecurityAuthService) ValidateSession(ctx context.Context, token, ip string) (*SecuritySession, *SecurityUser, error) {
	tokenHash := hashToken(token)

	query := `
		SELECT ss.id, ss.security_user_id, ss.ip_address, ss.user_agent, ss.created_at, ss.expires_at,
		       su.id, su.username, su.email, su.role, su.totp_enabled, su.is_active
		FROM security_sessions ss
		JOIN security_users su ON ss.security_user_id = su.id
		WHERE ss.token_hash = $1 
		  AND ss.expires_at > NOW()
		  AND ss.revoked_at IS NULL
		  AND su.is_active = true
	`

	var session SecuritySession
	var user SecurityUser
	err := s.db.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID, &session.SecurityUserID, &session.IPAddress, &session.UserAgent,
		&session.CreatedAt, &session.ExpiresAt,
		&user.ID, &user.Username, &user.Email, &user.Role, &user.TOTPEnabled, &user.IsActive,
	)
	if err != nil {
		return nil, nil, errors.New("invalid or expired session")
	}

	// Validate IP matches session IP (strict session binding)
	if session.IPAddress != ip {
		s.logger.Log(ctx, SecurityEvent{
			Event:        EventUnauthorizedAccess,
			SubjectType:  "session",
			SubjectValue: session.ID,
			IP:           ip,
			Details: map[string]interface{}{
				"reason":     "ip_mismatch",
				"session_ip": session.IPAddress,
				"request_ip": ip,
			},
		})
		return nil, nil, errors.New("session IP mismatch")
	}

	return &session, &user, nil
}

// RevokeSession revokes an active session
func (s *SecurityAuthService) RevokeSession(ctx context.Context, sessionID, reason string) error {
	query := `
		UPDATE security_sessions 
		SET revoked_at = NOW(), revoked_reason = $2
		WHERE id = $1 AND revoked_at IS NULL
	`
	_, err := s.db.Exec(ctx, query, sessionID, reason)
	return err
}

// ActivateBreakGlass creates a time-limited DEVELOPER_ROOT elevation
func (s *SecurityAuthService) ActivateBreakGlass(ctx context.Context, userID, justification string, durationMinutes int) (*BreakGlassSession, error) {
	if len(justification) < 50 {
		return nil, errors.New("justification must be at least 50 characters")
	}

	if durationMinutes > 60 {
		return nil, errors.New("break-glass duration cannot exceed 60 minutes")
	}

	expiresAt := time.Now().Add(time.Duration(durationMinutes) * time.Minute)

	query := `
		INSERT INTO break_glass_sessions (security_user_id, justification, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, activated_at
	`

	session := &BreakGlassSession{
		SecurityUserID: userID,
		Justification:  justification,
		ExpiresAt:      expiresAt,
	}

	err := s.db.QueryRow(ctx, query, userID, justification, expiresAt).
		Scan(&session.ID, &session.ActivatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to activate break-glass: %w", err)
	}

	// Log CRITICAL event
	s.logger.Log(ctx, SecurityEvent{
		Event:        EventBreakglassActivated,
		SubjectType:  "user_id",
		SubjectValue: HashValue(userID),
		Details: map[string]interface{}{
			"duration_minutes": durationMinutes,
			"justification":    justification[:min(100, len(justification))], // Truncate for log
		},
	})

	return session, nil
}

// CheckBreakGlassActive checks if user has an active break-glass session
func (s *SecurityAuthService) CheckBreakGlassActive(ctx context.Context, userID string) (*BreakGlassSession, bool, error) {
	query := `
		SELECT id, justification, activated_at, expires_at
		FROM break_glass_sessions
		WHERE security_user_id = $1 
		  AND expires_at > NOW()
		  AND revoked_at IS NULL
		ORDER BY activated_at DESC
		LIMIT 1
	`

	var session BreakGlassSession
	err := s.db.QueryRow(ctx, query, userID).Scan(
		&session.ID, &session.Justification, &session.ActivatedAt, &session.ExpiresAt,
	)
	if err != nil {
		return nil, false, nil // No active session
	}

	session.SecurityUserID = userID
	return &session, true, nil
}

// RevokeBreakGlass revokes an active break-glass session
func (s *SecurityAuthService) RevokeBreakGlass(ctx context.Context, sessionID, reason string) error {
	query := `
		UPDATE break_glass_sessions 
		SET revoked_at = NOW(), revoked_reason = $2
		WHERE id = $1 AND revoked_at IS NULL
	`
	result, err := s.db.Exec(ctx, query, sessionID, reason)
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		s.logger.Log(ctx, SecurityEvent{
			Event:        EventBreakglassRevoked,
			SubjectType:  "break_glass_session",
			SubjectValue: sessionID,
			Details:      map[string]interface{}{"reason": reason},
		})
	}

	return nil
}

// GenerateTOTPSecret generates a new TOTP secret for a user
func (s *SecurityAuthService) GenerateTOTPSecret(username string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "J-Expert Security",
		AccountName: username,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// EnableTOTP enables TOTP for a user after validating the initial code
func (s *SecurityAuthService) EnableTOTP(ctx context.Context, userID, secret, code string) error {
	// Validate code first
	if !totp.Validate(code, secret) {
		return errors.New("invalid TOTP code")
	}

	query := `
		UPDATE security_users 
		SET totp_secret = $2, totp_enabled = true, updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.Exec(ctx, query, userID, secret)
	return err
}

// StoreTempTOTPSecret stores a temporary TOTP secret during setup
// This is stored in totp_secret column but with totp_enabled = false
func (s *SecurityAuthService) StoreTempTOTPSecret(ctx context.Context, userID, secret string) error {
	query := `
		UPDATE security_users 
		SET totp_secret = $2, totp_enabled = false, updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.Exec(ctx, query, userID, secret)
	return err
}

// GetTempTOTPSecret retrieves the temporary TOTP secret for confirmation
func (s *SecurityAuthService) GetTempTOTPSecret(ctx context.Context, userID string) (string, error) {
	query := `
		SELECT totp_secret FROM security_users 
		WHERE id = $1 AND totp_enabled = false AND totp_secret IS NOT NULL
	`
	var secret string
	err := s.db.QueryRow(ctx, query, userID).Scan(&secret)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// Helper functions

func (s *SecurityAuthService) getUserByUsername(ctx context.Context, username string) (*SecurityUser, error) {
	query := `
		SELECT id, username, email, password_hash, role, totp_secret, totp_enabled,
		       is_active, last_login_at, last_login_ip, failed_login_attempts, locked_until,
		       created_at, updated_at
		FROM security_users
		WHERE username = $1 AND is_active = true
	`

	user := &SecurityUser{}
	var lastLoginIP *string
	err := s.db.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.TOTPSecret, &user.TOTPEnabled, &user.IsActive,
		&user.LastLoginAt, &lastLoginIP, &user.FailedLoginAttempts, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lastLoginIP != nil {
		user.LastLoginIP = *lastLoginIP
	}

	return user, nil
}

func (s *SecurityAuthService) incrementFailedAttempts(ctx context.Context, userID, ip, userAgent string) {
	query := `
		UPDATE security_users 
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE 
		        WHEN failed_login_attempts + 1 >= $2 THEN NOW() + $3::interval 
		        ELSE locked_until 
		    END,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING failed_login_attempts
	`
	var attempts int
	s.db.QueryRow(ctx, query, userID, s.maxAttempts, fmt.Sprintf("%d minutes", int(s.lockDuration.Minutes()))).Scan(&attempts)

	s.logFailedLogin(ctx, userID, ip, userAgent, "invalid_password")

	if attempts >= s.maxAttempts {
		s.logger.Log(ctx, SecurityEvent{
			Event:        EventLoginBlocked,
			SubjectType:  "user_id",
			SubjectValue: HashValue(userID),
			IP:           ip,
			UserAgent:    userAgent,
			Details:      map[string]interface{}{"reason": "max_attempts_exceeded"},
		})
	}
}

func (s *SecurityAuthService) logFailedLogin(ctx context.Context, identifier, ip, userAgent, reason string) {
	s.logger.Log(ctx, SecurityEvent{
		Event:        EventSecDashboardLoginFailed,
		SubjectType:  "identifier",
		SubjectValue: HashValue(identifier),
		IP:           ip,
		UserAgent:    userAgent,
		Details:      map[string]interface{}{"reason": reason},
	})
}

func (s *SecurityAuthService) updateLastLogin(ctx context.Context, userID, ip string) {
	query := `
		UPDATE security_users 
		SET last_login_at = NOW(), last_login_ip = $2, failed_login_attempts = 0, locked_until = NULL
		WHERE id = $1
	`
	s.db.Exec(ctx, query, userID, ip)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// ConstantTimeCompare compares two strings in constant time
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
