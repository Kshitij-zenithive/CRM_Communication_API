// cmd/worker/main.go
package main

import (
	"context"
	"errors"
	"log" // Use standard log for fatal errors before logger setup
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"crm-communication-api/config"
	"crm-communication-api/internal/auth"

	// dbmodel "crm-communication-api/internal/model" // No longer needed directly
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"

	// "crm-communication-api/internal/util" // Using slog directly

	// "github.com/google/uuid" // No longer needed
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger" // Import GORM logger
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig() // Correctly handle error
	if err != nil {
		log.Fatalf("FATAL: Failed to load config: %v", err)
	}

	// Initialize logger using slog
	logLevel := slog.LevelInfo
	if strings.ToLower(cfg.GoEnv) == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger) // Set global default logger

	// Initialize database connection
	// Use GORM logger config similar to server/main.go
	gormLogLevel := gormlogger.Silent
	if cfg.GoEnv == "development" {
		gormLogLevel = gormlogger.Info
	}
	gormLogger := gormlogger.New(slog.NewLogLogger(logger.Handler(), slog.LevelInfo), gormlogger.Config{
		SlowThreshold: time.Second, LogLevel: gormLogLevel, IgnoreRecordNotFoundError: true, Colorful: false,
	})
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{Logger: gormLogger}) // Use DatabaseURL, pass logger
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
		sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute)
		if err = sqlDB.Ping(); err != nil {
			logger.Error("Failed to ping database", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
	logger.Info("Worker database connection established")

	// Initialize repository
	repo := repository.NewRepository(db)

	// Initialize necessary services
	// We need AuthRepository for EmailService's helpers, Auth Service itself for GetGmailService
	authSvc, err := auth.NewGoogleAuthService(repo, cfg, logger) // Use constructor that returns error
	if err != nil {
		logger.Error("Failed to initialize AuthService", slog.String("error", err.Error()))
		// Depending on worker needs, maybe continue without authSvc? Or exit.
		os.Exit(1)
	}

	// Instantiate ONLY the services the worker explicitly NEEDS for its current tasks (email sync, token cleanup)
	emailSvc := service.NewEmailService(
		repo,    // Implements EmailRepository
		repo,    // Implements ClientRepository
		authSvc, // Pass the *concrete* type that implements auth.Service AND GetGmailService
		repo,    // Implements AuthRepository for user lookups within email service
		nil,     // No TemplateService needed for sync
		nil,     // No TimelineService needed for sync (unless SyncGmail adds events)
		logger,
	)
	// interactionService := service.NewInteractionService(repo) // REMOVED - Not currently used by worker tasks shown

	// Create a context for the worker with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info("Worker started", slog.String("sync_interval", cfg.GmailSyncInterval.String()))

	// Ticker for Gmail Sync
	syncTicker := time.NewTicker(cfg.GmailSyncInterval)
	defer syncTicker.Stop()

	// Ticker for Token Cleanup (e.g., every hour)
	cleanupInterval := 1 * time.Hour // Make configurable if needed
	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-syncTicker.C:
			logger.Info("Worker running email sync task...")
			// Fetch all users
			users, err := repo.GetAllUsers(ctx)
			if err != nil {
				logger.Error("Error fetching users for email sync", slog.String("error", err.Error()))
				continue // Skip this cycle
			}
			logger.Debug("Fetched users for email sync", slog.Int("count", len(users)))

			// Loop and Sync Gmail for eligible users
			for _, user := range users {
				// Check if user has linked a Google account
				_, err := repo.GetOAuthProvider(ctx, user.ID, "google")
				if err != nil {
					if errors.Is(err, repository.ErrNotFound) {
						logger.Debug("Skipping Gmail sync: Google account not linked", slog.String("userID", user.ID.String()))
					} else {
						logger.Error("Failed to get OAuth provider info for sync", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
					}
					continue // Skip this user
				}
				// Synchronize emails
				logger.Debug("Syncing emails for user", slog.String("userID", user.ID.String()), slog.String("email", user.Email))
				if err := emailSvc.SyncGmail(ctx, user.ID.String()); err != nil {
					logger.Error("Failed to sync Gmail for user", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
				}
			}
			logger.Info("Email sync iteration completed.")

		case <-cleanupTicker.C:
			logger.Info("Worker running expired token cleanup task...")
			if err := repo.DeleteExpiredRefreshTokens(ctx); err != nil {
				logger.Error("Failed to delete expired refresh tokens", slog.String("error", err.Error()))
			} else {
				logger.Info("Expired refresh token cleanup complete")
			}

		case <-ctx.Done(): // Handle shutdown signal
			logger.Info("Worker shutting down due to signal...")
			// Perform any necessary cleanup before exiting
			return
		}
	}
}
