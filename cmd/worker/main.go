// cmd/worker/main.go (CORRECTED)

package main

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/internal/auth"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"
	"crm-communication-api/internal/util"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	logger := util.NewLogger() // No argument needed

	// Initialize database connection
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize repository
	repo := repository.NewRepository(db)

	// Initialize services (only EmailService is directly needed for syncing)
	googleAuthService := auth.NewGoogleAuthService(cfg, repo) // Pass the repository
	// templateService := service.NewTemplateService(repo)    // Not needed right now, but keep for future
	emailService := service.NewEmailService(repo, googleAuthService.GetGmailService) // Corrected call
	// taskService := service.NewTaskService(repo)            // Commented out - not used
	// reminderService := service.NewReminderService(repo)     // Commented out - not used
	// interactionService := service.NewInteractionService(repo) //Not needed at this moment

	// Create a context for the worker
	ctx := context.Background()
	// Create a ticker for periodic tasks
	ticker := time.NewTicker(cfg.GmailSyncInterval) // Use config
	defer ticker.Stop()

	log.Println("Worker started")

	for {
		select {
		case <-ticker.C:
			// Example: Synchronize emails for all users
			log.Println("Running periodic tasks...")

			// 1. Fetch all users (you might want to paginate this for a large number of users)
			users, err := repo.GetAllUsers(ctx) // Use repository method
			if err != nil {
				log.Printf("Error fetching users: %v", err)
				continue // Go to the next tick
			}

			// 2. Loop and Sync
			for _, user := range users {
				//Check if user has linked a google Account
				_, err := repo.GetOAuthProvider(ctx, user.ID, "google")
				if err != nil {
					if err == repository.ErrNotFound {
						log.Printf("User %s (%s) has not linked a Google account. Skipping Gmail sync.", user.Name, user.Email)
					} else {
						log.Printf("Failed to get Auth Provider, err: %v", err)
					}
					continue // Skip
				}
				// Synchronize emails for each user
				log.Printf("Syncing emails for user: %s", user.Email) // Log user
				if err := emailService.SyncGmail(ctx, user.ID.String()); err != nil {
					logger.Error("Failed to sync Gmail for user", "user_id", user.ID, "error", err)
					//Consider retries
				}
			}
			log.Println("Email sync completed.")

		case <-ctx.Done(): // Handle shutdown signal
			log.Println("Worker shutting down")
			return
		}
	}
}
