package postgres

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresVoteRepository implements VoteRepository using GORM
type PostgresVoteRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresVoteRepository creates a new PostgreSQL vote repository
func NewPostgresVoteRepository(db *gorm.DB) *PostgresVoteRepository {
	return &PostgresVoteRepository{
		db:  db,
		log: logger.Repository("vote"),
	}
}

func (r *PostgresVoteRepository) Create(vote *vote.Vote) error {
	r.log.Debug("creating new vote", "vote_id", vote.ID, "event_id", vote.EventID, "voter_id", vote.VoterID)

	// Validate vote before creating
	if err := vote.Validate(); err != nil {
		r.log.Error("vote validation failed", "error", err, "vote_id", vote.ID)
		return fmt.Errorf("vote validation failed: %w", err)
	}

	if err := r.db.Create(vote).Error; err != nil {
		r.log.Error("failed to create vote", "error", err, "vote_id", vote.ID)
		return fmt.Errorf("failed to create vote: %w", err)
	}

	r.log.Info("vote created successfully", "vote_id", vote.ID, "event_id", vote.EventID, "voter_id", vote.VoterID)
	return nil
}

func (r *PostgresVoteRepository) GetByID(id string) (*vote.Vote, error) {
	r.log.Debug("retrieving vote by ID", "vote_id", id)

	voteID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid vote ID format", "vote_id", id, "error", err)
		return nil, errors.New("invalid vote ID format")
	}

	var v vote.Vote
	if err := r.db.Preload("Event").Preload("Voter").Preload("Attachment").First(&v, voteID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("vote not found", "vote_id", id)
			return nil, errors.New("vote not found")
		}
		r.log.Error("failed to retrieve vote", "vote_id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve vote: %w", err)
	}

	r.log.Debug("vote retrieved successfully", "vote_id", id)
	return &v, nil
}

