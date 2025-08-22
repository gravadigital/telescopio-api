package event

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/gravadigital/telescopio-api/internal/domain/participant"
)

type Event struct {
	ID             string             `json:"id" db:"id"`
	Name           string             `json:"name" db:"name"`
	AuthorID       string             `json:"author_id" db:"author_id"`
	Author         *participant.User  `json:"author,omitempty" db:"-"`
	Description    string             `json:"description" db:"description"`
	StartDate      time.Time          `json:"start_date" db:"start_date"`
	EndDate        time.Time          `json:"end_date" db:"end_date"`
	Stage          Stage              `json:"stage" db:"stage"`
	ParticipantIDs []string           `json:"participant_ids" db:"participant_ids"`
	Participants   []participant.User `json:"participants,omitempty" db:"-"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
}

func NewEvent(name, description, authorID string, startDate, endDate time.Time) *Event {
	return &Event{
		ID:             generateEventID(),
		Name:           name,
		Description:    description,
		AuthorID:       authorID,
		StartDate:      startDate,
		EndDate:        endDate,
		Stage:          StageCreation,
		ParticipantIDs: make([]string, 0),
		CreatedAt:      time.Now(),
	}
}

// AddParticipant agrega un participante al evento
func (e *Event) AddParticipant(userID string) {
	if slices.Contains(e.ParticipantIDs, userID) {
		return
	}
	e.ParticipantIDs = append(e.ParticipantIDs, userID)
}

// RemoveParticipant remueve un participante del evento
func (e *Event) RemoveParticipant(userID string) {
	for i, participantID := range e.ParticipantIDs {
		if participantID == userID {
			e.ParticipantIDs = append(e.ParticipantIDs[:i], e.ParticipantIDs[i+1:]...)
			return
		}
	}
}

// IsParticipant verifica si un usuario es participante
func (e *Event) IsParticipant(userID string) bool {
	return slices.Contains(e.ParticipantIDs, userID)
}

// IsAuthor verifica si un usuario es el autor del evento
func (e *Event) IsAuthor(userID string) bool {
	return e.AuthorID == userID
}

// CanTransitionTo verifica si el evento puede transicionar a un nuevo stage
func (e *Event) CanTransitionTo(newStage Stage) bool {
	transitions := map[Stage][]Stage{
		StageCreation:     {StageRegistration},
		StageRegistration: {StageSubmission},
		StageSubmission:   {StageVoting},
		StageVoting:       {StageResult},
		StageResult:       {}, // No hay transiciones desde Result
	}

	allowedTransitions, exists := transitions[e.Stage]
	if !exists {
		return false
	}

	return slices.Contains(allowedTransitions, newStage)
}

// UpdateStage actualiza el stage del evento si la transición es válida
func (e *Event) UpdateStage(newStage Stage) bool {
	if e.CanTransitionTo(newStage) {
		e.Stage = newStage
		return true
	}
	return false
}

// TODO: Implementar esta función
func generateEventID() string {
	// Por ahora retorna un placeholder
	// Podrías usar UUID, nanoid, etc.
	return "event_" + strconv.FormatInt(time.Now().Unix(), 10)
}

func (e Event) New() Event {
	return Event{
		ID:             e.ID,
		Name:           e.Name,
		AuthorID:       e.AuthorID,
		Description:    e.Description,
		StartDate:      e.StartDate,
		EndDate:        e.EndDate,
		Stage:          e.Stage,
		ParticipantIDs: make([]string, len(e.ParticipantIDs)),
		Participants:   make([]participant.User, len(e.Participants)),
		CreatedAt:      e.CreatedAt,
	}
}

type Stage byte

const (
	StageCreation Stage = iota
	StageRegistration
	StageSubmission
	StageVoting
	StageResult
	// StageDisabled
)

func (s Stage) String() string {
	switch s {
	case StageCreation:
		return "creation"
	case StageRegistration:
		return "registration"
	case StageSubmission:
		return "attachment_upload"
	case StageVoting:
		return "voting"
	case StageResult:
		return "results"
	default:
		return "unknown"
	}
}

// MarshalJSON implements the json.Marshaler interface
func (s Stage) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (s *Stage) UnmarshalJSON(data []byte) error {
	str := string(data)
	// Remove quotes
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
func StageFromString(s string) (Stage, bool) {
	switch s {
	case "creation":
		return StageCreation, true
	case "registration":
		return StageRegistration, true
	case "attachment_upload":
		return StageSubmission, true
	case "voting":
		return StageVoting, true
	case "results":
		return StageResult, true
	default:
		return StageCreation, false
	}
}
