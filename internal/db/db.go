// internal/db/db.go
package db

import (
	"crm-communication-api/config"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// InitDB initializes and returns a GORM database connection pool.
func InitDB(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	// Configure GORM's logger
	gormLogLevel := gormlogger.Silent
	// Log SQL queries in development environment for debugging
	if cfg.GoEnv == "development" {
		gormLogLevel = gormlogger.Info
	}
	gormLogger := gormlogger.New(
		// Use an adapter to integrate with slog
		slog.NewLogLogger(logger.Handler(), slog.LevelInfo),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond, // Log queries slower than 200ms
			LogLevel:                  gormLogLevel,           // Set log level based on environment
			IgnoreRecordNotFoundError: true,                   // Don't log ErrRecordNotFound errors
			Colorful:                  false,                  // Disable colored output for consistency
		},
	)

	// Connect to the database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger, // Use the configured GORM logger
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get the underlying sql.DB instance for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)             // Max number of idle connections
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)             // Max number of open connections
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute) // Max lifetime of connections

	// Ping the database to verify connection
	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}


	logger.Info("Database connection established and pool configured")
	return db, nil
}