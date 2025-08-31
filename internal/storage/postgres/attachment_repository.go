package postgres

import (
	"errors"

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

	if err := r.db.Create(attachment).Error; err != nil {
		r.log.Error("failed to create attachment", "error", err, "attachment_id", attachment.ID)
		return err
	}

	r.log.Info("attachment created successfully", "attachment_id", attachment.ID, "filename", attachment.Filename)
	return nil
}

func (r *PostgresAttachmentRepository) GetByID(id string) (*attachment.Attachment, error) {
	r.log.Debug("retrieving attachment by ID", "attachment_id", id)

	attachmentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid attachment ID format", "attachment_id", id, "error", err)
		return nil, errors.New("invalid attachment ID format")
	}

	var att attachment.Attachment
	if err := r.db.Preload("Event").Preload("Participant").Preload("Votes").First(&att, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("attachment not found", "attachment_id", id)
			return nil, errors.New("attachment not found")
		}
		r.log.Error("failed to retrieve attachment", "attachment_id", id, "error", err)
		return nil, err
	}

	r.log.Debug("attachment retrieved successfully", "attachment_id", id, "filename", att.Filename)
	return &att, nil
}

func (r *PostgresAttachmentRepository) GetByEventID(eventID string) ([]*attachment.Attachment, error) {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, errors.New("invalid event ID format")
	}

	var attachments []*attachment.Attachment
	if err := r.db.Preload("Participant").Where("event_id = ?", eventUUID).Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *PostgresAttachmentRepository) GetByParticipantID(participantID string) ([]*attachment.Attachment, error) {
	participantUUID, err := uuid.Parse(participantID)
	if err != nil {
		return nil, errors.New("invalid participant ID format")
	}

	var attachments []*attachment.Attachment
	if err := r.db.Preload("Event").Where("participant_id = ?", participantUUID).Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *PostgresAttachmentRepository) Delete(id string) error {
	attachmentID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid attachment ID format")
	}

	if err := r.db.Delete(&attachment.Attachment{}, attachmentID).Error; err != nil {
		return err
	}
	return nil
}

func (r *PostgresAttachmentRepository) UpdateVoteCount(id string, count int) error {
	attachmentID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid attachment ID format")
	}

	if err := r.db.Model(&attachment.Attachment{}).Where("id = ?", attachmentID).Update("vote_count", count).Error; err != nil {
		return err
	}
	return nil
}
