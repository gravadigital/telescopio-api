package vote

import (
	"time"
)

type Vote struct {
	ID           string    `json:"id" db:"id"`
	EventID      string    `json:"event_id" db:"event_id"`
	VoterID      string    `json:"voter_id" db:"voter_id"`
	AttachmentID string    `json:"attachment_id" db:"attachment_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

func NewVote(eventID, voterID, attachmentID string) *Vote {
	return &Vote{
		ID:           generateVoteID(),
		EventID:      eventID,
		VoterID:      voterID,
		AttachmentID: attachmentID,
		CreatedAt:    time.Now(),
	}
}

func generateVoteID() string {
	return "vote_" + time.Now().Format("20060102150405")
}
