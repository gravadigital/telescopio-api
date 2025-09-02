package attachment

import (
	"time"

	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/domain/common"
	"gorm.io/gorm"
)

type Attachment struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EventID       uuid.UUID `json:"event_id" gorm:"type:uuid;not null"`
	ParticipantID uuid.UUID `json:"participant_id" gorm:"type:uuid;not null"`
	Filename      string    `json:"filename" gorm:"not null"`
	OriginalName  string    `json:"original_name" gorm:"not null"`
	FilePath      string    `json:"file_path" gorm:"not null"`
	FileSize      int64     `json:"file_size" gorm:"not null"`
	MimeType      string    `json:"mime_type" gorm:"not null"`
	VoteCount     int       `json:"vote_count" gorm:"default:0"`
	UploadedAt    time.Time `json:"uploaded_at" gorm:"autoCreateTime"`

	// Relations - using shared types to avoid circular imports
	Event       common.SharedEvent  `json:"event,omitempty" gorm:"foreignKey:EventID"`
	Participant common.SharedUser   `json:"participant,omitempty" gorm:"foreignKey:ParticipantID"`
	Votes       []common.SharedVote `json:"votes,omitempty" gorm:"foreignKey:AttachmentID"`
}

// TableName overrides the table name
func (Attachment) TableName() string {
	return "attachments"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func NewAttachment(eventID, participantID uuid.UUID, filename, originalName, filePath, mimeType string, fileSize int64) *Attachment {
	return &Attachment{
		ID:            uuid.New(),
		EventID:       eventID,
		ParticipantID: participantID,
		Filename:      filename,
		OriginalName:  originalName,
		FilePath:      filePath,
		FileSize:      fileSize,
		MimeType:      mimeType,
		VoteCount:     0,
		UploadedAt:    time.Now(),
	}
}

// Implement common.AttachmentInterface to avoid circular imports

func (a *Attachment) GetID() uuid.UUID {
	return a.ID
}

func (a *Attachment) GetOriginalName() string {
	return a.OriginalName
}

func (a *Attachment) GetParticipantID() uuid.UUID {
	return a.ParticipantID
}
