package postgres

import (
	"errors"
	"fmt"
	"time"

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

// GetUserParticipatingEvents returns all events where a user participates (excluding events they created)
func (r *PostgresEventRepository) GetUserParticipatingEvents(userID string) ([]*event.Event, error) {
	r.log.Debug("retrieving participating events for user", "user_id", userID)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID format", "user_id", userID, "error", err)
		return nil, errors.New("invalid user ID format")
	}

	var events []*event.Event
	err = r.db.
		Joins("JOIN event_participants ON events.id = event_participants.event_id").
		Where("event_participants.user_id = ?", userUUID).
		Where("events.author_id != ?", userUUID).
		Order("events.created_at DESC").
		Find(&events).Error

	if err != nil {
		r.log.Error("failed to get user participating events", "user_id", userID, "error", err)
		return nil, err
	}

	r.log.Debug("retrieved participating events", "user_id", userID, "count", len(events))
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

// UpdateStageWithEstimatedDate updates event stage and sets the estimated end date for that stage
func (r *PostgresEventRepository) UpdateStageWithEstimatedDate(eventID string, stage event.Stage, estimatedDate *time.Time) error {
	r.log.Debug("updating event stage with estimated date", "event_id", eventID, "stage", stage.String())

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	updates := map[string]interface{}{
		"stage": stage,
	}

	// Set the appropriate estimated date field based on stage
	if estimatedDate != nil {
		switch stage {
		case event.StageParticipation:
			updates["participation_estimated_end_date"] = estimatedDate
		case event.StageVoting:
			updates["voting_estimated_end_date"] = estimatedDate
		}
	}

	if err := r.db.Model(&event.Event{}).Where("id = ?", eventUUID).Updates(updates).Error; err != nil {
		r.log.Error("failed to update event stage with estimated date", "event_id", eventID, "error", err)
		return err
	}

	r.log.Info("event stage updated with estimated date", "event_id", eventID, "stage", stage.String())
	return nil
}

// UpdateEstimatedEndDate updates only the estimated end date for a specific stage
func (r *PostgresEventRepository) UpdateEstimatedEndDate(eventID string, stage event.Stage, newDate time.Time) error {
	r.log.Debug("updating estimated end date", "event_id", eventID, "stage", stage.String(), "new_date", newDate)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	var columnName string
	switch stage {
	case event.StageParticipation:
		columnName = "participation_estimated_end_date"
	case event.StageVoting:
		columnName = "voting_estimated_end_date"
	default:
		return fmt.Errorf("invalid stage for estimated end date: %s", stage)
	}

	if err := r.db.Model(&event.Event{}).Where("id = ?", eventUUID).Update(columnName, newDate).Error; err != nil {
		r.log.Error("failed to update estimated end date", "event_id", eventID, "error", err)
		return err
	}

	r.log.Info("estimated end date updated", "event_id", eventID, "stage", stage.String())
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

	// Verify event exists
	var evt event.Event
	if err := r.db.First(&evt, eventUUID).Error; err != nil {
		r.log.Error("event not found for participant addition", "event_id", eventID, "error", err)
		return errors.New("event not found")
	}

	// Verify user exists
	var user participant.User
	if err := r.db.First(&user, userUUID).Error; err != nil {
		r.log.Error("user not found for participant addition", "user_id", userID, "error", err)
		return errors.New("user not found")
	}

	// Insert directly into event_participants junction table
	// Use raw SQL to avoid GORM association issues
	query := `
		INSERT INTO event_participants (event_id, user_id, joined_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (event_id, user_id) DO NOTHING
	`

	if err := r.db.Exec(query, eventUUID, userUUID).Error; err != nil {
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

// AddParticipantWithRole adds a participant to an event with a specific role
func (r *PostgresEventRepository) AddParticipantWithRole(eventID, userID string, role event.EventParticipantRole) error {
	r.log.Debug("adding participant with role to event", "event_id", eventID, "user_id", userID, "role", role)

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

	// Verify event exists
	var evt event.Event
	if err := r.db.First(&evt, eventUUID).Error; err != nil {
		r.log.Error("event not found for participant addition", "event_id", eventID, "error", err)
		return errors.New("event not found")
	}

	// Verify user exists
	var user participant.User
	if err := r.db.First(&user, userUUID).Error; err != nil {
		r.log.Error("user not found for participant addition", "user_id", userID, "error", err)
		return errors.New("user not found")
	}

	// Insert with role
	query := `
		INSERT INTO event_participants (event_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (event_id, user_id) DO UPDATE SET role = $3
	`

	if err := r.db.Exec(query, eventUUID, userUUID, role.String()).Error; err != nil {
		r.log.Error("failed to add participant with role to event", "event_id", eventID, "user_id", userID, "role", role, "error", err)
		return err
	}

	r.log.Info("participant added to event with role", "event_id", eventID, "user_id", userID, "role", role)
	return nil
}

// GetParticipantRole gets the role of a user in a specific event
func (r *PostgresEventRepository) GetParticipantRole(eventID, userID string) (*event.EventParticipantRole, error) {
	r.log.Debug("getting participant role", "event_id", eventID, "user_id", userID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, errors.New("invalid event ID format")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	var roleStr string
	query := `SELECT role FROM event_participants WHERE event_id = $1 AND user_id = $2`

	if err := r.db.Raw(query, eventUUID, userUUID).Scan(&roleStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("participant not found in event")
		}
		return nil, err
	}

	role := event.EventParticipantRole(roleStr)
	return &role, nil
}

// IsEventCreator checks if a user is the creator of an event
func (r *PostgresEventRepository) IsEventCreator(eventID, userID string) (bool, error) {
	role, err := r.GetParticipantRole(eventID, userID)
	if err != nil {
		return false, err
	}
	return *role == event.RoleCreator, nil
}

// IsEventParticipant checks if a user is a participant (any role) in an event
func (r *PostgresEventRepository) IsEventParticipant(eventID, userID string) (bool, error) {
	_, err := r.GetParticipantRole(eventID, userID)
	if err != nil {
		if err.Error() == "participant not found in event" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// PauseEvent toggles the is_paused field of an event.
func (r *PostgresEventRepository) PauseEvent(eventID string, paused bool) error {
	r.log.Debug("toggling event pause", "event_id", eventID, "paused", paused)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	if err := r.db.Model(&event.Event{}).Where("id = ?", eventUUID).Update("is_paused", paused).Error; err != nil {
		r.log.Error("failed to toggle event pause", "event_id", eventID, "error", err)
		return err
	}

	r.log.Info("event pause toggled", "event_id", eventID, "paused", paused)
	return nil
}

// CancelEvent marks an event as cancelled by setting is_cancelled = true.
func (r *PostgresEventRepository) CancelEvent(eventID string) error {
	r.log.Debug("cancelling event", "event_id", eventID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	if err := r.db.Model(&event.Event{}).Where("id = ?", eventUUID).Update("is_cancelled", true).Error; err != nil {
		r.log.Error("failed to cancel event", "event_id", eventID, "error", err)
		return err
	}

	r.log.Info("event cancelled successfully", "event_id", eventID)
	return nil
}
