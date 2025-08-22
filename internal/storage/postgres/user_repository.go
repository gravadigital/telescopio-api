package postgres

import (
	"errors"

	"github.com/gravadigital/telescopio-api/internal/domain/participant"
)

// InMemoryUserRepository is a temporary in-memory implementation
// TODO: Replace with actual PostgreSQL implementation
type InMemoryUserRepository struct {
	users      map[string]*participant.User
	emailIndex map[string]string // email -> user_id
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:      make(map[string]*participant.User),
		emailIndex: make(map[string]string),
	}
}

func (r *InMemoryUserRepository) Create(user *participant.User) error {
	r.users[user.ID] = user
	// TODO: In real implementation, email should be unique
	return nil
}

func (r *InMemoryUserRepository) GetByID(id string) (*participant.User, error) {
	user, exists := r.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (r *InMemoryUserRepository) GetByEmail(email string) (*participant.User, error) {
	userID, exists := r.emailIndex[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return r.GetByID(userID)
}

func (r *InMemoryUserRepository) GetAll() ([]*participant.User, error) {
	users := make([]*participant.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

func (r *InMemoryUserRepository) Update(user *participant.User) error {
	_, exists := r.users[user.ID]
	if !exists {
		return errors.New("user not found")
	}
	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepository) GetEventParticipants(eventID string) ([]*participant.User, error) {
	var participants []*participant.User
	for _, user := range r.users {
		if user.HasJoinedEvent(eventID) {
			participants = append(participants, user)
		}
	}
	return participants, nil
}
