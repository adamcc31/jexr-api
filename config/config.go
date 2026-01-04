package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DBUrl             string
	SupabaseUrl       string
	SupabaseKey       string
	SupabaseJWTSecret string
	FrontendURL       string
	// SMTP Configuration (Brevo)
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string
	SMTPFromEmail  string // Verified sender email (different from SMTP login)
	ContactEmailTo string
	// Redis/Upstash Configuration
	UpstashRedisURL      string
	UpstashRedisPassword string
	// Rate Limiting Configuration
	RateLimitWindowSeconds   int
	RateLimitLoginThreshold  int
	RateLimitGlobalThreshold int
	FailedLoginBlockMinutes  int
	FailedLoginMaxAttempts   int
	// Security Configuration
	SecurityLogToDB bool // Whether to persist security events to database
}

func LoadConfig() (*Config, error) {
	// Load .env file (Hanya efektif di Local, diabaikan di Production jika file tidak ada)
	_ = godotenv.Load()

	cfg := &Config{
		Port:  getEnv("PORT", "8080"),
		DBUrl: getEnv("DATABASE_URL", ""),
		// Sanitasi: Hapus slash di akhir URL untuk mencegah double slash (misal: .co//auth)
		SupabaseUrl:       strings.TrimRight(getEnv("SUPABASE_URL", ""), "/"),
		SupabaseKey:       getEnv("SUPABASE_KEY", getEnv("SUPABASE_ANON_KEY", "")),
		SupabaseJWTSecret: getEnv("SUPABASE_JWT_SECRET", getEnv("SUPABASE_JWT_KEY", "")),
		FrontendURL:       strings.TrimRight(getEnv("FRONTEND_URL", "http://localhost:3000"), "/"),
		// SMTP Configuration
		SMTPHost:       getEnv("SMTP_HOST", "smtp-relay.brevo.com"),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUsername:   getEnv("SMTP_USERNAME", ""),
		SMTPPassword:   getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:  getEnv("SMTP_FROM_EMAIL", "noreply@jexpertrecruitment.com"), // Must be verified in Brevo
		ContactEmailTo: getEnv("CONTACT_EMAIL_TO", "info@jexpertrecruitment.com"),
		// Redis/Upstash Configuration
		UpstashRedisURL:      getEnv("UPSTASH_REDIS_URL", ""),
		UpstashRedisPassword: getEnv("UPSTASH_REDIS_PASSWORD", ""),
		// Rate Limiting Configuration (with sensible defaults)
		RateLimitWindowSeconds:   getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),    // 1 minute window
		RateLimitLoginThreshold:  getEnvInt("RATE_LIMIT_LOGIN_THRESHOLD", 10),   // 10 login attempts per window
		RateLimitGlobalThreshold: getEnvInt("RATE_LIMIT_GLOBAL_THRESHOLD", 100), // 100 requests per window
		FailedLoginBlockMinutes:  getEnvInt("FAILED_LOGIN_BLOCK_MINUTES", 15),   // 15 minute block
		FailedLoginMaxAttempts:   getEnvInt("FAILED_LOGIN_MAX_ATTEMPTS", 5),     // 5 failed attempts before block
		// Security Configuration
		SecurityLogToDB: getEnvBool("SECURITY_LOG_TO_DB", true), // Persist security events to DB by default
	}

	// Validasi dasar untuk mencegah panic aneh nanti
	if cfg.DBUrl == "" {
		log.Println("WARNING: DATABASE_URL is missing. Application may fail to connect.")
	}

	// Log Redis configuration status (helpful for debugging)
	if cfg.UpstashRedisURL == "" {
		log.Println("WARNING: UPSTASH_REDIS_URL not configured. Rate limiting will use in-memory fallback.")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvInt returns an integer environment variable or fallback if not set/invalid
func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

// getEnvBool returns a boolean environment variable or fallback if not set/invalid
func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}
