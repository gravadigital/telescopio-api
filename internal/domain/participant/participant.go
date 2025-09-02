package participant

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Email           string    `json:"email" db:"email"`
	LastName        string    `json:"lastname" db:"lastname"`
	JoinedEventIDs  []string  `json:"joined_event_ids" db:"joined_event_ids"`
	CreatedEventIDs []string  `json:"created_event_ids" db:"created_event_ids"`
	Role            string    `json:"role" db:"role"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// NewUser crea un nuevo usuario
func NewUser(name, email string) *User {
	return &User{
		ID:              uuid.New().String(),
		Name:            name,
		Email:           email,
		JoinedEventIDs:  make([]string, 0),
		CreatedEventIDs: make([]string, 0),
		Role:            "participant",
		CreatedAt:       time.Now(),
	}
}

// JoinEvent agrega un evento a la lista de eventos en los que participa
func (u *User) JoinEvent(eventID string) {
	if slices.Contains(u.JoinedEventIDs, eventID) {
		return // already register
	}
	u.JoinedEventIDs = append(u.JoinedEventIDs, eventID)
}

// LeaveEvent remueve un evento de la lista de eventos en los que participa
func (u *User) LeaveEvent(eventID string) {
	for i, joinedEventID := range u.JoinedEventIDs {
		if joinedEventID == eventID {
			u.JoinedEventIDs = append(u.JoinedEventIDs[:i], u.JoinedEventIDs[i+1:]...)
			return
		}
	}
}

// CreateEvent agrega un evento a la lista de eventos creados
func (u *User) CreateEvent(eventID string) {
	u.CreatedEventIDs = append(u.CreatedEventIDs, eventID)
}

// HasJoinedEvent verifica si el usuario está registrado en un evento
func (u *User) HasJoinedEvent(eventID string) bool {
	return slices.Contains(u.JoinedEventIDs, eventID)
}

// HasCreatedEvent verifica si el usuario creó un evento específico
func (u *User) HasCreatedEvent(eventID string) bool {
	return slices.Contains(u.CreatedEventIDs, eventID)
}
