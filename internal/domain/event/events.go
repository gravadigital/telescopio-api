package event

import (
	"database/sql/driver"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Event represents a voting event for telescope time allocation
type Event struct {
	ID                            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name                          string     `json:"name" gorm:"not null"`
	Description                   string     `json:"description" gorm:"not null"`
	AuthorID                      uuid.UUID  `json:"author_id" gorm:"type:uuid;not null"`
	StartDate                     time.Time  `json:"start_date" gorm:"not null"`
	EndDate                       time.Time  `json:"end_date" gorm:"not null"`
	Organizer                     string     `json:"organizer" gorm:"default:''"`
	Stage                         Stage      `json:"stage" gorm:"type:event_stage;not null;default:'creation'"`
	MaxParticipants               *int       `json:"max_participants,omitempty" gorm:"default:null"`
	ParticipationEstimatedEndDate *time.Time `json:"participation_estimated_end_date,omitempty" gorm:"type:date"`
	VotingEstimatedEndDate        *time.Time `json:"voting_estimated_end_date,omitempty" gorm:"type:date"`
	ShareableLink                 string     `json:"shareable_link,omitempty" gorm:"default:''"`
	IsCancelled                   bool       `json:"is_cancelled" gorm:"default:false"`
	CreatedAt                     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                     time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (Event) TableName() string {
	return "events"
}

// BeforeCreate sets a UUID before creating the record
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// NewEvent creates a new event with the given parameters
func NewEvent(name, description string, authorID uuid.UUID, startDate, endDate time.Time, organizer string) *Event {
	id := uuid.New()
	return &Event{
		ID:            id,
		Name:          name,
		Description:   description,
		AuthorID:      authorID,
		StartDate:     startDate,
		EndDate:       endDate,
		Organizer:     organizer,
		Stage:         StageCreation,
		ShareableLink: "/events/" + id.String(),
		CreatedAt:     time.Now(),
	}
}

// IsAuthor checks if the given user ID is the author of this event
func (e *Event) IsAuthor(userID uuid.UUID) bool {
	return e.AuthorID == userID
}

// CanTransitionTo checks if the event can transition to a new stage
func (e *Event) CanTransitionTo(newStage Stage) bool {
	transitions := map[Stage][]Stage{
		StageCreation:      {StageParticipation},
		StageParticipation: {StageVoting},
		StageVoting:        {StageResult},
		StageResult:        {}, // NOTE: No transitions from Result
	}

	allowedTransitions, exists := transitions[e.Stage]
	if !exists {
		return false
	}

	return slices.Contains(allowedTransitions, newStage)
}

// UpdateStage updates the stage if the transition is valid
func (e *Event) UpdateStage(newStage Stage) error {
	if !e.CanTransitionTo(newStage) {
		return fmt.Errorf("cannot transition from %s to %s", e.Stage, newStage)
	}
	e.Stage = newStage
	return nil
}

// Cancel marks the event as cancelled.
func (e *Event) Cancel() {
	e.IsCancelled = true
}

// Validate checks if the event data is valid
func (e *Event) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.Description == "" {
		return fmt.Errorf("description is required")
	}
	if e.AuthorID == uuid.Nil {
		return fmt.Errorf("author_id is required")
	}
	if e.EndDate.Before(e.StartDate) {
		return fmt.Errorf("end_date must be after start_date")
	}
	return nil
}

// Implement common.EventInterface for consistency with other domains
func (e *Event) GetID() uuid.UUID {
	return e.ID
}

func (e *Event) GetName() string {
	return e.Name
}

// Stage represents the current stage of an event
type Stage string

const (
	StageCreation      Stage = "creation"
	StageParticipation Stage = "participation"
	StageVoting        Stage = "voting"
	StageResult        Stage = "results"
)

func (s Stage) String() string {
	return string(s)
}

// MarshalJSON implements the json.Marshaler interface
func (s Stage) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (s *Stage) UnmarshalJSON(data []byte) error {
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	stage, valid := StageFromString(str)
	if !valid {
		return fmt.Errorf("invalid stage: %s", str)
	}
	*s = stage
	return nil
}

// StageFromString converts a string to a Stage
// StageFromString converts a string to a Stage
func StageFromString(s string) (Stage, bool) {
	switch s {
	case "creation":
		return StageCreation, true
	case "participation":
		return StageParticipation, true
	case "voting":
		return StageVoting, true
	case "results":
		return StageResult, true
	default:
		return StageCreation, false
	}
}
// Scan implements the sql.Scanner interface for database deserialization
func (s *Stage) Scan(value interface{}) error {
	if value == nil {
		*s = StageCreation
		return nil
	}

	str, ok := value.(string)
	if !ok {
		// Try to convert bytes to string
		if b, ok := value.([]byte); ok {
			str = string(b)
		} else {
			return fmt.Errorf("cannot scan %T into Stage", value)
		}
	}

	stage, valid := StageFromString(str)
	if !valid {
		return fmt.Errorf("invalid stage value: %s", str)
	}
	*s = stage
	return nil
}

// Value implements the driver.Valuer interface for database serialization
func (s Stage) Value() (driver.Value, error) {
	return string(s), nil
}
