package postgres

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	attachmentDomain "github.com/gravadigital/telescopio-api/internal/domain/attachment"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresAttachmentRepository implements AttachmentRepository using GORM
type PostgresAttachmentRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresAttachmentRepository creates a new PostgreSQL attachment repository
func NewPostgresAttachmentRepository(db *gorm.DB) *PostgresAttachmentRepository {
	return &PostgresAttachmentRepository{
		db:  db,
		log: logger.Repository("attachment"),
	}
}

func (r *PostgresAttachmentRepository) Create(attachment *attachmentDomain.Attachment) error {
	r.log.Debug("creating new attachment", "attachment_id", attachment.ID, "filename", attachment.Filename, "event_id", attachment.EventID)

	// Basic validation
	if attachment == nil {
		r.log.Error("attachment cannot be nil")
		return fmt.Errorf("attachment cannot be nil")
	}

	if attachment.Filename == "" {
		r.log.Error("attachment filename cannot be empty", "attachment_id", attachment.ID)
		return fmt.Errorf("attachment filename cannot be empty")
	}

	if attachment.OriginalName == "" {
		r.log.Error("attachment original name cannot be empty", "attachment_id", attachment.ID)
		return fmt.Errorf("attachment original name cannot be empty")
	}

	if attachment.FilePath == "" {
		r.log.Error("attachment file path cannot be empty", "attachment_id", attachment.ID)
		return fmt.Errorf("attachment file path cannot be empty")
	}

	// Validate file size (if specified)
	if attachment.FileSize < 0 {
		r.log.Error("attachment file size cannot be negative", "attachment_id", attachment.ID, "file_size", attachment.FileSize)
		return fmt.Errorf("attachment file size cannot be negative")
	}

	// Validate that event exists
	if attachment.EventID != uuid.Nil {
		var eventExists bool
		if err := r.db.Model(&struct{ ID uuid.UUID }{}).
			Select("COUNT(*) > 0").
			Where("id = ?", attachment.EventID).
			Scan(&eventExists).Error; err != nil {
			r.log.Error("failed to check event existence", "event_id", attachment.EventID, "error", err)
			return fmt.Errorf("failed to validate event existence: %w", err)
		}

		if !eventExists {
			r.log.Error("event does not exist", "event_id", attachment.EventID, "attachment_id", attachment.ID)
			return fmt.Errorf("event does not exist")
		}
	}

	// Validate that participant exists
	if attachment.ParticipantID != uuid.Nil {
		var participantExists bool
		if err := r.db.Table("users").
			Select("COUNT(*) > 0").
			Where("id = ?", attachment.ParticipantID).
			Scan(&participantExists).Error; err != nil {
			r.log.Error("failed to check participant existence", "participant_id", attachment.ParticipantID, "error", err)
			return fmt.Errorf("failed to validate participant existence: %w", err)
		}

		if !participantExists {
			r.log.Error("participant does not exist", "participant_id", attachment.ParticipantID, "attachment_id", attachment.ID)
			return fmt.Errorf("participant does not exist")
		}
	}

	if err := r.db.Create(attachment).Error; err != nil {
		r.log.Error("failed to create attachment", "error", err, "attachment_id", attachment.ID)
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	r.log.Info("attachment created successfully", "attachment_id", attachment.ID, "filename", attachment.Filename)
	return nil
}

func (r *PostgresAttachmentRepository) GetByID(id string) (*attachmentDomain.Attachment, error) {
	r.log.Debug("retrieving attachment by ID", "attachment_id", id)

	if id == "" {
		r.log.Error("attachment ID cannot be empty")
		return nil, errors.New("attachment ID cannot be empty")
	}

	attachmentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid attachment ID format", "attachment_id", id, "error", err)
		return nil, fmt.Errorf("invalid attachment ID format: %w", err)
	}

	var att attachmentDomain.Attachment
	if err := r.db.Preload("Event").Preload("Participant").Preload("Votes").First(&att, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("attachment not found", "attachment_id", id)
			return nil, errors.New("attachment not found")
		}
		r.log.Error("failed to retrieve attachment", "attachment_id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve attachment: %w", err)
	}

	r.log.Debug("attachment retrieved successfully", "attachment_id", id, "filename", att.Filename)
	return &att, nil
}

