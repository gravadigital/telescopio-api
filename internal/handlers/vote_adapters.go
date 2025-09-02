package handlers

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/domain/common"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

// VoteRepositoryAdapter adapts postgres.VoteRepository to vote.VoteRepository
// with enhanced error handling, logging, and validation
type VoteRepositoryAdapter struct {
	repo postgres.VoteRepository
	log  *log.Logger
}

func NewVoteRepositoryAdapter(repo postgres.VoteRepository) *VoteRepositoryAdapter {
	return &VoteRepositoryAdapter{
		repo: repo,
		log:  logger.Repository("vote_adapter"),
	}
}

func (a *VoteRepositoryAdapter) Create(vote *vote.Vote) error {
	if vote == nil {
		a.log.Error("attempt to create nil vote")
		return fmt.Errorf("vote cannot be nil")
	}

	a.log.Debug("creating vote", "vote_id", vote.ID, "event_id", vote.EventID, "voter_id", vote.VoterID)

	if err := a.repo.Create(vote); err != nil {
		a.log.Error("failed to create vote", "vote_id", vote.ID, "error", err)
		return fmt.Errorf("failed to create vote: %w", err)
	}

	a.log.Debug("vote created successfully", "vote_id", vote.ID)
	return nil
}

func (a *VoteRepositoryAdapter) GetByID(id string) (*vote.Vote, error) {
	if id == "" {
		a.log.Error("empty vote ID provided")
		return nil, fmt.Errorf("vote ID cannot be empty")
	}

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		a.log.Error("invalid vote ID format", "id", id, "error", err)
		return nil, fmt.Errorf("invalid vote ID format: %w", err)
	}

	a.log.Debug("retrieving vote by ID", "vote_id", id)

	result, err := a.repo.GetByID(id)
	if err != nil {
		a.log.Error("failed to retrieve vote", "vote_id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve vote %s: %w", id, err)
	}

	a.log.Debug("vote retrieved successfully", "vote_id", id)
	return result, nil
}

