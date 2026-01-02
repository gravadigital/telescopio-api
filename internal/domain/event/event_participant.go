package event

import (
	"time"

	"github.com/google/uuid"
)

// EventParticipantRole represents the role of a user within a specific event
type EventParticipantRole string

const (
	RoleCreator     EventParticipantRole = "creator"
	RoleParticipant EventParticipantRole = "participant"
)

// String returns the string representation of the role
func (r EventParticipantRole) String() string {
	return string(r)
}

// IsValid checks if the role is valid
func (r EventParticipantRole) IsValid() bool {
	return r == RoleCreator || r == RoleParticipant
}

// EventParticipant represents the relationship between a user and an event with a role
type EventParticipant struct {
	EventID  uuid.UUID            `json:"event_id" gorm:"type:uuid;primaryKey"`
	UserID   uuid.UUID            `json:"user_id" gorm:"type:uuid;primaryKey"`
	Role     EventParticipantRole `json:"role" gorm:"type:event_participant_role;not null;default:'participant'"`
	JoinedAt time.Time            `json:"joined_at" gorm:"autoCreateTime"`
}

// TableName overrides the table name used by GORM
func (EventParticipant) TableName() string {
	return "event_participants"
}

// IsCreator checks if this participant is the creator of the event
func (ep *EventParticipant) IsCreator() bool {
	return ep.Role == RoleCreator
}

// IsParticipant checks if this is a regular participant
func (ep *EventParticipant) IsParticipant() bool {
	return ep.Role == RoleParticipant
}

// CanManageEvent checks if this participant can manage the event
// (change stages, configure voting, etc.)
func (ep *EventParticipant) CanManageEvent() bool {
	return ep.IsCreator()
}

// CanVote checks if this participant can vote
// (all participants can vote, including creators)
func (ep *EventParticipant) CanVote() bool {
	return true // Both creators and participants can vote
}
