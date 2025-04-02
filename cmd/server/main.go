// cmd/server/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log" // Use standard log ONLY for fatal startup errors
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings" // Added import
	"syscall"
	"time"

	"crm-communication-api/config"
	// "crm-communication-api/graph"
	"crm-communication-api/graph/generated" // This needs `go generate` AFTER schema is okay
	graphresolver "crm-communication-api/graph/resolver"
	"crm-communication-api/internal/auth"
	"crm-communication-api/internal/db" // CORRECTED import
	"crm-communication-api/internal/middleware"
	dbmodel "crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"
	"crm-communication-api/internal/websocket"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/google/uuid" // Added uuid import
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

const defaultPort = "8080"

// application struct holds ALL potential dependencies needed by the resolver + server setup
type application struct {
	config      *config.Config
	logger      *slog.Logger
	db          *gorm.DB
	repo        *repository.Repository // Concrete repo
	authSvc     auth.Service           // Auth Interface
	chatSvc     service.ChatService
	emailSvc    service.EmailService
	templateSvc service.TemplateService
	timelineSvc service.TimelineService
	userSvc     service.UserService   // Added field
	clientSvc   service.ClientService // Added field
	// interactionSvc service.InteractionService // Added field
	// taskSvc        service.TaskService       // Added field
	// reminderSvc    service.ReminderService   // Added field
	notificationSvc service.NotificationService // Added field
	schedulerSvc    service.SchedulerService    // Added field
	hub             *websocket.Hub
	router          *mux.Router // Added field
}

func main() {
	cfg := setupConfig()
	logger := setupLogger(cfg)
	slog.SetDefault(logger)

	db := setupDatabase(cfg, logger)
	runMigrations(db, logger)

	repo := repository.NewRepository(db)
	hub := websocket.NewHub()
	go hub.Run()

	app := &application{ // Initialize application struct
		config: cfg,
		logger: logger,
		db:     db,
		repo:   repo,
		hub:    hub,
	}

	app.setupServices(repo)                 // Initialize ALL services (using placeholders where needed)
	gqlHandler := app.setupGraphQLHandler() // Setup GraphQL handler
	app.setupRouter(gqlHandler)             // Setup Router

	app.startServer() // Start server
}

// --- Setup Functions ---

func setupConfig() *config.Config {
	cfg, err := config.LoadConfig() // CORRECTED: No arguments
	if err != nil {
		log.Fatalf("FATAL: Cannot load config: %v", err)
	}
	return cfg
}

