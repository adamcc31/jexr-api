package config

import (
	"log"
	"os"
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
	}

	// Validasi dasar untuk mencegah panic aneh nanti
	if cfg.DBUrl == "" {
		log.Println("WARNING: DATABASE_URL is missing. Application may fail to connect.")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
