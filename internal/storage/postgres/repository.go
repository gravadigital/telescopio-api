package postgres

import (
	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
)

// PaginationParams contains pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PaginatedResult contains paginated results
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// SearchParams contains search and filtering parameters
type SearchParams struct {
	Query    string            `json:"query"`
	Filters  map[string]string `json:"filters"`
	SortBy   string            `json:"sort_by"`
	SortDesc bool              `json:"sort_desc"`
}

// RepositoryTransaction defines transaction interface for atomic operations
type RepositoryTransaction interface {
	Commit() error
	Rollback() error
}

// TransactionalRepository provides transaction support
type TransactionalRepository interface {
	BeginTransaction() (RepositoryTransaction, error)
}

// EventRepository define los metodos para interactuar con los eventos en la DB.
type EventRepository interface {
	Create(event *event.Event) error
	GetByID(id string) (*event.Event, error)
	GetAll() ([]*event.Event, error)
	GetAllPaginated(params PaginationParams) (*PaginatedResult, error)
	GetByAuthor(authorID string) ([]*event.Event, error)
	GetByParticipant(participantID string) ([]*event.Event, error)
	Update(event *event.Event) error
	Delete(id string) error
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
	GetAllPaginated(params PaginationParams) (*PaginatedResult, error)
	Update(user *participant.User) error
	Delete(id string) error
	GetEventParticipants(eventID string) ([]*participant.User, error)
	GetEventParticipantsPaginated(eventID string, params PaginationParams) (*PaginatedResult, error)
}

// AttachmentRepository define los métodos para interactuar con los archivos adjuntos
type AttachmentRepository interface {
	Create(attachment *attachment.Attachment) error
	GetByID(id string) (*attachment.Attachment, error)
	GetByEventID(eventID string) ([]*attachment.Attachment, error)
	GetByEventIDPaginated(eventID string, params PaginationParams) (*PaginatedResult, error)
	GetByParticipantID(participantID string) ([]*attachment.Attachment, error)
	Update(attachment *attachment.Attachment) error
	UpdatePartial(id string, updates map[string]interface{}) error
	Delete(id string) error
	UpdateVoteCount(id string, count int) error
}

// VoteRepository define los métodos para interactuar con los votos
type VoteRepository interface {
	Create(vote *vote.Vote) error
	GetByID(id string) (*vote.Vote, error)
	GetByEventID(eventID string) ([]*vote.Vote, error)
	GetByEventIDPaginated(eventID string, params PaginationParams) (*PaginatedResult, error)
	GetByVoterID(voterID string) ([]*vote.Vote, error)
	GetByVoterIDPaginated(voterID string, params PaginationParams) (*PaginatedResult, error)
	GetByAttachmentID(attachmentID string) ([]*vote.Vote, error)
	Update(vote *vote.Vote) error
	Delete(id string) error
	HasVoted(eventID, voterID string) (bool, error)

	// Assignment methods
	CreateAssignment(assignment *vote.Assignment) error
	GetAssignmentsByEventID(eventID string) ([]*vote.Assignment, error)
	GetAssignmentsByEventIDPaginated(eventID string, params PaginationParams) (*PaginatedResult, error)
	GetAssignmentByParticipant(eventID, participantID string) (*vote.Assignment, error)
	UpdateAssignment(assignment *vote.Assignment) error
	DeleteAssignment(id string) error
}

// VotingConfigurationRepository define los métodos para interactuar con configuraciones de votación
type VotingConfigurationRepository interface {
	Create(config *vote.VotingConfiguration) error
	GetByID(id string) (*vote.VotingConfiguration, error)
	GetByEventID(eventID string) (*vote.VotingConfiguration, error)
	Update(config *vote.VotingConfiguration) error
	Delete(eventID string) error
	ValidateConfiguration(config *vote.VotingConfiguration) error
}

// VotingResultsRepository define los métodos para interactuar con resultados de votación
type VotingResultsRepository interface {
	Create(results *vote.VotingResults) error
	GetByID(id string) (*vote.VotingResults, error)
	GetByEventID(eventID string) (*vote.VotingResults, error)
	Update(results *vote.VotingResults) error
	Delete(eventID string) error
	CalculateResults(eventID string) (*vote.VotingResults, error)
	GetRankingByEvent(eventID string) ([]vote.AttachmentResult, error)
}

// RepositoryContainer provides access to all repositories
type RepositoryContainer interface {
	Events() EventRepository
	Users() UserRepository
	Attachments() AttachmentRepository
	Votes() VoteRepository
	VotingConfigurations() VotingConfigurationRepository
	VotingResults() VotingResultsRepository
	Health() error
	Close() error
}

// Aquí iría la configuración de la conexión a la DB (sqlx, gorm, etc.)
