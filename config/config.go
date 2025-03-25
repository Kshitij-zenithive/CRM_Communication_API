// config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL           string
	GoogleClientID        string
	GoogleClientSecret    string
	GoogleRedirectURL     string
	SessionSecret         string
	JWTSecretKey          string
	RefreshTokenSecretKey string
	JWTExpiry             time.Duration
	RefreshTokenExpiry    time.Duration
	LogLevel              string
	Port                  string
	GmailSyncInterval     time.Duration
	BaseURL               string // Added Base URL
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables directly.")
	}

	jwtExpiryStr := getEnvOrDefault("JWT_EXPIRY_TIME", "15")
	jwtExpiry, err := strconv.Atoi(jwtExpiryStr)
	if err != nil {
		log.Fatal("Invalid JWT_EXPIRY_TIME:", err)
	}

	refreshExpiryStr := getEnvOrDefault("REFRESH_TOKEN_EXPIRY", "168")
	refreshExpiry, err := strconv.Atoi(refreshExpiryStr)
	if err != nil {
		log.Fatal("Invalid REFRESH_TOKEN_EXPIRY:", err)
	}

	gmailSyncIntervalStr := getEnvOrDefault("GMAIL_SYNC_INTERVAL", "300")
	gmailSyncInterval, err := strconv.Atoi(gmailSyncIntervalStr)
	if err != nil {
		log.Fatal("Invalid GMAIL_SYNC_INTERVAL:", err)
	}

	return &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:     os.Getenv("GOOGLE_REDIRECT_URL"),
		SessionSecret:         os.Getenv("SESSION_SECRET"),
		JWTSecretKey:          getEnvOrDefault("JWT_SECRET_KEY", "default_access_token_secret_key"),
		RefreshTokenSecretKey: getEnvOrDefault("REFRESH_TOKEN_SECRET_KEY", "default_refresh_token_secret_key"),
		JWTExpiry:             time.Duration(jwtExpiry) * time.Minute,
		RefreshTokenExpiry:    time.Duration(refreshExpiry) * time.Hour,
		LogLevel:              getEnvOrDefault("LOG_LEVEL", "INFO"),
		Port:                  getEnvOrDefault("PORT", "8080"),
		GmailSyncInterval:     time.Duration(gmailSyncInterval) * time.Second,
		BaseURL:               getEnvOrDefault("BASE_URL", "http://localhost:8080"), // Default base URL
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
