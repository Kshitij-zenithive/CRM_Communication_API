// internal/model/models.go
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq" // Added for text array support in Email recipients
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// --- Core Entities ---

// User represents a system user (internal agent, employee)
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Email     string    `gorm:"type:varchar(100);unique;not null" json:"email"`
	Password  string    `gorm:"type:varchar(100)" json:"-"`                           // Not exposed
	Avatar    string    `gorm:"type:varchar(255)" json:"avatar"`                      // URL or path
	Role      string    `gorm:"type:varchar(50);default:'user';not null" json:"role"` // e.g., "admin", "agent", "manager"
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`

	// GORM Relations (primarily for service layer use, hidden from basic JSON)
	// MessagesSent            []Message                 `gorm:"foreignKey:SenderID" json:"-"` // Messages sent by this user
	// EmailAssociations       []Email                   `gorm:"foreignKey:UserID" json:"-"`   // Emails involving this user (sender/internal recipient)
	ConversationParticipations []ConversationParticipant `gorm:"foreignKey:UserID" json:"-"` // Conversations this user is part of

	RefreshTokens  []RefreshToken  `gorm:"foreignKey:UserID..."`
	OAuthProviders []OAuthProvider `gorm:"foreignKey:UserID..."`
}

// Client represents an external client/customer
type Client struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Email     string    `gorm:"type:varchar(100);unique;not null" json:"email"`
	Company   *string   `gorm:"type:varchar(100)" json:"company"` // Optional
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`

	// GORM Relations
	// Emails        []Email        `gorm:"foreignKey:ClientID" json:"-"` // Emails linked to this client
	// Conversations []Conversation `gorm:"foreignKey:ClientID" json:"-"` // Client-specific conversations
}

// Add this struct definition:
// RefreshToken stores refresh tokens associated with a user.
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`          // Foreign key to User
	Token     string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"` // The refresh token string (unique, hidden)
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // Optional: Uncomment if soft delete is desired for tokens

	// GORM Relations
	// User      User      `gorm:"foreignKey:UserID" json:"-"` // Belongs to User (optional preload)
}

// Add this struct definition:
// OAuthProvider stores external OAuth credentials linked to a user.
type OAuthProvider struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index:idx_user_provider,unique" json:"userId"`          // Foreign key to User, part of composite unique index
	Provider     string    `gorm:"type:varchar(50);not null;index:idx_user_provider,unique" json:"provider"` // e.g., "google", part of composite unique index
	ProviderID   string    `gorm:"type:varchar(255);not null;index" json:"providerId"`                       // User ID from the provider
	AccessToken  string    `gorm:"type:text;not null" json:"-"`                                              // Access Token (hidden)
	RefreshToken string    `gorm:"type:text" json:"-"`                                                       // Refresh Token (hidden, might be empty)
	ExpiresAt    time.Time `json:"expiresAt"`                                                                // Expiry of the AccessToken
	CreatedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`
	// DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // Optional: Uncomment if soft delete is desired

	// GORM Relations
	// User         User      `gorm:"foreignKey:UserID" json:"-"` // Belongs to User (optional preload)
}

// --- Communication Channels ---

// Conversation represents a distinct thread of communication
type Conversation struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Type      string     `gorm:"type:varchar(50);not null" json:"type"` // E.g., ConversationTypeClient, ConversationTypeDirect, ConversationTypeGroup
	Name      *string    `gorm:"type:varchar(100)" json:"name"`         // Optional name, primarily for group chats
	ClientID  *uuid.UUID `gorm:"type:uuid;index" json:"clientId"`       // Link to Client if it's a client-specific conversation
	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`

	// GORM Relations
	Participants []ConversationParticipant `gorm:"foreignKey:ConversationID" json:"participants"` // Eager load participants often useful
	Messages     []Message                 `gorm:"foreignKey:ConversationID" json:"-"`            // Usually loaded on demand
	Client       *Client                   `gorm:"foreignKey:ClientID" json:"client,omitempty"`   // Optional client relation
}

