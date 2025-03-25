// graph/resolver.go (Corrected dependencies)

package graph

import (
	"crm-communication-api/config"
	"crm-communication-api/internal/auth"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/service"
	"crm-communication-api/internal/websocket"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	ChatService        *service.ChatService
	EmailService       *service.EmailService
	InteractionService *service.InteractionService
	TaskService        *service.TaskService     // Add TaskService
	ReminderService    *service.ReminderService // Add ReminderService
	UserService        *service.UserService
	TemplateService    *service.TemplateService
	AuthService        *auth.GoogleAuthService
	Repository         *repository.Repository
	Hub                *websocket.Hub
	Config             *config.Config
}
