package common

import "github.com/google/uuid"

type EventInterface interface {
	GetID() uuid.UUID
	GetName() string
}

type UserInterface interface {
	GetID() uuid.UUID
	GetName() string
}

type AttachmentInterface interface {
	GetID() uuid.UUID
	GetOriginalName() string
	GetParticipantID() uuid.UUID
}

type VoteInterface interface {
	GetID() uuid.UUID
	GetAttachmentID() uuid.UUID
}

type EntityReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name,omitempty"`
}
