// graph/resolver.go
package resolver

import (
	"crm-communication-api/config"
	"crm-communication-api/internal/auth"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"
	websocket "crm-communication-api/internal/websocket"
)

// Resolver is the root resolver struct holding dependencies.
// Its fields match the services needed by the resolver methods.
type Resolver struct {
	ChatService     service.ChatService // Use interface types
	EmailService    service.EmailService
	TemplateService service.TemplateService
	TimelineService service.TimelineService
	AuthService     auth.Service           // Use interface type
	Repository      *repository.Repository // Keep concrete if specific repo methods needed directly? Or use interfaces? Using concrete for now.
	Hub             *websocket.Hub
	Config          *config.Config
	// Add fields for other services ONLY if they are directly called by resolvers
	// UserSvc         service.UserService
	// ClientSvc       service.ClientService
	// InteractionSvc  service.InteractionService
	// TaskSvc         service.TaskService
	// ReminderSvc     service.ReminderService
}

// NewResolver is the constructor for the main Resolver.
// It should accept all dependencies needed by any resolver method.
// Pass nil for services that aren't implemented yet if necessary.
func NewResolver(
	chatSvc service.ChatService,
	emailSvc service.EmailService,
	templateSvc service.TemplateService,
	timelineSvc service.TimelineService,
	repo *repository.Repository, // Accepting concrete repo
	authSvc auth.Service, // Accepting auth interface
	hub *websocket.Hub,
	cfg *config.Config,
	// taskSvc service.TaskService, // Placeholder arg
	// reminderSvc service.ReminderService, // Placeholder arg
	userSvc service.UserService, // Placeholder arg
	clientSvc service.ClientService, // Placeholder arg
	// interactionSvc service.InteractionService, // Placeholder arg
) *Resolver {
	return &Resolver{
		ChatService:     chatSvc,
		EmailService:    emailSvc,
		TemplateService: templateSvc,
		TimelineService: timelineSvc,
		Repository:      repo,
		AuthService:     authSvc,
		Hub:             hub,
		Config:          cfg,
		// Assign other services passed in
		// TaskSvc:         taskSvc,
		// ReminderSvc:     reminderSvc,
		// UserSvc:         userSvc,
		// ClientSvc:       clientSvc,
		// InteractionSvc:  interactionSvc,
	}
}
