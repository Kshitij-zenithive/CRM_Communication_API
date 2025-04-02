// config/config.go

package config

import (
	"fmt"
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
	JWTExpiry             time.Duration // Changed name convention
	RefreshTokenExpiry    time.Duration // Changed name convention
	LogLevel              string
	Port                  string
	GmailSyncInterval     time.Duration
	BaseURL               string
	JWTIssuer             string // <-- ADDED: Issuer name for JWT

	// Fields needed for db.InitDB
	DBHost                   string
	DBPort                   string
	DBUser                   string
	DBPassword               string
	DBName                   string
	DBSSLMode                string
	GoEnv                    string // e.g., "development", "production"
	DBMaxIdleConns           int
	DBMaxOpenConns           int
	DBConnMaxLifetimeMinutes int

	// Added missing JWT/Refresh config fields used in auth/jwt.go
	JWTExpiryMinutes           int // Keep original int if used elsewhere
	RefreshTokenExpirationDays int // Keep original int if used elsewhere

}

func LoadConfig() (*Config, error) { // Return error for better handling
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables directly.")
		// Don't treat as fatal, allow env var usage
	}

	// --- JWT/Refresh Timing ---
	jwtExpiryMinStr := getEnvOrDefault("JWT_EXPIRY_MINUTES", "15") // Use MINUTES consistently
	jwtExpiryMin, err := strconv.Atoi(jwtExpiryMinStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_MINUTES: %w", err)
	}

	refreshExpiryDaysStr := getEnvOrDefault("REFRESH_TOKEN_EXPIRY_DAYS", "7") // Use DAYS consistently
	refreshExpiryDays, err := strconv.Atoi(refreshExpiryDaysStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REFRESH_TOKEN_EXPIRY_DAYS: %w", err)
	}

	// --- GMAIL Sync ---
	gmailSyncIntervalStr := getEnvOrDefault("GMAIL_SYNC_INTERVAL_SECONDS", "300") // Use SECONDS
	gmailSyncIntervalSec, err := strconv.Atoi(gmailSyncIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid GMAIL_SYNC_INTERVAL_SECONDS: %w", err)
	}

	// --- DB Pool ---
	dbMaxIdleStr := getEnvOrDefault("DB_MAX_IDLE_CONNS", "10")
	dbMaxIdle, err := strconv.Atoi(dbMaxIdleStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_IDLE_CONNS: %w", err)
	}
	dbMaxOpenStr := getEnvOrDefault("DB_MAX_OPEN_CONNS", "100")
	dbMaxOpen, err := strconv.Atoi(dbMaxOpenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_OPEN_CONNS: %w", err)
	}
	dbConnMaxLifeStr := getEnvOrDefault("DB_CONN_MAX_LIFETIME_MINUTES", "60")
	dbConnMaxLife, err := strconv.Atoi(dbConnMaxLifeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME_MINUTES: %w", err)
	}

	cfg := &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"), // Keep if used directly by GORM Open
		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:     os.Getenv("GOOGLE_REDIRECT_URL"),
		SessionSecret:         os.Getenv("SESSION_SECRET"),                                                                 // Used? Maybe for sessions if added later
		JWTSecretKey:          getEnvOrDefault("JWT_SECRET_KEY", "default_access_token_secret_key_longer_than_32_bytes"),   // Ensure default is secure
		RefreshTokenSecretKey: getEnvOrDefault("REFRESH_TOKEN_SECRET_KEY", "default_refresh_token_secret_key_much_longer"), // Separate, secure default
		JWTExpiry:             time.Duration(jwtExpiryMin) * time.Minute,                                                   // Use time.Duration
		RefreshTokenExpiry:    time.Duration(refreshExpiryDays) * 24 * time.Hour,                                           // Use time.Duration
		LogLevel:              getEnvOrDefault("LOG_LEVEL", "INFO"),
		Port:                  getEnvOrDefault("PORT", "8080"),
		GmailSyncInterval:     time.Duration(gmailSyncIntervalSec) * time.Second,
		BaseURL:               getEnvOrDefault("BASE_URL", "http://localhost:8080"),
		JWTIssuer:             getEnvOrDefault("JWT_ISSUER", "crm-communication-api"), // <-- LOADED

		// DB Fields needed by db.InitDB
		DBHost:                   getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:                   getEnvOrDefault("DB_PORT", "5432"),
		DBUser:                   getEnvOrDefault("DB_USER", "postgres"),
		DBPassword:               getEnvOrDefault("DB_PASSWORD", "password"),
		DBName:                   getEnvOrDefault("DB_NAME", "crm_comm_db"),
		DBSSLMode:                getEnvOrDefault("DB_SSLMODE", "disable"),
		GoEnv:                    getEnvOrDefault("GO_ENV", "development"),
		DBMaxIdleConns:           dbMaxIdle,
		DBMaxOpenConns:           dbMaxOpen,
		DBConnMaxLifetimeMinutes: dbConnMaxLife,

		// Keep original int fields if they are directly used elsewhere (e.g. in older JWT code)
		JWTExpiryMinutes:           jwtExpiryMin,
		RefreshTokenExpirationDays: refreshExpiryDays,
	}

	// Validate required fields
	if cfg.JWTSecretKey == "" || len(cfg.JWTSecretKey) < 32 {
		return nil, fmt.Errorf("JWT_SECRET_KEY is required and must be at least 32 bytes long")
	}
	// Add other validations as needed

	return cfg, nil
}

// getEnvOrDefault remains the same
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
