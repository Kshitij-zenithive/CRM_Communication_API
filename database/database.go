// database/database.go
package database

import (
	"fmt"
	"log"

	"crm-communication-api/config"
	"crm-communication-api/internal/model"

	"gorm.io/driver/postgres" // Or your preferred database driver
	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
}

func Connect(cfg *config.Config) (*DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto Migrate (Consider using a dedicated migration tool)
	if err := db.AutoMigrate(
		&model.User{},
		&model.Client{},
		&model.Conversation{},            // Add Conversation
		&model.ConversationParticipant{}, // Add ConversationParticipant
		&model.Message{},
		&model.MessageMention{},
		&model.Email{},
		&model.EmailAttachment{},
		&model.TimelineEvent{},
		&model.OAuthProvider{},
		&model.RefreshToken{},
		&model.Task{},     // Add Task
		&model.Reminder{}, // Add Reminder
		&model.EmailTemplate{},
	); err != nil {
		log.Fatalf("Failed auto migration %v", err)
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return &DB{db}, nil
}
