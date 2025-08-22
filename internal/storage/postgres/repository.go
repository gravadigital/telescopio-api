package postgres

import (
	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
)

// EventRepository define los metodos para interactuar con los eventos en la DB.
type EventRepository interface {
	Create(event *event.Event) error
	GetByID(id string) (*event.Event, error)
	GetAll() ([]*event.Event, error)
	GetByAuthor(authorID string) ([]*event.Event, error)
	GetByParticipant(participantID string) ([]*event.Event, error)
	UpdateStage(eventID string, stage event.Stage) error
	AddParticipant(eventID, userID string) error
	RemoveParticipant(eventID, userID string) error
}

// UserRepository define los métodos para interactuar con los usuarios en la DB.
type UserRepository interface {
	Create(user *participant.User) error
	GetByID(id string) (*participant.User, error)
	GetByEmail(email string) (*participant.User, error)
	GetAll() ([]*participant.User, error)
	Update(user *participant.User) error
	GetEventParticipants(eventID string) ([]*participant.User, error)
}

// AttachmentRepository define los métodos para interactuar con los archivos adjuntos
type AttachmentRepository interface {
	Create(attachment *attachment.Attachment) error
	GetByID(id string) (*attachment.Attachment, error)
	GetByEventID(eventID string) ([]*attachment.Attachment, error)
	GetByParticipantID(participantID string) ([]*attachment.Attachment, error)
	Delete(id string) error
	UpdateVoteCount(id string, count int) error
}

// VoteRepository define los métodos para interactuar con los votos
type VoteRepository interface {
	Create(vote *vote.Vote) error
	GetByID(id string) (*vote.Vote, error)
	GetByEventID(eventID string) ([]*vote.Vote, error)
	GetByVoterID(voterID string) ([]*vote.Vote, error)
	GetByAttachmentID(attachmentID string) ([]*vote.Vote, error)
	HasVoted(eventID, voterID string) (bool, error)
	GetEventResults(eventID string) (map[string]int, error) // attachment_id -> vote_count
}

// Aquí iría la configuración de la conexión a la DB (sqlx, gorm, etc.)
