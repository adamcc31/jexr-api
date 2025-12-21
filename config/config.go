package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DBUrl             string
	SupabaseUrl       string
	SupabaseKey       string
	SupabaseJWTSecret string
	// SMTP Configuration (Brevo)
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string
	ContactEmailTo string // Recipient for contact form submissions
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	return &Config{
		Port:              getEnv("PORT", "8080"),
		DBUrl:             getEnv("DATABASE_URL", ""),
		SupabaseUrl:       getEnv("SUPABASE_URL", ""),
		SupabaseKey:       getEnv("SUPABASE_KEY", getEnv("SUPABASE_ANON_KEY", "")),
		SupabaseJWTSecret: getEnv("SUPABASE_JWT_SECRET", getEnv("SUPABASE_JWT_KEY", "")),
		// SMTP Configuration
		SMTPHost:       getEnv("SMTP_HOST", "smtp-relay.brevo.com"),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUsername:   getEnv("SMTP_USERNAME", ""),
		SMTPPassword:   getEnv("SMTP_PASSWORD", ""),
		ContactEmailTo: getEnv("CONTACT_EMAIL_TO", "info@jexpertrecruitment.com"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
