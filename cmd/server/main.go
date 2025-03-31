// cmd/server/main.go
package main

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/graph"
	"crm-communication-api/internal/auth"
	"crm-communication-api/internal/db"
	"crm-communication-api/internal/middleware"
	"crm-communication-api/internal/model" // Import model for migrations
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// --- Configuration ---
	cfg := config.LoadConfig() // Load config without arguments
	if cfg == nil {
		slog.Error("Failed to load configuration")
		os.Exit(1)
	}

	// --- Logging ---
	logLevel := slog.LevelInfo
	if cfg.GoEnv == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger) // Set as default logger

	logger.Info("Starting CRM Communication API", slog.String("environment", cfg.GoEnv))

	// --- Database ---
	database, err := db.InitDB(cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	// Optional: Get underlying sql.DB for closing later if needed, though GORM handles closing
	// sqlDB, _ := database.DB()
	// defer sqlDB.Close() // Or handle closing during graceful shutdown

	// --- Migrations ---
	logger.Info("Running database migrations...")
	err = database.AutoMigrate(
		&model.User{},
		&model.Client{},
		&model.Conversation{},
		&model.ConversationParticipant{},
		&model.Message{},
		&model.Email{},
		&model.EmailAttachment{},
		&model.EmailTemplate{},
		&model.RefreshToken{},  // Include new models
		&model.OAuthProvider{}, // Include new models
		&model.TimelineEvent{},
		&model.MessageReadReceipt{},
		// Add any other models here
	)
	if err != nil {
		logger.Error("Database migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Database migrations completed successfully")

	// --- Repository ---
	repo := repository.NewRepository(database)
	logger.Info("Repository initialized")

	// --- Services ---
	
	// Initialize AuthService (constructor defined in internal/auth/service.go)
	authService, err := auth.NewAuthService(repo, cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize AuthService", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Example usage of authService to avoid "declared and not used" error
	logger.Info("AuthService initialized successfully", slog.String("service", fmt.Sprintf("%T", authService)))
	
	// cmd/server/main.go

// ... imports ...

	googleAuthService := auth.NewGoogleAuthService(cfg, repo)
	templateService := service.NewTemplateService(repo)
	emailService := service.NewEmailService(repo, repo, googleAuthService.GetGmailService, templateService, timelineService, logger) // CORRECTED: Pass repo as userRepo
	chatService := service.NewChatService(repo, repo, repo, timelineService, notificationSvc, schedulerSvc, logger) // Assuming ChatService needs all repos
	taskService := service.NewTaskService(repo)
	reminderService := service.NewReminderService(repo)
	userService := service.NewUserService(repo)
    clientService := service.NewClientService(repo) // Added ClientService instantiation
	interactionService := service.NewInteractionService(repo)
    timelineService := service.NewTimelineService(repo) // Added TimelineService instantiation
    notificationSvc := service.NewNotificationService(hub) // Added NotificationService instantiation (using InMemory example)
    schedulerSvc := service.NewSimpleSchedulerService(logger) // Added SchedulerService instantiation (using Simple example)


	// Initialize GraphQL resolver
	resolver := graph.NewResolver(
        chatService,
        emailService,
        templateService,
        timelineService, // Pass timeline service
		interactionService, // Pass interaction service (needed by other services, maybe?) - Review dependencies
        taskService,       // Pass task service
        reminderService,   // Pass reminder service
        userService,       // Pass user service
        clientService,     // Pass client service
        repo,
        googleAuthService, // Pass as auth.Service interface
        hub,
        cfg,
    )





	// --- HTTP Router ---
	mux := http.NewServeMux()

	// --- Middleware ---
	authMiddleware := middleware.AuthMiddleware(cfg) // Create instance of auth middleware

	// Example usage of authMiddleware
	mux.Handle("/secure-endpoint", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status": "secure"}`)
	})))

	// --- Routes ---
	// Basic health check
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status": "ok"}`)
	})

	// Placeholder for GraphQL Handler (using auth middleware)
	// graphQLHandler := /* ... Initialize your GraphQL handler here ... */
	// mux.Handle("/graphql", authMiddleware(graphQLHandler)) // Apply auth middleware

	// Add HTTP handlers for OAuth callbacks if needed (might not use auth middleware)
	// mux.HandleFunc("GET /auth/google/callback", /* Handler using authService */ )


	// --- HTTP Server ---
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: mux, // Use the main router
		// Add timeouts for production readiness
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// --- Start Server Goroutine ---
	go func() {
		logger.Info("Server starting", slog.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	// Notify channel on SIGINT (Ctrl+C) or SIGTERM (sent by Docker/Kubernetes)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	logger.Info("Shutdown signal received", slog.String("signal", sig.String()))

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30-second timeout
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1) // Exit with error if shutdown fails
	}

	logger.Info("Server exiting gracefully")
}



// cmd/server/main.go
// package main