func setupLogger(cfg *config.Config) *slog.Logger {
	logLevel := slog.LevelInfo
	if strings.ToLower(cfg.GoEnv) == "development" { // Use strings.ToLower
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	logger.Info("Logger initialized", slog.String("level", logLevel.String()))
	return logger
}

func setupDatabase(cfg *config.Config, logger *slog.Logger) *gorm.DB {
	dbConn, err := db.InitDB(cfg, logger) // CORRECTED: Use db.InitDB from internal/db
	if err != nil {
		logger.Error("Failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Database connection established")
	return dbConn // Return *gorm.DB
}

func runMigrations(db *gorm.DB, logger *slog.Logger) {
	logger.Info("Running database migrations...")
	// Ensure all models are included
	err := db.AutoMigrate(
		&dbmodel.User{}, &dbmodel.RefreshToken{}, &dbmodel.OAuthProvider{},
		&dbmodel.Client{}, &dbmodel.Conversation{}, &dbmodel.ConversationParticipant{},
		&dbmodel.Message{}, &dbmodel.MessageReadReceipt{},
		&dbmodel.Email{}, &dbmodel.EmailAttachment{}, &dbmodel.EmailTemplate{},
		&dbmodel.TimelineEvent{},
		// &dbmodel.Task{},    // Add Task model if/when defined
		// &dbmodel.Reminder{}, // Add Reminder model if/when defined
	)
	if err != nil {
		logger.Error("Database migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Database migrations completed")
}

// setupServices initializes all application services (using placeholders where necessary).
func (app *application) setupServices(repo *repository.Repository) {
	app.logger.Info("Initializing services...")

	// --- Auth Service (using Google for example) ---
	googleAuthConcrete, err := auth.NewGoogleAuthService(repo, app.config, app.logger) // Corrected constructor call
	if err != nil {
		app.logger.Error("Failed to initialize GoogleAuthService", slog.String("error", err.Error()))
		os.Exit(1)
	}
	app.authSvc = googleAuthConcrete // Assign concrete instance to interface field

	// --- Core Implemented/Required Services ---
	// app.templateSvc = NewTemplateService(repo, app.logger)   // Use placeholder constructor below
	// app.timelineSvc = NewTimelineService(repo, app.logger)   // Use placeholder constructor below

	// Placeholders for Notification and Scheduler (defined at end of file)
	notificationSvc := NewNotificationService(app.hub, app.logger)
	schedulerSvc := NewSimpleSchedulerService(app.logger)

	// EmailService
	app.emailSvc = service.NewEmailService(repo, repo, app.authSvc, repo, app.templateSvc, app.timelineSvc, app.logger) // Correct arguments

	// ChatService
	app.chatSvc = service.NewChatService(repo, repo, repo, app.timelineSvc, notificationSvc, schedulerSvc, app.logger) // Correct arguments

	// --- Initialize Other Services using Placeholders ---
	app.userSvc = NewUserService(repo, app.logger)     // Using placeholder
	app.clientSvc = NewClientService(repo, app.logger) // Using placeholder
	// app.interactionSvc = NewInteractionService(repo, app.logger) // Using placeholder
	// app.taskSvc = NewTaskService(repo, app.logger)             // Using placeholder
	// app.reminderSvc = NewReminderService(repo, app.logger)     // Using placeholder
	app.notificationSvc = notificationSvc // Assign placeholder instance
	app.schedulerSvc = schedulerSvc       // Assign placeholder instance

	app.logger.Info("Services initialized")
}

// setupGraphQLHandler initializes the gqlgen server.
func (app *application) setupGraphQLHandler() http.Handler {
	// Create the main GraphQL resolver, injecting all app services
	resolver := graphresolver.NewResolver( // CORRECTED: Use constructor
		app.chatSvc,
		app.emailSvc,
		app.templateSvc,
		app.timelineSvc,
		app.repo,
		app.authSvc,
		app.hub,
		app.config,
		// app.taskSvc,        // Pass initialized (placeholder) service
		// app.reminderSvc,    // Pass initialized (placeholder) service
		app.userSvc,   // Pass initialized (placeholder) service
		app.clientSvc, // Pass initialized (placeholder) service
		// app.interactionSvc, // Pass initialized (placeholder) service
	)

	// !! IMPORTANT !!
	// The next line will FAIL until `go generate ./...` is run in the `graph` directory.
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	app.logger.Info("GraphQL handler configured")
	return srv
}

// setupRouter configures the HTTP routes.
func (app *application) setupRouter(gqlHandler http.Handler) {
	router := mux.NewRouter()
	app.router = router

	// --- Middleware ---
	// ... (CORS and AuthMiddleware setup remains the same) ...
	c := cors.New(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions}, AllowedHeaders: []string{"Authorization", "Content-Type", "Accept"}, AllowCredentials: true, MaxAge: 300})
	router.Use(c.Handler)
	router.Use(middleware.AuthMiddleware(app.config))

	// --- Routes ---
	// ... (ping, query, playground, ws routes remain the same) ...
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { /* ... */ }).Methods(http.MethodGet)
	if gqlHandler != nil {
		router.Handle("/query", gqlHandler).Methods(http.MethodPost, http.MethodGet)
		if app.config.GoEnv == "development" {
			router.Handle("/", playground.Handler("GraphQL playground", "/query")).Methods(http.MethodGet)
		} else {
			router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }).Methods(http.MethodGet)
		}
	} else { /* ... handler not initialized message ... */
	}
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { websocket.ServeWs(app.hub, w, r) })

	// --- OAuth Routes ---
	// Get concrete service, check if it's the right type AND configured
	googleAuthSvcConcrete, ok := app.authSvc.(*auth.GoogleAuthService)
	if ok && googleAuthSvcConcrete != nil && googleAuthSvcConcrete.IsConfigured() { // CORRECTED: Check IsConfigured()
		authRouter := router.PathPrefix("/auth").Subrouter() // Routes without default auth middleware

		// Login route
		authRouter.HandleFunc("/google/login", func(w http.ResponseWriter, r *http.Request) {
			url := googleAuthSvcConcrete.GetAuthCodeURL() // CORRECTED: Call GetAuthCodeURL
			if url == "" {                                // Check if URL is empty (meaning not configured)
				http.Error(w, "Google Auth not configured", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		}).Methods(http.MethodGet)

		// Callback route
		authRouter.HandleFunc("/google/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			state := r.URL.Query().Get("state") // Get state if using it for verification
			if code == "" {
				http.Error(w, "Code is required", http.StatusBadRequest)
				return
			}
			if state == "" {
				http.Error(w, "State is required", http.StatusBadRequest)
				return
			} // Require state

			// TODO: Add state verification logic here using googleAuthSvcConcrete.verifyOAuthState(state) if implemented

			payload, err := googleAuthSvcConcrete.AuthenticateGoogleUser(r.Context(), code)
			if err != nil {
				http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
				return
			}

			// Set cookies
			http.SetCookie(w, &http.Cookie{Name: "accessToken", Value: payload.AccessToken, Path: "/", HttpOnly: true, Secure: app.config.GoEnv == "production", SameSite: http.SameSiteLaxMode, MaxAge: int(app.config.JWTExpiry.Seconds())})
			http.SetCookie(w, &http.Cookie{Name: "refreshToken", Value: payload.RefreshToken, Path: "/", HttpOnly: true, Secure: app.config.GoEnv == "production", SameSite: http.SameSiteLaxMode, MaxAge: int(app.config.RefreshTokenExpiry.Seconds())})
			http.Redirect(w, r, app.config.BaseURL, http.StatusTemporaryRedirect)
		}).Methods(http.MethodGet)
		app.logger.Info("Google OAuth routes configured")
	} else {
		app.logger.Warn("GoogleAuthService not available or not configured, skipping Google OAuth route setup")
	}

	app.logger.Info("Router configured")
	// No return needed as router is stored in app struct
}

