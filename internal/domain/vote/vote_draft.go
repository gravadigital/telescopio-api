package vote

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DraftRanking represents a single ranking selection within a draft
type DraftRanking struct {
	AttachmentID uuid.UUID `json:"attachment_id"`
	Rank         int       `json:"rank"`
}

// DraftRankings is a slice of DraftRanking with JSONB serialization support
type DraftRankings []DraftRanking

func (d DraftRankings) Value() (driver.Value, error) {
	if d == nil {
		return "[]", nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DraftRankings: %w", err)
	}
	return string(b), nil
}

func (d *DraftRankings) Scan(value interface{}) error {
	if value == nil {
		*d = DraftRankings{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan DraftRankings: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, d)
}

// VoteDraft stores a participant's partial voting selections before final submission.
// There is at most one draft per (assignment_id, participant_id) pair.
type VoteDraft struct {
	ID            uuid.UUID    `json:"id"             gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EventID       uuid.UUID    `json:"event_id"       gorm:"type:uuid;not null"`
	AssignmentID  uuid.UUID    `json:"assignment_id"  gorm:"type:uuid;not null"`
	ParticipantID uuid.UUID    `json:"participant_id" gorm:"type:uuid;not null"`
	Rankings      DraftRankings `json:"rankings"      gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt     time.Time    `json:"created_at"     gorm:"autoCreateTime"`
	UpdatedAt     time.Time    `json:"updated_at"     gorm:"autoUpdateTime"`
}

func (VoteDraft) TableName() string {
	return "vote_drafts"
}

func (v *VoteDraft) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
