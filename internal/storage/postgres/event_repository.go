package postgres

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
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
	if err := r.db.First(&evt, eventID).Error; err != nil {
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
	if err := r.db.Find(&events).Error; err != nil {
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
	if err := r.db.Where("author_id = ?", authorUUID).Find(&events).Error; err != nil {
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
	if err := r.db.Joins("JOIN event_participants ON events.id = event_participants.event_id").
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

func (r *PostgresEventRepository) Update(event *event.Event) error {
	r.log.Debug("updating event", "event_id", event.ID, "name", event.Name)

	// Validate event before updating
	if err := event.Validate(); err != nil {
		r.log.Error("event validation failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("event validation failed: %w", err)
	}

	if err := r.db.Save(event).Error; err != nil {
		r.log.Error("failed to update event", "event_id", event.ID, "error", err)
		return fmt.Errorf("failed to update event: %w", err)
	}

	r.log.Info("event updated successfully", "event_id", event.ID, "name", event.Name)
	return nil
}

func (r *PostgresEventRepository) Delete(id string) error {
	r.log.Debug("deleting event", "event_id", id)

	eventID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", id, "error", err)
		return errors.New("invalid event ID format")
	}

	// Start a transaction for safe deletion
	tx := r.db.Begin()
	if tx.Error != nil {
		r.log.Error("failed to start transaction for event deletion", "event_id", id, "error", tx.Error)
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Check if event exists
	var event event.Event
	if err := tx.First(&event, eventID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent event", "event_id", id)
			return errors.New("event not found")
		}
		r.log.Error("failed to check event existence for deletion", "event_id", id, "error", err)
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	// Delete related data first (in correct order due to foreign key constraints)
	// 1. Delete voting results
	if err := tx.Where("event_id = ?", eventID).Delete(&vote.VotingResults{}).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete voting results", "event_id", id, "error", err)
		return fmt.Errorf("failed to delete voting results: %w", err)
	}

	// 2. Delete voting configurations
	if err := tx.Where("event_id = ?", eventID).Delete(&vote.VotingConfiguration{}).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete voting configuration", "event_id", id, "error", err)
		return fmt.Errorf("failed to delete voting configuration: %w", err)
	}

	// 3. Delete votes
	if err := tx.Where("event_id = ?", eventID).Delete(&vote.Vote{}).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete votes", "event_id", id, "error", err)
		return fmt.Errorf("failed to delete votes: %w", err)
	}

	// 4. Delete assignments
	if err := tx.Where("event_id = ?", eventID).Delete(&vote.Assignment{}).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete assignments", "event_id", id, "error", err)
		return fmt.Errorf("failed to delete assignments: %w", err)
	}

	// 5. Delete attachments
	if err := tx.Where("event_id = ?", eventID).Delete(&attachment.Attachment{}).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete attachments", "event_id", id, "error", err)
		return fmt.Errorf("failed to delete attachments: %w", err)
	}

	// 6. Remove participant associations
	if err := tx.Model(&event).Association("Participants").Clear(); err != nil {
		tx.Rollback()
		r.log.Error("failed to clear participant associations", "event_id", id, "error", err)
		return fmt.Errorf("failed to clear participant associations: %w", err)
	}

	// 7. Finally delete the event
	if err := tx.Delete(&event).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete event", "event_id", id, "error", err)
		return fmt.Errorf("failed to delete event: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		r.log.Error("failed to commit event deletion transaction", "event_id", id, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("event deleted successfully", "event_id", id, "name", event.Name)
	return nil
}

func (r *PostgresEventRepository) GetAllPaginated(params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving events with pagination", "page", params.Page, "page_size", params.PageSize)

	// Set default values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100 // Maximum page size limit
	}

	offset := (params.Page - 1) * params.PageSize

	// Get total count
	var total int64
	if err := r.db.Model(&event.Event{}).Count(&total).Error; err != nil {
		r.log.Error("failed to count events", "error", err)
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// Get paginated events
	var events []*event.Event
	if err := r.db.Offset(offset).Limit(params.PageSize).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		r.log.Error("failed to retrieve paginated events", "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated events: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       events,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated events retrieved successfully",
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"total_pages", totalPages,
		"returned_count", len(events))

	return result, nil
}