// ConversationParticipant links Users to Conversations
type ConversationParticipant struct {
	// Using composite primary key for simplicity and uniqueness guarantee
	ConversationID uuid.UUID `gorm:"type:uuid;primaryKey" json:"conversationId"`
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"userId"`
	JoinedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"joinedAt"`

	// GORM Relations (ensure correct foreign key refs if not using composite PK)
	// If using composite PK, GORM might infer relations, but explicit tags can help clarity
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"-"`
	User         User         `gorm:"foreignKey:UserID" json:"user"` // Often useful to include User info
}

// Message represents a single message within a Conversation (Chat)
type Message struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ConversationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"conversationId"`
	SenderID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"senderId"` // User who sent it
	Content        string         `gorm:"type:text;not null" json:"content"`
	Type           MessageType    `gorm:"type:varchar(50);not null;default:'TEXT'" json:"type"` // Type of message
	Metadata       datatypes.JSON `gorm:"type:jsonb"`                                           // Switched to gorm's JSON type
	CreatedAt      time.Time      `gorm:"default:CURRENT_TIMESTAMP;index" json:"createdAt"`     // Index for sorting
	UpdatedAt      time.Time      `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"` // Soft delete support

	// GORM Relations
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"-"` // Belongs to conversation
	Sender       User         `gorm:"foreignKey:SenderID" json:"sender"`  // Message sender details
}

// MessageReadReceipt tracks when a user has read a specific message.
type MessageReadReceipt struct {
	MessageID uuid.UUID `gorm:"type:uuid;primaryKey"` // Composite primary key with UserID
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"` // Composite primary key with MessageID
	ReadAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	// GORM Relations (optional but good practice)
	Message Message `gorm:"foreignKey:MessageID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User    User    `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// Email represents an email communication linked to a Client and User
type Email struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ClientID   *uuid.UUID     `gorm:"type:uuid;index" json:"clientId"`           // Nullable, as some emails might not be client-related
	UserID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"userId"`    // Internal user associated (sender/recipient context)
	Provider   string         `gorm:"type:varchar(50)" json:"provider"`          // e.g., "gmail", "manual" - Source system
	ProviderID string         `gorm:"type:varchar(255);index" json:"providerId"` // ID from the source system (e.g., Gmail message ID), potentially unique per provider
	ThreadID   string         `gorm:"type:varchar(255);index" json:"threadId"`   // Provider's thread ID
	Subject    string         `gorm:"type:varchar(255)" json:"subject"`
	From       string         `gorm:"type:varchar(255);not null" json:"from"`     // Single From address
	To         pq.StringArray `gorm:"type:text[];not null" json:"to"`             // List of To recipients
	Cc         pq.StringArray `gorm:"type:text[]" json:"cc,omitempty"`            // List of Cc recipients
	Bcc        pq.StringArray `gorm:"type:text[]" json:"bcc,omitempty"`           // List of Bcc recipients (maybe store hashed/masked?)
	BodyHTML   string         `gorm:"type:text" json:"bodyHtml,omitempty"`        // HTML content
	BodyText   string         `gorm:"type:text" json:"bodyText,omitempty"`        // Plain text content
	Snippet    string         `gorm:"type:text" json:"snippet,omitempty"`         // Short text snippet
	SentAt     *time.Time     `gorm:"index" json:"sentAt,omitempty"`              // Timestamp when sent (if known)
	ReceivedAt *time.Time     `gorm:"index" json:"receivedAt,omitempty"`          // Timestamp when received (if known)
	IsRead     bool           `gorm:"default:false" json:"isRead"`                // Read status for the internal user
	Direction  EmailDirection `gorm:"type:varchar(20);not null" json:"direction"` // "INBOUND" or "OUTBOUND" relative to the system
	CreatedAt  time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`

	// GORM Relations
	Attachments []EmailAttachment `gorm:"foreignKey:EmailID" json:"attachments,omitempty"`
	Client      *Client           `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	User        User              `gorm:"foreignKey:UserID" json:"-"` // Associated internal user
}

// EmailAttachment represents a file attached to an Email
type EmailAttachment struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	EmailID     uuid.UUID `gorm:"type:uuid;not null;index" json:"emailId"`
	Filename    string    `gorm:"type:varchar(255);not null" json:"filename"`
	MimeType    string    `gorm:"type:varchar(100)" json:"mimeType"`
	Size        int64     `gorm:"type:bigint;not null" json:"size"`
	StoragePath string    `gorm:"type:varchar(512);not null" json:"-"` // Path in storage (S3, GCS, local), NOT exposed directly
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`

	// GORM Relations
	Email Email `gorm:"foreignKey:EmailID" json:"-"`
}

