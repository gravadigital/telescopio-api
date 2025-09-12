package vote

import "github.com/gravadigital/telescopio-api/internal/domain/common"

// Repository interfaces for the voting service

type VoteRepository interface {
	Create(vote *Vote) error
	GetByID(id string) (*Vote, error)
	GetByEventID(eventID string) ([]*Vote, error)
	GetByVoterID(voterID string) ([]*Vote, error)
	GetAssignmentsByEventID(eventID string) ([]*Assignment, error)
	CreateAssignment(assignment *Assignment) error
	GetAssignmentByParticipant(eventID, participantID string) (*Assignment, error)
	UpdateAssignment(assignment *Assignment) error
}

// AttachmentRepository uses interface to avoid circular imports
type AttachmentRepository interface {
	GetByID(id string) (common.AttachmentInterface, error)
	GetByEventID(eventID string) ([]common.AttachmentInterface, error)
}

// UserRepository uses interface to avoid circular imports
type UserRepository interface {
	GetByID(id string) (common.UserInterface, error)
}
