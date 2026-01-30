package config

import (
	"os"
	"strings"
)

type Config struct {
	Database DatabaseConfig
	Supabase SupabaseConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	GinMode  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type SupabaseConfig struct {
	URL            string
	AnonKey        string
	ServiceRoleKey string
	Bucket         string
}

type JWTConfig struct {
	Secret string
	Expiry string
}

type SMTPConfig struct {
	Host     string
	Port     string
	Email    string
	Password string
}

func New() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "rpms_db"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Supabase: SupabaseConfig{
			URL:            getEnv("SUPABASE_URL", ""),
			AnonKey:        getEnv("SUPABASE_ANON_KEY", ""),
			ServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
			Bucket:         getEnv("SUPABASE_BUCKET", "chat-attachments"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-secret-key"),
			Expiry: getEnv("JWT_EXPIRY", "24h"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnv("SMTP_PORT", "587"),
			Email:    getEnv("SMTP_EMAIL", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
		},
		GinMode: getEnv("GIN_MODE", "debug"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDatabaseURL() string {
	return c.buildDatabaseURL()
}

func (c *Config) buildDatabaseURL() string {
	var sb strings.Builder

	sb.WriteString("postgres://")
	sb.WriteString(c.Database.User)
	if c.Database.Password != "" {
		sb.WriteString(":")
		sb.WriteString(c.Database.Password)
	}
	sb.WriteString("@")
	sb.WriteString(c.Database.Host)
	sb.WriteString(":")
	sb.WriteString(c.Database.Port)
	sb.WriteString("/")
	sb.WriteString(c.Database.DBName)

	if c.Database.SSLMode != "" {
		sb.WriteString("?sslmode=")
		sb.WriteString(c.Database.SSLMode)
	}

	return sb.String()
}

func (c *Config) GetCORSOrigins() []string {
	origins := getEnv("CORS_ORIGINS", "http://localhost:3000")
	return strings.Split(origins, ",")
}