// import (
// 	"context"
// 	"crm-communication-api/config"
// 	"crm-communication-api/internal/auth"
// 	"crm-communication-api/internal/db"
// 	"crm-communication-api/internal/middleware"
// 	"crm-communication-api/internal/model" // Import model for migrations
// 	"crm-communication-api/internal/repository"
// 	"errors"
// 	"fmt"
// 	"log/slog"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"
// )

// func main() {
// 	// --- Configuration ---
// 	cfg, err := config.LoadConfig(".") // Load config from current directory (or specify path)
// 	if err != nil {
// 		slog.Error("Failed to load configuration", slog.String("error", err.Error()))
// 		os.Exit(1)
// 	}

// 	// --- Logging ---
// 	logLevel := slog.LevelInfo
// 	if cfg.GoEnv == "development" {
// 		logLevel = slog.LevelDebug
// 	}
// 	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
// 	slog.SetDefault(logger) // Set as default logger

// 	logger.Info("Starting CRM Communication API", slog.String("environment", cfg.GoEnv))

// 	// --- Database ---
// 	database, err := db.InitDB(cfg, logger)
// 	if err != nil {
// 		logger.Error("Failed to initialize database", slog.String("error", err.Error()))
// 		os.Exit(1)
// 	}
// 	// Optional: Get underlying sql.DB for closing later if needed, though GORM handles closing
// 	// sqlDB, _ := database.DB()
// 	// defer sqlDB.Close() // Or handle closing during graceful shutdown

// 	// --- Migrations ---
// 	logger.Info("Running database migrations...")
// 	err = database.AutoMigrate(
// 		&model.User{},
// 		&model.Client{},
// 		&model.Conversation{},
// 		&model.ConversationParticipant{},
// 		&model.Message{},
// 		&model.Email{},
// 		&model.EmailAttachment{},
// 		&model.EmailTemplate{},
// 		&model.RefreshToken{},  // Include new models
// 		&model.OAuthProvider{}, // Include new models
// 		&model.TimelineEvent{},
// 		&model.MessageReadReceipt{},
// 		// Add any other models here
// 	)
// 	if err != nil {
// 		logger.Error("Database migration failed", slog.String("error", err.Error()))
// 		os.Exit(1)
// 	}
// 	logger.Info("Database migrations completed successfully")

// 	// --- Repository ---
// 	repo := repository.NewRepository(database)
// 	logger.Info("Repository initialized")

// 	// --- Services ---
// 	// Initialize AuthService (constructor defined in internal/auth/service.go)
// 	authService, err := auth.NewAuthService(repo, cfg, logger)
// 	if err != nil {
// 		logger.Error("Failed to initialize AuthService", slog.String("error", err.Error()))
// 		os.Exit(1)
// 	}
// 	// Initialize other services here later (e.g., ConversationService, ClientService)

// 	// --- HTTP Router ---
// 	mux := http.NewServeMux()

// 	// --- Middleware ---
// 	authMiddleware := middleware.AuthMiddleware(cfg) // Create instance of auth middleware

// 	// --- Routes ---
// 	// Basic health check
// 	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusOK)
// 		fmt.Fprintln(w, `{"status": "ok"}`)
// 	})

// 	// Placeholder for GraphQL Handler (using auth middleware)
// 	// graphQLHandler := /* ... Initialize your GraphQL handler here ... */
// 	// mux.Handle("/graphql", authMiddleware(graphQLHandler)) // Apply auth middleware

// 	// Add HTTP handlers for OAuth callbacks if needed (might not use auth middleware)
// 	// mux.HandleFunc("GET /auth/google/callback", /* Handler using authService */ )


// 	// --- HTTP Server ---
// 	server := &http.Server{
// 		Addr:    fmt.Sprintf(":%s", cfg.Port),
// 		Handler: mux, // Use the main router
// 		// Add timeouts for production readiness
// 		ReadTimeout:  5 * time.Second,
// 		WriteTimeout: 10 * time.Second,
// 		IdleTimeout:  120 * time.Second,
// 	}

// 	// --- Start Server Goroutine ---
// 	go func() {
// 		logger.Info("Server starting", slog.String("address", server.Addr))
// 		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
// 			logger.Error("Server failed to start", slog.String("error", err.Error()))
// 			os.Exit(1)
// 		}
// 	}()

// 	// --- Graceful Shutdown ---
// 	quit := make(chan os.Signal, 1)
// 	// Notify channel on SIGINT (Ctrl+C) or SIGTERM (sent by Docker/Kubernetes)
// 	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

// 	// Block until a signal is received
// 	sig := <-quit
// 	logger.Info("Shutdown signal received", slog.String("signal", sig.String()))

// 	// Create a context with timeout for shutdown
// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30-second timeout
// 	defer cancel()

// 	// Attempt graceful shutdown
// 	if err := server.Shutdown(ctx); err != nil {
// 		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
// 		os.Exit(1) // Exit with error if shutdown fails
// 	}

// 	logger.Info("Server exiting gracefully")
// }