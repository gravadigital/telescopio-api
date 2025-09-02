package postgres

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
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

func (r *PostgresAttachmentRepository) Create(attachment *attachment.Attachment) error {
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

	if err := r.db.Create(attachment).Error; err != nil {
		r.log.Error("failed to create attachment", "error", err, "attachment_id", attachment.ID)
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	r.log.Info("attachment created successfully", "attachment_id", attachment.ID, "filename", attachment.Filename)
	return nil
}

func (r *PostgresAttachmentRepository) GetByID(id string) (*attachment.Attachment, error) {
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

	var att attachment.Attachment
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

func (r *PostgresAttachmentRepository) GetByEventID(eventID string) ([]*attachment.Attachment, error) {
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

	var attachments []*attachment.Attachment
	if err := r.db.Preload("Participant").Where("event_id = ?", eventUUID).Find(&attachments).Error; err != nil {
		r.log.Error("failed to retrieve attachments by event ID", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve attachments by event ID: %w", err)
	}

	r.log.Debug("attachments retrieved successfully", "event_id", eventID, "count", len(attachments))
	return attachments, nil
}

func (r *PostgresAttachmentRepository) GetByParticipantID(participantID string) ([]*attachment.Attachment, error) {
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

	var attachments []*attachment.Attachment
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

	// Check if attachment exists before deletion
	var attachment attachment.Attachment
	if err := r.db.First(&attachment, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent attachment", "attachment_id", id)
			return errors.New("attachment not found")
		}
		r.log.Error("failed to check attachment existence for deletion", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to check attachment existence: %w", err)
	}

	if err := r.db.Delete(&attachment).Error; err != nil {
		r.log.Error("failed to delete attachment", "attachment_id", id, "error", err)
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	r.log.Info("attachment deleted successfully", "attachment_id", id, "filename", attachment.Filename)
	return nil
}

func (r *PostgresAttachmentRepository) Update(attachment *attachment.Attachment) error {
	r.log.Debug("updating attachment", "attachment_id", attachment.ID, "filename", attachment.Filename)

	if attachment == nil {
		r.log.Error("attachment cannot be nil")
		return fmt.Errorf("attachment cannot be nil")
	}

	if attachment.Filename == "" {
		r.log.Error("attachment filename cannot be empty", "attachment_id", attachment.ID)
		return fmt.Errorf("attachment filename cannot be empty")
	}

	if err := r.db.Save(attachment).Error; err != nil {
		r.log.Error("failed to update attachment", "attachment_id", attachment.ID, "error", err)
		return fmt.Errorf("failed to update attachment: %w", err)
	}

	r.log.Info("attachment updated successfully", "attachment_id", attachment.ID, "filename", attachment.Filename)
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
	var attachment attachment.Attachment
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
