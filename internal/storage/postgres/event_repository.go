package postgres

import (
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresEventRepository implements EventRepository using GORM
type PostgresEventRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresEventRepository creates a new PostgreSQL event repository
func NewPostgresEventRepository(db *gorm.DB) *PostgresEventRepository {
	return &PostgresEventRepository{
		db:  db,
		log: logger.Repository("event"),
	}
}

func (r *PostgresEventRepository) Create(event *event.Event) error {
	r.log.Debug("creating new event", "event_id", event.ID, "name", event.Name)

	if err := r.db.Create(event).Error; err != nil {
		r.log.Error("failed to create event", "error", err, "event_id", event.ID)
		return err
	}

	r.log.Info("event created successfully", "event_id", event.ID, "name", event.Name)
	return nil
}

func (r *PostgresEventRepository) GetByID(id string) (*event.Event, error) {
	r.log.Debug("retrieving event by ID", "event_id", id)

	eventID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", id, "error", err)
		return nil, errors.New("invalid event ID format")
	}

	var evt event.Event
	if err := r.db.Preload("Author").Preload("Participants").Preload("Attachments").Preload("Votes").First(&evt, eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("event not found", "event_id", id)
			return nil, errors.New("event not found")
		}
		r.log.Error("failed to retrieve event", "event_id", id, "error", err)
		return nil, err
	}

	r.log.Debug("event retrieved successfully", "event_id", id, "name", evt.Name)
	return &evt, nil
}

func (r *PostgresEventRepository) GetAll() ([]*event.Event, error) {
	r.log.Debug("retrieving all events")

	var events []*event.Event
	if err := r.db.Preload("Author").Preload("Participants").Find(&events).Error; err != nil {
		r.log.Error("failed to retrieve events", "error", err)
		return nil, err
	}

	r.log.Debug("events retrieved successfully", "count", len(events))
	return events, nil
}

func (r *PostgresEventRepository) GetByAuthor(authorID string) ([]*event.Event, error) {
	r.log.Debug("retrieving events by author", "author_id", authorID)

	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		r.log.Error("invalid author ID format", "author_id", authorID, "error", err)
		return nil, errors.New("invalid author ID format")
	}

	var events []*event.Event
	if err := r.db.Preload("Author").Preload("Participants").Where("author_id = ?", authorUUID).Find(&events).Error; err != nil {
		r.log.Error("failed to retrieve events by author", "author_id", authorID, "error", err)
		return nil, err
	}

	r.log.Debug("events by author retrieved successfully", "author_id", authorID, "count", len(events))
	return events, nil
}

func (r *PostgresEventRepository) GetByParticipant(participantID string) ([]*event.Event, error) {
	participantUUID, err := uuid.Parse(participantID)
	if err != nil {
		return nil, errors.New("invalid participant ID format")
	}

	var events []*event.Event
	if err := r.db.Preload("Author").Preload("Participants").
		Joins("JOIN event_participants ON events.id = event_participants.event_id").
		Where("event_participants.user_id = ?", participantUUID).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *PostgresEventRepository) UpdateStage(eventID string, stage event.Stage) error {
	r.log.Debug("updating event stage", "event_id", eventID, "new_stage", stage.String())

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	if err := r.db.Model(&event.Event{}).Where("id = ?", eventUUID).Update("stage", stage).Error; err != nil {
		r.log.Error("failed to update event stage", "event_id", eventID, "new_stage", stage.String(), "error", err)
		return err
	}

	r.log.Info("event stage updated successfully", "event_id", eventID, "new_stage", stage.String())
	return nil
}

func (r *PostgresEventRepository) AddParticipant(eventID, userID string) error {
	r.log.Debug("adding participant to event", "event_id", eventID, "user_id", userID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID format", "user_id", userID, "error", err)
		return errors.New("invalid user ID format")
	}

	// Use GORM's association mode to add participant
	var evt event.Event
	if err := r.db.First(&evt, eventUUID).Error; err != nil {
		r.log.Error("event not found for participant addition", "event_id", eventID, "error", err)
		return errors.New("event not found")
	}

	var user participant.User
	if err := r.db.First(&user, userUUID).Error; err != nil {
		r.log.Error("user not found for participant addition", "user_id", userID, "error", err)
		return errors.New("user not found")
	}

	if err := r.db.Model(&evt).Association("Participants").Append(&user); err != nil {
		r.log.Error("failed to add participant to event", "event_id", eventID, "user_id", userID, "error", err)
		return err
	}

	r.log.Info("participant added to event successfully", "event_id", eventID, "user_id", userID)
	return nil
}

func (r *PostgresEventRepository) RemoveParticipant(eventID, userID string) error {
	r.log.Debug("removing participant from event", "event_id", eventID, "user_id", userID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID format", "user_id", userID, "error", err)
		return errors.New("invalid user ID format")
	}

	// Use GORM's association mode to remove participant
	var evt event.Event
	if err := r.db.First(&evt, eventUUID).Error; err != nil {
		r.log.Error("event not found for participant removal", "event_id", eventID, "error", err)
		return errors.New("event not found")
	}

	var user participant.User
	if err := r.db.First(&user, userUUID).Error; err != nil {
		r.log.Error("user not found for participant removal", "user_id", userID, "error", err)
		return errors.New("user not found")
	}

	if err := r.db.Model(&evt).Association("Participants").Delete(&user); err != nil {
		r.log.Error("failed to remove participant from event", "event_id", eventID, "user_id", userID, "error", err)
		return err
	}

	r.log.Info("participant removed from event successfully", "event_id", eventID, "user_id", userID)
	return nil
}

// Update updates an existing event
func (r *PostgresEventRepository) Update(event *event.Event) error {
	r.log.Debug("updating event", "event_id", event.ID, "name", event.Name)

	if err := r.db.Save(event).Error; err != nil {
		r.log.Error("failed to update event", "event_id", event.ID, "error", err)
		return err
	}

	r.log.Info("event updated successfully", "event_id", event.ID, "name", event.Name)
	return nil
}

// Delete deletes an event by ID
func (r *PostgresEventRepository) Delete(id string) error {
	r.log.Debug("deleting event", "event_id", id)

	eventUUID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", id, "error", err)
		return errors.New("invalid event ID format")
	}

	if err := r.db.Delete(&event.Event{}, eventUUID).Error; err != nil {
		r.log.Error("failed to delete event", "event_id", id, "error", err)
		return err
	}

	r.log.Info("event deleted successfully", "event_id", id)
	return nil
}

// InMemoryEventRepository - Simple in-memory implementation for development
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
	var events []*event.Event
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
		for _, pid := range event.ParticipantIDs {
			if pid == participantID {
				events = append(events, event)
				break
			}
		}
	}
	return events, nil
}

func (r *InMemoryEventRepository) Update(event *event.Event) error {
	r.events[event.ID] = event
	return nil
}

func (r *InMemoryEventRepository) Delete(id string) error {
	delete(r.events, id)
	return nil
}

func (r *InMemoryEventRepository) UpdateStage(eventID string, stage event.Stage) error {
	event, exists := r.events[eventID]
	if !exists {
		return errors.New("event not found")
	}
	event.Stage = stage
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