// EmailTemplate represents a reusable template for emails
type EmailTemplate struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"type:varchar(255);unique;not null" json:"name"` // Unique name for easy lookup
	Subject   string    `gorm:"type:varchar(255);not null" json:"subject"`     // Template subject (can contain placeholders)
	Body      string    `gorm:"type:text;not null" json:"body"`                // Template body (HTML or text, with placeholders)
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`
}

// --- Enums and Helper Types ---

// ConversationType defines the types of conversations
type ConversationType string

const (
	ConversationTypeDirect ConversationType = "DIRECT" // DM between internal users
	ConversationTypeGroup  ConversationType = "GROUP"  // Group chat between internal users
	ConversationTypeClient ConversationType = "CLIENT" // Chat involving a client (potentially linking ClientID)
)

// MessageType defines the purpose or type of a message
type MessageType string

const (
	MessageTypeText          MessageType = "TEXT"            // Standard user-generated text message
	MessageTypeSystem        MessageType = "SYSTEM"          // System notification (e.g., user joined, convo created)
	MessageTypeCommandOutput MessageType = "COMMAND_OUTPUT"  // Output/confirmation from a slash command
	MessageTypeEmailInThread MessageType = "EMAIL_IN_THREAD" // Represents an email shown within the chat timeline
)

// EmailDirection indicates email flow relative to the system
type EmailDirection string

const (
	EmailDirectionInbound  EmailDirection = "INBOUND"
	EmailDirectionOutbound EmailDirection = "OUTBOUND"
)

type TimelineEvent struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`            // User who performed or is related to the event
	ClientID    *uuid.UUID     `gorm:"type:uuid;index" json:"clientId,omitempty"`         // Optional client related to the event
	EventType   string         `gorm:"type:varchar(100);not null;index" json:"eventType"` // e.g., "MESSAGE_SENT", "EMAIL_RECEIVED", "REMINDER_SET", "LOGIN"
	Description string         `gorm:"type:text" json:"description"`                      // Human-readable description
	RelatedID   *uuid.UUID     `gorm:"type:uuid;index" json:"relatedId,omitempty"`        // Optional ID of related entity (Message, Email, Conversation, User)
	RelatedType *string        `gorm:"type:varchar(50)" json:"relatedType,omitempty"`     // Type of the related entity ("message", "email", etc.)
	Metadata    datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`              // Additional structured data
	EventTime   time.Time      `gorm:"not null;index" json:"eventTime"`                   // Timestamp when the event occurred
	CreatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP;autoUpdateTime" json:"updatedAt"`

	// GORM Relations (optional, depending on query needs)
	// User        User     `gorm:"foreignKey:UserID" json:"user"`
	// Client      *Client  `gorm:"foreignKey:ClientID" json:"client,omitempty"`
}

// BeforeCreate hook for TimelineEvent (if needed, UUID handled by default)
func (te *TimelineEvent) BeforeCreate(tx *gorm.DB) (err error) {
	// if te.ID == uuid.Nil { // Handled by default:uuid_generate_v4()
	// 	te.ID = uuid.New()
	// }
	return
}

// JSONB is a helper type for handling jsonb data in GORM/SQL
type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 || string(j) == "null" {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte) // GORM uses []byte for JSONB
	if !ok {
		// Handle string type as well if necessary, depending on driver
		strVal, strOk := value.(string)
		if !strOk {
			return errors.New("type assertion to []byte or string failed")
		}
		s = []byte(strVal)
	}
	*j = append((*j)[0:0], s...)
	return nil
}

// MarshalJSON returns j as the JSON encoding of j.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets *j to a copy of data.
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONB: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// --- GORM Hooks ---

// BeforeCreate hook for User to hash password
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// UUID is handled by default:uuid_generate_v4()
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// --- Password Helpers ---

// ComparePassword compares a plaintext password with the user's hashed password
func (u *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

// SetPassword hashes and sets a new password for the user (useful for updates)
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}