func (r *PostgresAttachmentRepository) GetByEventID(eventID string) ([]*attachmentDomain.Attachment, error) {
	r.log.Debug("retrieving attachments by event ID", "event_id", eventID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	var attachments []*attachmentDomain.Attachment
	if err := r.db.Preload("Participant").Where("event_id = ?", eventUUID).Find(&attachments).Error; err != nil {
		r.log.Error("failed to retrieve attachments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve attachments by event ID: %w", err)
	}

	r.log.Debug("attachments retrieved successfully", "event_id", eventID, "count", len(attachments))
	return attachments, nil
}

func (r *PostgresAttachmentRepository) GetByParticipantID(participantID string) ([]*attachmentDomain.Attachment, error) {
	r.log.Debug("retrieving attachments by participant ID", "participant_id", participantID)

	if participantID == "" {
		r.log.Error("participant ID cannot be empty")
		return nil, errors.New("participant ID cannot be empty")
	}

	participantUUID, err := uuid.Parse(participantID)
	if err != nil {
		r.log.Error("invalid participant ID format", "participant_id", participantID, "error", err)
		return nil, fmt.Errorf("invalid participant ID format: %w", err)
	}

	var attachments []*attachmentDomain.Attachment
	if err := r.db.Preload("Event").Where("participant_id = ?", participantUUID).Find(&attachments).Error; err != nil {
		r.log.Error("failed to retrieve attachments by participant ID", "participant_id", participantID, "error", err)
		return nil, fmt.Errorf("failed to retrieve attachments by participant ID: %w", err)
	}

	r.log.Debug("attachments retrieved successfully", "participant_id", participantID, "count", len(attachments))
	return attachments, nil
}

func (r *PostgresAttachmentRepository) Delete(id string) error {
	r.log.Debug("deleting attachment", "attachment_id", id)

	if id == "" {
		r.log.Error("attachment ID cannot be empty")
		return errors.New("attachment ID cannot be empty")
	}

	attachmentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid attachment ID format", "attachment_id", id, "error", err)
		return fmt.Errorf("invalid attachment ID format: %w", err)
	}

	// Start transaction for safe deletion
	tx := r.db.Begin()
	if tx.Error != nil {
		r.log.Error("failed to start transaction for attachment deletion", "attachment_id", id, "error", tx.Error)
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Check if attachment exists before deletion
	var attachment attachmentDomain.Attachment
	if err := tx.First(&attachment, attachmentID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent attachment", "attachment_id", id)
			return errors.New("attachment not found")
		}
		r.log.Error("failed to check attachment existence for deletion", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to check attachment existence: %w", err)
	}

	// Check for related votes
	var voteCount int64
	if err := tx.Model(&struct{ ID uuid.UUID }{}).
		Table("votes").
		Where("attachment_id = ?", attachmentID).
		Count(&voteCount).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to check vote dependencies", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to check vote dependencies: %w", err)
	}

	if voteCount > 0 {
		// Delete related votes first
		if err := tx.Where("attachment_id = ?", attachmentID).Delete(&struct{ ID uuid.UUID }{}).Error; err != nil {
			tx.Rollback()
			r.log.Error("failed to delete related votes", "attachment_id", id, "error", err)
			return fmt.Errorf("failed to delete related votes: %w", err)
		}
		r.log.Debug("deleted related votes", "attachment_id", id, "vote_count", voteCount)
	}

	// Delete the attachment
	if err := tx.Delete(&attachment).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete attachment", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		r.log.Error("failed to commit attachment deletion transaction", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("attachment deleted successfully", "attachment_id", id, "filename", attachment.Filename, "deleted_votes", voteCount)
	return nil
}

func (r *PostgresAttachmentRepository) Update(attachment *attachmentDomain.Attachment) error {
	r.log.Debug("updating attachment", "attachment_id", attachment.ID, "filename", attachment.Filename)

	if attachment == nil {
		r.log.Error("attachment cannot be nil")
		return fmt.Errorf("attachment cannot be nil")
	}

	if attachment.Filename == "" {
		r.log.Error("attachment filename cannot be empty", "attachment_id", attachment.ID)
		return fmt.Errorf("attachment filename cannot be empty")
	}

	if attachment.OriginalName == "" {
		r.log.Error("attachment original name cannot be empty", "attachment_id", attachment.ID)
		return fmt.Errorf("attachment original name cannot be empty")
	}

	// Validate file size (if specified)
	if attachment.FileSize < 0 {
		r.log.Error("attachment file size cannot be negative", "attachment_id", attachment.ID, "file_size", attachment.FileSize)
		return fmt.Errorf("attachment file size cannot be negative")
	}

	// Check if attachment exists before updating
	var existingAttachment attachmentDomain.Attachment
	if err := r.db.First(&existingAttachment, attachment.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("attachment not found for update", "attachment_id", attachment.ID)
			return errors.New("attachment not found")
		}
		r.log.Error("failed to check attachment existence for update", "attachment_id", attachment.ID, "error", err)
		return fmt.Errorf("failed to check attachment existence: %w", err)
	}

	if err := r.db.Save(attachment).Error; err != nil {
		r.log.Error("failed to update attachment", "attachment_id", attachment.ID, "error", err)
		return fmt.Errorf("failed to update attachment: %w", err)
	}

	r.log.Info("attachment updated successfully", "attachment_id", attachment.ID, "filename", attachment.Filename)
	return nil
}

