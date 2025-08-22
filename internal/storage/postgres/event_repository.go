package postgres

import (
	"errors"

	"github.com/gravadigital/telescopio-api/internal/domain/event"
)

// InMemoryEventRepository is a temporary in-memory implementation
// TODO: Replace with actual PostgreSQL implementation
type InMemoryEventRepository struct {
	events map[string]*event.Event
}

func NewInMemoryEventRepository() *InMemoryEventRepository {
	return &InMemoryEventRepository{
		events: make(map[string]*event.Event),
	}
}

func (r *InMemoryEventRepository) Create(event *event.Event) error {
	r.events[event.ID] = event
	return nil
}

func (r *InMemoryEventRepository) GetByID(id string) (*event.Event, error) {
	event, exists := r.events[id]
	if !exists {
		return nil, errors.New("event not found")
	}
	return event, nil
}

func (r *InMemoryEventRepository) GetAll() ([]*event.Event, error) {
	events := make([]*event.Event, 0, len(r.events))
	for _, event := range r.events {
		events = append(events, event)
	}
	return events, nil
}

func (r *InMemoryEventRepository) GetByAuthor(authorID string) ([]*event.Event, error) {
	var events []*event.Event
	for _, event := range r.events {
		if event.AuthorID == authorID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (r *InMemoryEventRepository) GetByParticipant(participantID string) ([]*event.Event, error) {
	var events []*event.Event
	for _, event := range r.events {
		if event.IsParticipant(participantID) {
			events = append(events, event)
		}
	}
	return events, nil
}

func (r *InMemoryEventRepository) UpdateStage(eventID string, stage event.Stage) error {
	event, exists := r.events[eventID]
	if !exists {
		return errors.New("event not found")
	}

	if !event.UpdateStage(stage) {
		return errors.New("invalid stage transition")
	}

	return nil
}

func (r *InMemoryEventRepository) AddParticipant(eventID, userID string) error {
	event, exists := r.events[eventID]
	if !exists {
		return errors.New("event not found")
	}

	event.AddParticipant(userID)
	return nil
}

func (r *InMemoryEventRepository) RemoveParticipant(eventID, userID string) error {
	event, exists := r.events[eventID]
	if !exists {
		return errors.New("event not found")
	}

	event.RemoveParticipant(userID)
	return nil
}
