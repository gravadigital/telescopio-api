package attachment

import (
	"time"
)

type Attachment struct {
	ID            string    `json:"id" db:"id"`
	EventID       string    `json:"event_id" db:"event_id"`
	ParticipantID string    `json:"participant_id" db:"participant_id"`
	Filename      string    `json:"filename" db:"filename"`
	OriginalName  string    `json:"original_name" db:"original_name"`
	FilePath      string    `json:"file_path" db:"file_path"`
	FileSize      int64     `json:"file_size" db:"file_size"`
	MimeType      string    `json:"mime_type" db:"mime_type"`
	VoteCount     int       `json:"vote_count" db:"vote_count"`
	UploadedAt    time.Time `json:"uploaded_at" db:"uploaded_at"`
}

func NewAttachment(eventID, participantID, filename, originalName, filePath, mimeType string, fileSize int64) *Attachment {
	return &Attachment{
		ID:            generateAttachmentID(),
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

func generateAttachmentID() string {
	return "attachment_" + time.Now().Format("20060102150405")
}