func (a *VoteRepositoryAdapter) GetByEventID(eventID string) ([]*vote.Vote, error) {
	if eventID == "" {
		a.log.Error("empty event ID provided")
		return nil, fmt.Errorf("event ID cannot be empty")
	}

	// Validate UUID format
	if _, err := uuid.Parse(eventID); err != nil {
		a.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	a.log.Debug("retrieving votes by event ID", "event_id", eventID)

	result, err := a.repo.GetByEventID(eventID)
	if err != nil {
		a.log.Error("failed to retrieve votes by event", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve votes for event %s: %w", eventID, err)
	}

	a.log.Debug("votes retrieved successfully", "event_id", eventID, "count", len(result))
	return result, nil
}

func (a *VoteRepositoryAdapter) GetByVoterID(voterID string) ([]*vote.Vote, error) {
	if voterID == "" {
		a.log.Error("empty voter ID provided")
		return nil, fmt.Errorf("voter ID cannot be empty")
	}

	// Validate UUID format
	if _, err := uuid.Parse(voterID); err != nil {
		a.log.Error("invalid voter ID format", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("invalid voter ID format: %w", err)
	}

	a.log.Debug("retrieving votes by voter ID", "voter_id", voterID)

	result, err := a.repo.GetByVoterID(voterID)
	if err != nil {
		a.log.Error("failed to retrieve votes by voter", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("failed to retrieve votes for voter %s: %w", voterID, err)
	}

	a.log.Debug("votes retrieved successfully", "voter_id", voterID, "count", len(result))
	return result, nil
}

func (a *VoteRepositoryAdapter) GetAssignmentsByEventID(eventID string) ([]*vote.Assignment, error) {
	if eventID == "" {
		a.log.Error("empty event ID provided for assignments")
		return nil, fmt.Errorf("event ID cannot be empty")
	}

	// Validate UUID format
	if _, err := uuid.Parse(eventID); err != nil {
		a.log.Error("invalid event ID format for assignments", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	a.log.Debug("retrieving assignments by event ID", "event_id", eventID)

	result, err := a.repo.GetAssignmentsByEventID(eventID)
	if err != nil {
		a.log.Error("failed to retrieve assignments by event", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve assignments for event %s: %w", eventID, err)
	}

	a.log.Debug("assignments retrieved successfully", "event_id", eventID, "count", len(result))
	return result, nil
}

func (a *VoteRepositoryAdapter) CreateAssignment(assignment *vote.Assignment) error {
	if assignment == nil {
		a.log.Error("attempt to create nil assignment")
		return fmt.Errorf("assignment cannot be nil")
	}

	a.log.Debug("creating assignment", 
		"assignment_id", assignment.ID, 
		"event_id", assignment.EventID, 
		"participant_id", assignment.ParticipantID)

	if err := a.repo.CreateAssignment(assignment); err != nil {
		a.log.Error("failed to create assignment", 
			"assignment_id", assignment.ID, 
			"event_id", assignment.EventID,
			"error", err)
		return fmt.Errorf("failed to create assignment: %w", err)
	}

	a.log.Debug("assignment created successfully", "assignment_id", assignment.ID)
	return nil
}

func (a *VoteRepositoryAdapter) GetAssignmentByParticipant(eventID, participantID string) (*vote.Assignment, error) {
	if eventID == "" {
		a.log.Error("empty event ID provided for participant assignment")
		return nil, fmt.Errorf("event ID cannot be empty")
	}

	if participantID == "" {
		a.log.Error("empty participant ID provided for assignment")
		return nil, fmt.Errorf("participant ID cannot be empty")
	}

	// Validate UUID formats
	if _, err := uuid.Parse(eventID); err != nil {
		a.log.Error("invalid event ID format for participant assignment", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	if _, err := uuid.Parse(participantID); err != nil {
		a.log.Error("invalid participant ID format for assignment", "participant_id", participantID, "error", err)
		return nil, fmt.Errorf("invalid participant ID format: %w", err)
	}

	a.log.Debug("retrieving assignment by participant", "event_id", eventID, "participant_id", participantID)

	result, err := a.repo.GetAssignmentByParticipant(eventID, participantID)
	if err != nil {
		a.log.Error("failed to retrieve assignment by participant", 
			"event_id", eventID, 
			"participant_id", participantID, 
			"error", err)
		return nil, fmt.Errorf("failed to retrieve assignment for participant %s in event %s: %w", participantID, eventID, err)
	}

	a.log.Debug("assignment retrieved successfully", 
		"event_id", eventID, 
		"participant_id", participantID,
		"assignment_id", result.ID)
	return result, nil
}

func (a *VoteRepositoryAdapter) UpdateAssignment(assignment *vote.Assignment) error {
	if assignment == nil {
		a.log.Error("attempt to update nil assignment")
		return fmt.Errorf("assignment cannot be nil")
	}

	a.log.Debug("updating assignment", 
		"assignment_id", assignment.ID, 
		"event_id", assignment.EventID, 
		"participant_id", assignment.ParticipantID,
		"is_completed", assignment.IsCompleted)

	if err := a.repo.UpdateAssignment(assignment); err != nil {
		a.log.Error("failed to update assignment", 
			"assignment_id", assignment.ID, 
			"error", err)
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	a.log.Debug("assignment updated successfully", "assignment_id", assignment.ID)
	return nil
}

// AttachmentRepositoryAdapter adapts postgres.AttachmentRepository to vote.AttachmentRepository
// with enhanced error handling, logging, and validation
type AttachmentRepositoryAdapter struct {
	repo postgres.AttachmentRepository
	log  *log.Logger
}

func NewAttachmentRepositoryAdapter(repo postgres.AttachmentRepository) *AttachmentRepositoryAdapter {
	return &AttachmentRepositoryAdapter{
		repo: repo,
		log:  logger.Repository("attachment_adapter"),
	}
}

func (a *AttachmentRepositoryAdapter) GetByID(id string) (common.AttachmentInterface, error) {
	a.log.Debug("Retrieving attachment by ID", "id", id)
	
	// Validate attachment ID format
	if _, err := uuid.Parse(id); err != nil {
		a.log.Error("Invalid attachment ID format", "id", id, "error", err)
		return nil, fmt.Errorf("invalid attachment ID format: %w", err)
	}
	
	attachmentObj, err := a.repo.GetByID(id)
	if err != nil {
		a.log.Error("Failed to retrieve attachment", "id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve attachment %s: %w", id, err)
	}

	a.log.Debug("Attachment retrieved successfully", "id", id)
	// Return the actual attachment object which implements AttachmentInterface
	return attachmentObj, nil
}

func (a *AttachmentRepositoryAdapter) GetByEventID(eventID string) ([]common.AttachmentInterface, error) {
	a.log.Debug("Retrieving attachments by event ID", "event_id", eventID)
	
	// Validate event ID format
	if _, err := uuid.Parse(eventID); err != nil {
		a.log.Error("Invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}
	
	attachments, err := a.repo.GetByEventID(eventID)
	if err != nil {
		a.log.Error("Failed to retrieve attachments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve attachments for event %s: %w", eventID, err)
	}

	// Convert from []*attachment.Attachment to []common.AttachmentInterface
	var attachmentInterfaces []common.AttachmentInterface
	for _, att := range attachments {
		attachmentInterfaces = append(attachmentInterfaces, att)
	}

	a.log.Debug("Attachments retrieved successfully", "event_id", eventID, "count", len(attachmentInterfaces))
	return attachmentInterfaces, nil
}

// UserRepositoryAdapter adapts postgres.UserRepository to vote.UserRepository
// with enhanced error handling, logging, and validation
type UserRepositoryAdapter struct {
	repo postgres.UserRepository
	log  *log.Logger
}

func NewUserRepositoryAdapter(repo postgres.UserRepository) *UserRepositoryAdapter {
	return &UserRepositoryAdapter{
		repo: repo,
		log:  logger.Repository("user_adapter"),
	}
}

func (a *UserRepositoryAdapter) GetByID(id string) (common.UserInterface, error) {
	a.log.Debug("Retrieving user by ID", "id", id)
	
	// Validate user ID format
	if _, err := uuid.Parse(id); err != nil {
		a.log.Error("Invalid user ID format", "id", id, "error", err)
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}
	
	user, err := a.repo.GetByID(id)
	if err != nil {
		a.log.Error("Failed to retrieve user", "id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve user %s: %w", id, err)
	}

	a.log.Debug("User retrieved successfully", "id", id)
	// Return the actual user object which implements UserInterface
	return user, nil
}
