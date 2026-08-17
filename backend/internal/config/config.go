package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL string
	MinIO       MinIOConfig
	SMTP        SMTPConfig
	JWTSecret   string
	AppURL      string
	APIURL      string
	Port        string
}

type MinIOConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	PublicUseSSL   bool
}

type SMTPConfig struct {
	Host          string
	Port          int
	User          string
	Password      string
	From          string
	TLSServerName string
	SkipTLSVerify bool
	Enabled       bool
}

func Load() *Config {
	endpoint, useSSL := parseMinIOEndpoint(
		getEnv("MINIO_ENDPOINT", "localhost:9000"),
		os.Getenv("MINIO_USE_SSL") == "true",
	)
	publicEndpoint, publicUseSSL := endpoint, useSSL
	if rawPublic := getEnv("MINIO_PUBLIC_ENDPOINT", ""); rawPublic != "" {
		publicEndpoint, publicUseSSL = parseMinIOEndpoint(
			rawPublic,
			os.Getenv("MINIO_PUBLIC_USE_SSL") == "true",
		)
	}
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	smtpHost := getEnv("SMTP_HOST", "")
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASS", "")
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres@localhost:5432/kasq?sslmode=disable"),
		MinIO: MinIOConfig{
			Endpoint:       endpoint,
			PublicEndpoint: publicEndpoint,
			AccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:         getEnv("MINIO_BUCKET", "kasq"),
			UseSSL:         useSSL,
			PublicUseSSL:   publicUseSSL,
		},
		SMTP: SMTPConfig{
			Host:          smtpHost,
			Port:          smtpPort,
			User:          smtpUser,
			Password:      smtpPass,
			From:          trimQuotes(getEnv("SMTP_FROM", smtpUser)),
			TLSServerName: getEnv("SMTP_TLS_SERVER_NAME", ""),
			SkipTLSVerify: os.Getenv("SMTP_INSECURE_SKIP_VERIFY") == "true",
			Enabled:       smtpHost != "" && smtpUser != "" && smtpPass != "",
		},
		JWTSecret: getEnv("JWT_SECRET", "dev-jwt-secret-change-in-production"),
		AppURL:    getEnv("APP_URL", "http://localhost:3008"),
		APIURL:    getEnv("API_URL", "http://localhost:8084"),
		Port:      getEnv("PORT", "8084"),
	}
}

func parseMinIOEndpoint(raw string, useSSL bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "https://"):
		return strings.TrimPrefix(raw, "https://"), true
	case strings.HasPrefix(raw, "http://"):
		return strings.TrimPrefix(raw, "http://"), false
	default:
		return raw, useSSL
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func trimQuotes(s string) string {
	return strings.Trim(s, `"'`)
}
