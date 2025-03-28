// cmd/server/main.go (Corrected)

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"crm-communication-api/config"
	"crm-communication-api/database"
	"crm-communication-api/graph"           // Corrected import
	"crm-communication-api/graph/generated" // Corrected import
	"crm-communication-api/internal/auth"
	"crm-communication-api/internal/middleware"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"
	"crm-communication-api/internal/websocket"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/rs/cors" // Import the cors package
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database connection
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize repository
	repo := repository.NewRepository(db.DB)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize services
	googleAuthService := auth.NewGoogleAuthService(cfg, repo)                        // Pass repository
	templateService := service.NewTemplateService(repo)                              // Use correct service
	emailService := service.NewEmailService(repo, googleAuthService.GetGmailService) //Corrected
	chatService := service.NewChatService(repo, hub)
	taskService := service.NewTaskService(repo)         // Corrected
	reminderService := service.NewReminderService(repo) // Corrected
	userService := service.NewUserService(repo)
	interactionService := service.NewInteractionService(repo)

	// Initialize GraphQL resolver
	resolver := &graph.Resolver{
		ChatService:        chatService,
		EmailService:       emailService,
		InteractionService: interactionService,
		TaskService:        taskService,     // Add TaskService
		ReminderService:    reminderService, // Add ReminderService
		UserService:        userService,     // Add UserService
		TemplateService:    templateService, // Corrected
		AuthService:        googleAuthService,
		Repository:         repo,
		Hub:                hub,
		Config:             cfg,
	}

	// Create a new GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	// Set up HTTP router
	r := mux.NewRouter()

	// CORS middleware setup (before any other middleware!)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // !!! IMPORTANT: In production, be specific!
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	r.Use(c.Handler) // Apply CORS to all routes

	// Middleware
	r.Use(middleware.AuthMiddleware(cfg))

	// Google OAuth routes (no middleware here)
	r.HandleFunc("/auth/google/login", func(w http.ResponseWriter, r *http.Request) {
		url := googleAuthService.GetAuthCodeURL()
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})

	r.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Authorization code is required", http.StatusBadRequest)
			return
		}

		jwtToken, refreshToken, err := googleAuthService.AuthenticateUser(r.Context(), code)
		if err != nil {
			http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
			return
		}
		//set cookies
		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    jwtToken,
			Path:     "/",
			HttpOnly: true,                 // Important for security
			Secure:   false,                // Set to true in production if using HTTPS
			SameSite: http.SameSiteLaxMode, //CSRF
			// Expires: ,  // Set expiry appropriately
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			// Expires: ,
		})

		// Redirect to frontend with tokens (consider using cookies or local storage)
		http.Redirect(w, r, cfg.BaseURL, http.StatusTemporaryRedirect) // Redirect to frontend
	})

	// GraphQL endpoint
	r.Handle("/query", srv)

	// GraphQL playground (for development)
	r.Handle("/playground", playground.Handler("GraphQL playground", "/query"))

	// WebSocket endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	})

	// Start HTTP server with graceful shutdown
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second, // Good practice
		WriteTimeout: 15 * time.Second, // Good practice
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a context with a timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
