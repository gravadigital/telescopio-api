package common

import "github.com/google/uuid"

// SharedEvent represents the minimal Event structure used across domains
type SharedEvent struct {
	ID   uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name string    `json:"name"`
}

// SharedUser represents the minimal User structure used across domains
type SharedUser struct {
	ID   uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name string    `json:"name"`
}

// SharedAttachment represents the minimal Attachment structure used across domains
type SharedAttachment struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	OriginalName  string    `json:"original_name"`
	ParticipantID uuid.UUID `json:"participant_id"`
}

// SharedVote represents the minimal Vote structure used across domains
type SharedVote struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	AttachmentID uuid.UUID `json:"attachment_id"`
}

// Interfaces for type safety without circular imports

type AttachmentInterface interface {
	GetID() uuid.UUID
	GetOriginalName() string
	GetParticipantID() uuid.UUID
}

type UserInterface interface {
	GetID() uuid.UUID
	GetName() string
}

type EventInterface interface {
	GetID() uuid.UUID
	GetName() string
}