func (r *PostgresVoteRepository) GetByEventID(eventID string) ([]*vote.Vote, error) {
	r.log.Debug("retrieving votes by event ID", "event_id", eventID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	var votes []*vote.Vote
	if err := r.db.Preload("Voter").Preload("Attachment").Where("event_id = ?", eventUUID).Find(&votes).Error; err != nil {
		r.log.Error("failed to retrieve votes by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve votes by event ID: %w", err)
	}

	r.log.Debug("votes retrieved successfully", "event_id", eventID, "count", len(votes))
	return votes, nil
}

func (r *PostgresVoteRepository) GetByEventIDPaginated(eventID string, params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving votes by event ID with pagination", "event_id", eventID, "page", params.Page, "page_size", params.PageSize)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	// Set default values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100 // Maximum page size limit
	}

	offset := (params.Page - 1) * params.PageSize

	// Get total count
	var total int64
	if err := r.db.Model(&vote.Vote{}).Where("event_id = ?", eventUUID).Count(&total).Error; err != nil {
		r.log.Error("failed to count votes by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count votes by event ID: %w", err)
	}

	// Get paginated votes
	var votes []*vote.Vote
	if err := r.db.Preload("Voter").Preload("Attachment").
		Where("event_id = ?", eventUUID).
		Offset(offset).Limit(params.PageSize).
		Order("voted_at DESC").
		Find(&votes).Error; err != nil {
		r.log.Error("failed to retrieve paginated votes by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated votes by event ID: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       votes,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated votes by event ID retrieved successfully",
		"event_id", eventID,
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"total_pages", totalPages,
		"returned_count", len(votes))

	return result, nil
}

func (r *PostgresVoteRepository) GetByVoterID(voterID string) ([]*vote.Vote, error) {
	r.log.Debug("retrieving votes by voter ID", "voter_id", voterID)

	if voterID == "" {
		r.log.Error("voter ID cannot be empty")
		return nil, errors.New("voter ID cannot be empty")
	}

	voterUUID, err := uuid.Parse(voterID)
	if err != nil {
		r.log.Error("invalid voter ID format", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("invalid voter ID format: %w", err)
	}

	var votes []*vote.Vote
	if err := r.db.Preload("Event").Preload("Attachment").Where("voter_id = ?", voterUUID).Find(&votes).Error; err != nil {
		r.log.Error("failed to retrieve votes by voter ID", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("failed to retrieve votes by voter ID: %w", err)
	}

	r.log.Debug("votes retrieved successfully", "voter_id", voterID, "count", len(votes))
	return votes, nil
}

func (r *PostgresVoteRepository) GetByVoterIDPaginated(voterID string, params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving votes by voter ID with pagination", "voter_id", voterID, "page", params.Page, "page_size", params.PageSize)

	if voterID == "" {
		r.log.Error("voter ID cannot be empty")
		return nil, errors.New("voter ID cannot be empty")
	}

	voterUUID, err := uuid.Parse(voterID)
	if err != nil {
		r.log.Error("invalid voter ID format", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("invalid voter ID format: %w", err)
	}

	// Set default values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100 // Maximum page size limit
	}

	offset := (params.Page - 1) * params.PageSize

	// Get total count
	var total int64
	if err := r.db.Model(&vote.Vote{}).Where("voter_id = ?", voterUUID).Count(&total).Error; err != nil {
		r.log.Error("failed to count votes by voter ID", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("failed to count votes by voter ID: %w", err)
	}

	// Get paginated votes
	var votes []*vote.Vote
	if err := r.db.Preload("Event").Preload("Attachment").
		Where("voter_id = ?", voterUUID).
		Offset(offset).Limit(params.PageSize).
		Order("voted_at DESC").
		Find(&votes).Error; err != nil {
		r.log.Error("failed to retrieve paginated votes by voter ID", "voter_id", voterID, "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated votes by voter ID: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       votes,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated votes by voter ID retrieved successfully",
		"voter_id", voterID,
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"total_pages", totalPages,
		"returned_count", len(votes))

	return result, nil
}

func (r *PostgresVoteRepository) GetByAttachmentID(attachmentID string) ([]*vote.Vote, error) {
	r.log.Debug("retrieving votes by attachment ID", "attachment_id", attachmentID)

	if attachmentID == "" {
		r.log.Error("attachment ID cannot be empty")
		return nil, errors.New("attachment ID cannot be empty")
	}

	attachmentUUID, err := uuid.Parse(attachmentID)
	if err != nil {
		r.log.Error("invalid attachment ID format", "attachment_id", attachmentID, "error", err)
		return nil, fmt.Errorf("invalid attachment ID format: %w", err)
	}

	var votes []*vote.Vote
	if err := r.db.Preload("Event").Preload("Voter").Where("attachment_id = ?", attachmentUUID).Find(&votes).Error; err != nil {
		r.log.Error("failed to retrieve votes by attachment ID", "attachment_id", attachmentID, "error", err)
		return nil, fmt.Errorf("failed to retrieve votes by attachment ID: %w", err)
	}

	r.log.Debug("votes retrieved successfully", "attachment_id", attachmentID, "count", len(votes))
	return votes, nil
}

func (r *PostgresVoteRepository) Update(voteEntity *vote.Vote) error {
	r.log.Debug("updating vote", "vote_id", voteEntity.ID, "event_id", voteEntity.EventID, "voter_id", voteEntity.VoterID)

	// Validate vote before updating
	if err := voteEntity.Validate(); err != nil {
		r.log.Error("vote validation failed", "error", err, "vote_id", voteEntity.ID)
		return fmt.Errorf("vote validation failed: %w", err)
	}

	// Check if vote exists
	var existingVote vote.Vote
	if err := r.db.First(&existingVote, voteEntity.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("vote not found for update", "vote_id", voteEntity.ID)
			return errors.New("vote not found")
		}
		r.log.Error("failed to check vote existence for update", "vote_id", voteEntity.ID, "error", err)
		return fmt.Errorf("failed to check vote existence: %w", err)
	}

	if err := r.db.Save(voteEntity).Error; err != nil {
		r.log.Error("failed to update vote", "error", err, "vote_id", voteEntity.ID)
		return fmt.Errorf("failed to update vote: %w", err)
	}

	r.log.Info("vote updated successfully", "vote_id", voteEntity.ID, "event_id", voteEntity.EventID, "voter_id", voteEntity.VoterID)
	return nil
}

func (r *PostgresVoteRepository) Delete(id string) error {
	r.log.Debug("deleting vote", "vote_id", id)

	voteID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid vote ID format", "vote_id", id, "error", err)
		return errors.New("invalid vote ID format")
	}

	// Check if vote exists
	var voteEntity vote.Vote
	if err := r.db.First(&voteEntity, voteID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent vote", "vote_id", id)
			return errors.New("vote not found")
		}
		r.log.Error("failed to check vote existence for deletion", "vote_id", id, "error", err)
		return fmt.Errorf("failed to check vote existence: %w", err)
	}

	if err := r.db.Delete(&voteEntity).Error; err != nil {
		r.log.Error("failed to delete vote", "vote_id", id, "error", err)
		return fmt.Errorf("failed to delete vote: %w", err)
	}

	r.log.Info("vote deleted successfully", "vote_id", id, "event_id", voteEntity.EventID, "voter_id", voteEntity.VoterID)
	return nil
}

func (r *PostgresVoteRepository) HasVoted(eventID, voterID string) (bool, error) {
	r.log.Debug("checking if voter has voted", "event_id", eventID, "voter_id", voterID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return false, errors.New("event ID cannot be empty")
	}

	if voterID == "" {
		r.log.Error("voter ID cannot be empty")
		return false, errors.New("voter ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return false, fmt.Errorf("invalid event ID format: %w", err)
	}

	voterUUID, err := uuid.Parse(voterID)
	if err != nil {
		r.log.Error("invalid voter ID format", "voter_id", voterID, "error", err)
		return false, fmt.Errorf("invalid voter ID format: %w", err)
	}

	var count int64
	if err := r.db.Model(&vote.Vote{}).Where("event_id = ? AND voter_id = ?", eventUUID, voterUUID).Count(&count).Error; err != nil {
		r.log.Error("failed to check voting status", "event_id", eventID, "voter_id", voterID, "error", err)
		return false, fmt.Errorf("failed to check voting status: %w", err)
	}

	hasVoted := count > 0
	r.log.Debug("voting status checked", "event_id", eventID, "voter_id", voterID, "has_voted", hasVoted, "vote_count", count)
	return hasVoted, nil
}

// Assignment methods

func (r *PostgresVoteRepository) CreateAssignment(assignment *vote.Assignment) error {
	r.log.Debug("creating new assignment", "assignment_id", assignment.ID, "event_id", assignment.EventID, "participant_id", assignment.ParticipantID)

	// Validate assignment before creating
	if assignment.EventID == uuid.Nil {
		r.log.Error("event ID cannot be nil", "assignment_id", assignment.ID)
		return errors.New("event ID is required")
	}

	if assignment.ParticipantID == uuid.Nil {
		r.log.Error("participant ID cannot be nil", "assignment_id", assignment.ID)
		return errors.New("participant ID is required")
	}

	if err := r.db.Create(assignment).Error; err != nil {
		r.log.Error("failed to create assignment", "error", err, "assignment_id", assignment.ID)
		return fmt.Errorf("failed to create assignment: %w", err)
	}

	r.log.Info("assignment created successfully", "assignment_id", assignment.ID, "event_id", assignment.EventID, "participant_id", assignment.ParticipantID)
	return nil
}

func (r *PostgresVoteRepository) GetAssignmentsByEventID(eventID string) ([]*vote.Assignment, error) {
	r.log.Debug("retrieving assignments by event ID", "event_id", eventID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	var assignments []*vote.Assignment
	if err := r.db.Preload("Participant").Preload("Event").Where("event_id = ?", eventUUID).Find(&assignments).Error; err != nil {
		r.log.Error("failed to retrieve assignments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve assignments by event ID: %w", err)
	}

	r.log.Debug("assignments retrieved successfully", "event_id", eventID, "count", len(assignments))
	return assignments, nil
}

func (r *PostgresVoteRepository) GetAssignmentsByEventIDPaginated(eventID string, params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving assignments by event ID with pagination", "event_id", eventID, "page", params.Page, "page_size", params.PageSize)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	// Set default values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100 // Maximum page size limit
	}

	offset := (params.Page - 1) * params.PageSize

	// Get total count
	var total int64
	if err := r.db.Model(&vote.Assignment{}).Where("event_id = ?", eventUUID).Count(&total).Error; err != nil {
		r.log.Error("failed to count assignments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count assignments by event ID: %w", err)
	}

	// Get paginated assignments
	var assignments []*vote.Assignment
	if err := r.db.Preload("Participant").Preload("Event").
		Where("event_id = ?", eventUUID).
		Offset(offset).Limit(params.PageSize).
		Order("created_at DESC").
		Find(&assignments).Error; err != nil {
		r.log.Error("failed to retrieve paginated assignments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated assignments by event ID: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       assignments,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated assignments by event ID retrieved successfully",
		"event_id", eventID,
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"total_pages", totalPages,
		"returned_count", len(assignments))

	return result, nil
}

func (r *PostgresVoteRepository) GetAssignmentByParticipant(eventID, participantID string) (*vote.Assignment, error) {
	r.log.Debug("retrieving assignment by participant", "event_id", eventID, "participant_id", participantID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	if participantID == "" {
		r.log.Error("participant ID cannot be empty")
		return nil, errors.New("participant ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	participantUUID, err := uuid.Parse(participantID)
	if err != nil {
		r.log.Error("invalid participant ID format", "participant_id", participantID, "error", err)
		return nil, fmt.Errorf("invalid participant ID format: %w", err)
	}

	var assignment vote.Assignment
	if err := r.db.Preload("Participant").Preload("Event").Where("event_id = ? AND participant_id = ?", eventUUID, participantUUID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("assignment not found", "event_id", eventID, "participant_id", participantID)
			return nil, errors.New("assignment not found")
		}
		r.log.Error("failed to retrieve assignment by participant", "event_id", eventID, "participant_id", participantID, "error", err)
		return nil, fmt.Errorf("failed to retrieve assignment by participant: %w", err)
	}

	r.log.Debug("assignment retrieved successfully", "event_id", eventID, "participant_id", participantID, "assignment_id", assignment.ID)
	return &assignment, nil
}

func (r *PostgresVoteRepository) UpdateAssignment(assignment *vote.Assignment) error {
	r.log.Debug("updating assignment", "assignment_id", assignment.ID)

	// Validate assignment before updating
	if assignment.EventID == uuid.Nil {
		r.log.Error("event ID cannot be nil", "assignment_id", assignment.ID)
		return errors.New("event ID is required")
	}

	if assignment.ParticipantID == uuid.Nil {
		r.log.Error("participant ID cannot be nil", "assignment_id", assignment.ID)
		return errors.New("participant ID is required")
	}

	// Check if assignment exists
	var existingAssignment vote.Assignment
	if err := r.db.First(&existingAssignment, assignment.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("assignment not found for update", "assignment_id", assignment.ID)
			return errors.New("assignment not found")
		}
		r.log.Error("failed to check assignment existence for update", "assignment_id", assignment.ID, "error", err)
		return fmt.Errorf("failed to check assignment existence: %w", err)
	}

	if err := r.db.Save(assignment).Error; err != nil {
		r.log.Error("failed to update assignment", "error", err, "assignment_id", assignment.ID)
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	r.log.Info("assignment updated successfully", "assignment_id", assignment.ID)
	return nil
}

func (r *PostgresVoteRepository) DeleteAssignment(id string) error {
	r.log.Debug("deleting assignment", "assignment_id", id)

	assignmentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid assignment ID format", "assignment_id", id, "error", err)
		return errors.New("invalid assignment ID format")
	}

	// Start a transaction for safe deletion
	tx := r.db.Begin()
	if tx.Error != nil {
		r.log.Error("failed to start transaction for assignment deletion", "assignment_id", id, "error", tx.Error)
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Check if assignment exists
	var assignment vote.Assignment
	if err := tx.First(&assignment, assignmentID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent assignment", "assignment_id", id)
			return errors.New("assignment not found")
		}
		r.log.Error("failed to check assignment existence for deletion", "assignment_id", id, "error", err)
		return fmt.Errorf("failed to check assignment existence: %w", err)
	}

	// Delete related votes first
	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&vote.Vote{}).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete related votes", "assignment_id", id, "error", err)
		return fmt.Errorf("failed to delete related votes: %w", err)
	}

	// Delete the assignment
	if err := tx.Delete(&assignment).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete assignment", "assignment_id", id, "error", err)
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		r.log.Error("failed to commit assignment deletion transaction", "assignment_id", id, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("assignment deleted successfully", "assignment_id", id, "event_id", assignment.EventID, "participant_id", assignment.ParticipantID)
	return nil
}

// GetVotesByAssignmentID retrieves all votes for a specific assignment
func (r *PostgresVoteRepository) GetVotesByAssignmentID(assignmentID string) ([]*vote.Vote, error) {
	r.log.Debug("retrieving votes by assignment ID", "assignment_id", assignmentID)

	if assignmentID == "" {
		r.log.Error("assignment ID cannot be empty")
		return nil, errors.New("assignment ID cannot be empty")
	}

	assignmentUUID, err := uuid.Parse(assignmentID)
	if err != nil {
		r.log.Error("invalid assignment ID format", "assignment_id", assignmentID, "error", err)
		return nil, fmt.Errorf("invalid assignment ID format: %w", err)
	}

	var votes []*vote.Vote
	if err := r.db.Preload("Event").Preload("Voter").Preload("Attachment").
		Where("assignment_id = ?", assignmentUUID).Find(&votes).Error; err != nil {
		r.log.Error("failed to retrieve votes by assignment ID", "assignment_id", assignmentID, "error", err)
		return nil, fmt.Errorf("failed to retrieve votes by assignment ID: %w", err)
	}

	r.log.Debug("votes by assignment retrieved successfully", "assignment_id", assignmentID, "count", len(votes))
	return votes, nil
}

// GetAssignmentProgress returns the completion status of an assignment
func (r *PostgresVoteRepository) GetAssignmentProgress(assignmentID string) (map[string]interface{}, error) {
	r.log.Debug("getting assignment progress", "assignment_id", assignmentID)

	assignmentUUID, err := uuid.Parse(assignmentID)
	if err != nil {
		r.log.Error("invalid assignment ID format", "assignment_id", assignmentID, "error", err)
		return nil, fmt.Errorf("invalid assignment ID format: %w", err)
	}

	// Get assignment details
	var assignment vote.Assignment
	if err := r.db.First(&assignment, assignmentUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("assignment not found", "assignment_id", assignmentID)
			return nil, errors.New("assignment not found")
		}
		r.log.Error("failed to retrieve assignment", "assignment_id", assignmentID, "error", err)
		return nil, fmt.Errorf("failed to retrieve assignment: %w", err)
	}

	// Count total attachments assigned
	totalAttachments := len(assignment.AttachmentIDs)

	// Count votes submitted for this assignment
	var votesSubmitted int64
	if err := r.db.Model(&vote.Vote{}).Where("assignment_id = ?", assignmentUUID).Count(&votesSubmitted).Error; err != nil {
		r.log.Error("failed to count votes for assignment", "assignment_id", assignmentID, "error", err)
		return nil, fmt.Errorf("failed to count votes: %w", err)
	}

	progress := map[string]interface{}{
		"assignment_id":     assignmentID,
		"participant_id":    assignment.ParticipantID.String(),
		"event_id":          assignment.EventID.String(),
		"total_attachments": totalAttachments,
		"votes_submitted":   votesSubmitted,
		"is_completed":      assignment.IsCompleted,
		"completion_rate":   float64(votesSubmitted) / float64(totalAttachments),
		"created_at":        assignment.CreatedAt,
		"completed_at":      assignment.CompletedAt,
	}

	r.log.Debug("assignment progress retrieved", "assignment_id", assignmentID, "progress", progress)
	return progress, nil
}

// GetVotingStatistics returns comprehensive voting statistics for an event
func (r *PostgresVoteRepository) GetVotingStatistics(eventID string) (map[string]interface{}, error) {
	r.log.Debug("getting voting statistics", "event_id", eventID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	stats := make(map[string]interface{})

	// Total votes count
	var totalVotes int64
	if err := r.db.Model(&vote.Vote{}).Where("event_id = ?", eventUUID).Count(&totalVotes).Error; err != nil {
		r.log.Error("failed to count total votes", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count total votes: %w", err)
	}
	stats["total_votes"] = totalVotes

	// Total assignments count
	var totalAssignments int64
	if err := r.db.Model(&vote.Assignment{}).Where("event_id = ?", eventUUID).Count(&totalAssignments).Error; err != nil {
		r.log.Error("failed to count total assignments", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count total assignments: %w", err)
	}
	stats["total_assignments"] = totalAssignments

	// Completed assignments count
	var completedAssignments int64
	if err := r.db.Model(&vote.Assignment{}).Where("event_id = ? AND is_completed = ?", eventUUID, true).Count(&completedAssignments).Error; err != nil {
		r.log.Error("failed to count completed assignments", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count completed assignments: %w", err)
	}
	stats["completed_assignments"] = completedAssignments

	// Unique voters count
	var uniqueVoters int64
	if err := r.db.Model(&vote.Vote{}).Where("event_id = ?", eventUUID).Distinct("voter_id").Count(&uniqueVoters).Error; err != nil {
		r.log.Error("failed to count unique voters", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count unique voters: %w", err)
	}
	stats["unique_voters"] = uniqueVoters

	// Unique attachments voted on
	var uniqueAttachments int64
	if err := r.db.Model(&vote.Vote{}).Where("event_id = ?", eventUUID).Distinct("attachment_id").Count(&uniqueAttachments).Error; err != nil {
		r.log.Error("failed to count unique attachments", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count unique attachments: %w", err)
	}
	stats["unique_attachments"] = uniqueAttachments

	// Calculate completion rate
	var completionRate float64
	if totalAssignments > 0 {
		completionRate = float64(completedAssignments) / float64(totalAssignments)
	}
	stats["completion_rate"] = completionRate

	// Average votes per attachment
	var avgVotesPerAttachment float64
	if uniqueAttachments > 0 {
		avgVotesPerAttachment = float64(totalVotes) / float64(uniqueAttachments)
	}
	stats["avg_votes_per_attachment"] = avgVotesPerAttachment

	stats["event_id"] = eventID

	r.log.Debug("voting statistics retrieved successfully", "event_id", eventID, "stats", stats)
	return stats, nil
}
