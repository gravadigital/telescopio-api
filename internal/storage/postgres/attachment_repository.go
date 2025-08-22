package postgres

import (
	"errors"

	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
)

// InMemoryAttachmentRepository is a temporary in-memory implementation
// TODO: Replace with actual PostgreSQL implementation
type InMemoryAttachmentRepository struct {
	attachments map[string]*attachment.Attachment
}

func NewInMemoryAttachmentRepository() *InMemoryAttachmentRepository {
	return &InMemoryAttachmentRepository{
		attachments: make(map[string]*attachment.Attachment),
	}
}

func (r *InMemoryAttachmentRepository) Create(attachment *attachment.Attachment) error {
	r.attachments[attachment.ID] = attachment
	return nil
}

func (r *InMemoryAttachmentRepository) GetByID(id string) (*attachment.Attachment, error) {
	attachment, exists := r.attachments[id]
	if !exists {
		return nil, errors.New("attachment not found")
	}
	return attachment, nil
}

func (r *InMemoryAttachmentRepository) GetByEventID(eventID string) ([]*attachment.Attachment, error) {
	var attachments []*attachment.Attachment
	for _, attachment := range r.attachments {
		if attachment.EventID == eventID {
			attachments = append(attachments, attachment)
		}
	}
	return attachments, nil
}

func (r *InMemoryAttachmentRepository) GetByParticipantID(participantID string) ([]*attachment.Attachment, error) {
	var attachments []*attachment.Attachment
	for _, attachment := range r.attachments {
		if attachment.ParticipantID == participantID {
			attachments = append(attachments, attachment)
		}
	}
	return attachments, nil
}

func (r *InMemoryAttachmentRepository) Delete(id string) error {
	_, exists := r.attachments[id]
	if !exists {
		return errors.New("attachment not found")
	}
	delete(r.attachments, id)
	return nil
}

func (r *InMemoryAttachmentRepository) UpdateVoteCount(id string, count int) error {
	attachment, exists := r.attachments[id]
	if !exists {
		return errors.New("attachment not found")
	}
	attachment.VoteCount = count
	return nil
}