// GetByEventIDPaginated retrieves attachments by event ID with pagination
func (r *PostgresAttachmentRepository) GetByEventIDPaginated(eventID string, params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving attachments by event ID with pagination", "event_id", eventID, "page", params.Page, "page_size", params.PageSize)

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
		params.PageSize = 100
	}

	offset := (params.Page - 1) * params.PageSize

	// Get total count
	var total int64
	if err := r.db.Model(&attachmentDomain.Attachment{}).Where("event_id = ?", eventUUID).Count(&total).Error; err != nil {
		r.log.Error("failed to count attachments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count attachments: %w", err)
	}

	// Get paginated attachments
	var attachments []*attachmentDomain.Attachment
	if err := r.db.Preload("Participant").
		Where("event_id = ?", eventUUID).
		Offset(offset).Limit(params.PageSize).
		Order("created_at DESC").
		Find(&attachments).Error; err != nil {
		r.log.Error("failed to retrieve paginated attachments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated attachments: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       attachments,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated attachments retrieved successfully",
		"event_id", eventID,
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"returned_count", len(attachments))

	return result, nil
}

// UpdatePartial updates only specified fields of an attachment
func (r *PostgresAttachmentRepository) UpdatePartial(id string, updates map[string]interface{}) error {
	r.log.Debug("updating attachment partially", "attachment_id", id, "fields", len(updates))

	if id == "" {
		r.log.Error("attachment ID cannot be empty")
		return errors.New("attachment ID cannot be empty")
	}

	if len(updates) == 0 {
		r.log.Error("no updates provided")
		return errors.New("no updates provided")
	}

	attachmentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid attachment ID format", "attachment_id", id, "error", err)
		return fmt.Errorf("invalid attachment ID format: %w", err)
	}

	// Check if attachment exists
	var existing attachmentDomain.Attachment
	if err := r.db.First(&existing, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("attachment not found for partial update", "attachment_id", id)
			return errors.New("attachment not found")
		}
		r.log.Error("failed to check attachment existence for partial update", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to check attachment existence: %w", err)
	}

	// Validate critical fields if they're being updated
	if filename, ok := updates["filename"]; ok {
		if filename == "" {
			r.log.Error("filename cannot be empty in partial update", "attachment_id", id)
			return fmt.Errorf("filename cannot be empty")
		}
	}

	if originalName, ok := updates["original_name"]; ok {
		if originalName == "" {
			r.log.Error("original_name cannot be empty in partial update", "attachment_id", id)
			return fmt.Errorf("original_name cannot be empty")
		}
	}

	if fileSize, ok := updates["file_size"]; ok {
		if size, isInt := fileSize.(int64); isInt && size < 0 {
			r.log.Error("file_size cannot be negative in partial update", "attachment_id", id, "file_size", size)
			return fmt.Errorf("file_size cannot be negative")
		}
	}

	if err := r.db.Model(&existing).Updates(updates).Error; err != nil {
		r.log.Error("failed to partially update attachment", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to partially update attachment: %w", err)
	}

	r.log.Info("attachment partially updated successfully", "attachment_id", id, "updated_fields", len(updates))
	return nil
}

func (r *PostgresAttachmentRepository) UpdateVoteCount(id string, count int) error {
	r.log.Debug("updating attachment vote count", "attachment_id", id, "new_count", count)

	if id == "" {
		r.log.Error("attachment ID cannot be empty")
		return errors.New("attachment ID cannot be empty")
	}

	if count < 0 {
		r.log.Error("vote count cannot be negative", "attachment_id", id, "count", count)
		return errors.New("vote count cannot be negative")
	}

	attachmentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid attachment ID format", "attachment_id", id, "error", err)
		return fmt.Errorf("invalid attachment ID format: %w", err)
	}

	// Check if attachment exists before updating
	var attachment attachmentDomain.Attachment
	if err := r.db.First(&attachment, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("attachment not found for vote count update", "attachment_id", id)
			return errors.New("attachment not found")
		}
		r.log.Error("failed to check attachment existence for vote count update", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to check attachment existence: %w", err)
	}

	if err := r.db.Model(&attachment).Update("vote_count", count).Error; err != nil {
		r.log.Error("failed to update attachment vote count", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to update attachment vote count: %w", err)
	}

	r.log.Info("attachment vote count updated successfully", "attachment_id", id, "old_count", attachment.VoteCount, "new_count", count)
	return nil
}