// startServer starts the HTTP server.
func (app *application) startServer() {
	server := &http.Server{
		Addr:        fmt.Sprintf(":%s", app.config.Port),
		Handler:     app.router, // Use router from app struct
		ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 120 * time.Second,
	}
	app.logger.Info("Server starting", slog.String("address", server.Addr))
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	app.waitForShutdown(server, serverErr)
}

// waitForShutdown handles graceful server shutdown.
func (app *application) waitForShutdown(server *http.Server, serverErr chan error) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("Server failed unexpectedly", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-quit:
		app.logger.Info("Shutdown signal received", slog.String("signal", sig.String()))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
	app.logger.Info("Server exiting gracefully")
}

// --- TEMPORARY Placeholder Service Constructors (REMOVE / MOVE TO internal/service/*_service.go) ---

type SimpleSchedulerService struct{ logger *slog.Logger }

func NewSimpleSchedulerService(logger *slog.Logger) service.SchedulerService {
	return &SimpleSchedulerService{logger: logger}
}
func (s *SimpleSchedulerService) ScheduleReminder(ctx context.Context, rt time.Time, u uuid.UUID, m string, c uuid.UUID) error {
	s.logger.Info("Placeholder: ScheduleReminder called", slog.Time("remindAt", rt), slog.String("userID", u.String()))
	return nil
}

type NotificationService struct {
	hub    *websocket.Hub
	logger *slog.Logger
}

func NewNotificationService(hub *websocket.Hub, logger *slog.Logger) service.NotificationService {
	return &NotificationService{hub: hub, logger: logger.With(slog.String("service", "NotificationService"))}
}
func (s *NotificationService) BroadcastNewMessage(ctx context.Context, cID uuid.UUID, msg *dbmodel.Message) error {
	s.logger.Debug("Placeholder: BroadcastNewMessage")
	s.hub.Broadcast <- websocket.Message{RoomID: cID.String(), Type: "new_message", Payload: msg}
	return nil
}
func (s *NotificationService) BroadcastMessageRead(ctx context.Context, cID uuid.UUID, payload map[string]interface{}) error {
	s.logger.Debug("Placeholder: BroadcastMessageRead")
	s.hub.Broadcast <- websocket.Message{RoomID: cID.String(), Type: "message_read", Payload: payload}
	return nil
}
func (s *NotificationService) GetNewMessageChannel(ctx context.Context, cID uuid.UUID) (<-chan *dbmodel.Message, error) {
	ch := make(chan *dbmodel.Message) /* Need proper hub logic */
	return ch, fmt.Errorf("Placeholder: GetNewMessageChannel")
}

type userService struct {
	repo   repository.AuthRepository
	logger *slog.Logger
}

func NewUserService(repo repository.AuthRepository, logger *slog.Logger) service.UserService {
	return &userService{repo: repo, logger: logger}
}
func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*dbmodel.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

type clientService struct {
	repo   repository.ClientRepository
	logger *slog.Logger
}

func NewClientService(repo repository.ClientRepository, logger *slog.Logger) service.ClientService {
	return &clientService{repo: repo, logger: logger}
}
func (s *clientService) GetClient(ctx context.Context, id uuid.UUID) (*dbmodel.Client, error) {
	return s.repo.GetClientByID(ctx, id)
}
func (s *clientService) CreateClient(ctx context.Context, client *dbmodel.Client) (*dbmodel.Client, error) {
	err := s.repo.CreateClient(ctx, client)
	return client, err
}
func (s *clientService) GetAllClients(ctx context.Context) ([]dbmodel.Client, error) {
	return s.repo.GetAllClients(ctx)
}
func (s *clientService) UpdateClient(ctx context.Context, client *dbmodel.Client) (*dbmodel.Client, error) {
	err := s.repo.UpdateClient(ctx, client)
	return client, err
}

type interactionService struct {
	repo   repository.TimelineRepository
	logger *slog.Logger
}

// func NewInteractionService(repo repository.TimelineRepository, logger *slog.Logger) service.InteractionService {
// 	return &interactionService{repo: repo, logger: logger}
// }
// func (s *interactionService) RecordInteraction(ctx context.Context, interaction dbmodel.Interaction) error { /* TODO */
// 	return nil
// }

type taskService struct {
	repo   repository.Repository
	logger *slog.Logger
} // Needs specific TaskRepository interface later
// func NewTaskService(repo repository.Repository, logger *slog.Logger) service.TaskService {
// 	return &taskService{repo: repo, logger: logger}
// }

// TODO: Implement TaskService methods

type reminderService struct {
	repo   repository.Repository
	logger *slog.Logger
} // Needs specific ReminderRepository interface later
// func NewReminderService(repo repository.Repository, logger *slog.Logger) service.ReminderService {
// 	return &reminderService{repo: repo, logger: logger}
// }

// TODO: Implement ReminderService methods

// Assume TemplateService constructor exists in its file:
// func NewTemplateService(repo repository.TemplateRepository, logger *slog.Logger) service.TemplateService { ... }
// Assume TimelineService constructor exists in its file:
// func NewTimelineService(repo repository.TimelineRepository, logger *slog.Logger) service.TimelineService { ... }

// Helper in auth/google.go needed:
/*
func (s *GoogleAuthService) IsConfigured() bool {
    return s.googleOAuthConfig != nil && s.googleOAuthConfig.ClientID != "" && s.googleOAuthConfig.ClientSecret != ""
}
*/
