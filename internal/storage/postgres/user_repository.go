package postgres

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresUserRepository implements UserRepository using GORM
type PostgresUserRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db:  db,
		log: logger.Repository("user"),
	}
}

func (r *PostgresUserRepository) Create(user *participant.User) error {
	r.log.Debug("Creating user", "email", user.Email, "name", user.Name)

	// Validate user before creating
	if err := user.Validate(); err != nil {
		r.log.Error("User validation failed", "error", err)
		return fmt.Errorf("user validation failed: %w", err)
	}

	// Check if user with email already exists
	var existingUser participant.User
	if err := r.db.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		r.log.Error("User with email already exists", "email", user.Email)
		return fmt.Errorf("user with email %s already exists", user.Email)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Error("Failed to check existing user", "email", user.Email, "error", err)
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if err := r.db.Create(user).Error; err != nil {
		r.log.Error("Failed to create user", "error", err, "email", user.Email)
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.log.Info("User created successfully", "id", user.ID, "email", user.Email)
	return nil
}

func (r *PostgresUserRepository) GetByID(id string) (*participant.User, error) {
	r.log.Debug("retrieving user by ID", "user_id", id)

	userID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("Invalid user ID format", "id", id, "error", err)
		return nil, errors.New("invalid user ID format")
	}

	var user participant.User
	if err := r.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("User not found", "id", id)
			return nil, errors.New("user not found")
		}
		r.log.Error("Failed to get user by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	r.log.Debug("User retrieved successfully", "id", user.ID, "email", user.Email)
	return &user, nil
}

func (r *PostgresUserRepository) GetByEmail(email string) (*participant.User, error) {
	r.log.Debug("retrieving user by email", "email", email)

	if email == "" {
		r.log.Error("empty email provided")
		return nil, errors.New("email cannot be empty")
	}

	var user participant.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("User not found", "email", email)
			return nil, errors.New("user not found")
		}
		r.log.Error("Failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	r.log.Debug("User retrieved successfully", "id", user.ID, "email", user.Email)
	return &user, nil
}

func (r *PostgresUserRepository) GetAll() ([]*participant.User, error) {
	var users []*participant.User
	if err := r.db.Find(&users).Error; err != nil {
		r.log.Error("Failed to get all users", "error", err)
		return nil, err
	}

	r.log.Debug("Retrieved all users", "count", len(users))
	return users, nil
}

func (r *PostgresUserRepository) GetAllPaginated(params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving users with pagination", "page", params.Page, "page_size", params.PageSize)

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
	if err := r.db.Model(&participant.User{}).Count(&total).Error; err != nil {
		r.log.Error("failed to count users", "error", err)
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated users
	var users []*participant.User
	if err := r.db.Offset(offset).Limit(params.PageSize).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		r.log.Error("failed to retrieve paginated users", "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated users: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       users,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated users retrieved successfully",
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"total_pages", totalPages,
		"returned_count", len(users))

	return result, nil
}

func (r *PostgresUserRepository) Update(user *participant.User) error {
	r.log.Debug("Updating user", "id", user.ID, "email", user.Email)

	// Validate user before updating
	if err := user.Validate(); err != nil {
		r.log.Error("User validation failed", "error", err)
		return fmt.Errorf("user validation failed: %w", err)
	}

	// Check if user exists
	var existingUser participant.User
	if err := r.db.First(&existingUser, user.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("User not found for update", "id", user.ID)
			return errors.New("user not found")
		}
		r.log.Error("Failed to check user existence for update", "id", user.ID, "error", err)
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// Check if another user with the same email exists (excluding current user)
	var emailUser participant.User
	if err := r.db.Where("email = ? AND id != ?", user.Email, user.ID).First(&emailUser).Error; err == nil {
		r.log.Error("Another user with email already exists", "email", user.Email, "current_id", user.ID)
		return fmt.Errorf("another user with email %s already exists", user.Email)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Error("Failed to check email uniqueness", "email", user.Email, "error", err)
		return fmt.Errorf("failed to check email uniqueness: %w", err)
	}

	if err := r.db.Save(user).Error; err != nil {
		r.log.Error("Failed to update user", "error", err, "id", user.ID)
		return fmt.Errorf("failed to update user: %w", err)
	}

	r.log.Info("User updated successfully", "id", user.ID, "email", user.Email)
	return nil
}

func (r *PostgresUserRepository) Delete(id string) error {
	r.log.Debug("deleting user", "user_id", id)

	userID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid user ID format", "user_id", id, "error", err)
		return errors.New("invalid user ID format")
	}

	// Start a transaction for safe deletion
	tx := r.db.Begin()
	if tx.Error != nil {
		r.log.Error("failed to start transaction for user deletion", "user_id", id, "error", tx.Error)
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Check if user exists
	var user participant.User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent user", "user_id", id)
			return errors.New("user not found")
		}
		r.log.Error("failed to check user existence for deletion", "user_id", id, "error", err)
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// Remove from event participations first (many-to-many relationship)
	if err := tx.Exec("DELETE FROM event_participants WHERE user_id = ?", userID).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to remove user from event participations", "user_id", id, "error", err)
		return fmt.Errorf("failed to remove user from event participations: %w", err)
	}

	// Note: We don't delete user's votes, assignments, or attachments as they are important for data integrity
	// Those should be soft-deleted or archived if needed, or handled by cascade rules

	// Delete the user
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		r.log.Error("failed to delete user", "user_id", id, "error", err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		r.log.Error("failed to commit user deletion transaction", "user_id", id, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("user deleted successfully", "user_id", id, "email", user.Email)
	return nil
}

func (r *PostgresUserRepository) GetEventParticipants(eventID string) ([]*participant.User, error) {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("Invalid event ID format", "eventID", eventID, "error", err)
		return nil, errors.New("invalid event ID format")
	}

	var users []*participant.User
	if err := r.db.Joins("JOIN event_participants ON users.id = event_participants.user_id").
		Where("event_participants.event_id = ?", eventUUID).
		Find(&users).Error; err != nil {
		r.log.Error("Failed to get event participants", "eventID", eventID, "error", err)
		return nil, err
	}

	r.log.Debug("Retrieved event participants", "eventID", eventID, "count", len(users))
	return users, nil
}

func (r *PostgresUserRepository) GetEventParticipantsPaginated(eventID string, params PaginationParams) (*PaginatedResult, error) {
	r.log.Debug("retrieving event participants with pagination", "event_id", eventID, "page", params.Page, "page_size", params.PageSize)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, errors.New("invalid event ID format")
	}

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

	// Get total count of participants for this event
	var total int64
	if err := r.db.Model(&participant.User{}).
		Joins("JOIN event_participants ON users.id = event_participants.user_id").
		Where("event_participants.event_id = ?", eventUUID).
		Count(&total).Error; err != nil {
		r.log.Error("failed to count event participants", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to count event participants: %w", err)
	}

	// Get paginated participants
	var users []*participant.User
	if err := r.db.Joins("JOIN event_participants ON users.id = event_participants.user_id").
		Where("event_participants.event_id = ?", eventUUID).
		Offset(offset).Limit(params.PageSize).
		Order("event_participants.joined_at DESC").
		Find(&users).Error; err != nil {
		r.log.Error("failed to retrieve paginated event participants", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve paginated event participants: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	result := &PaginatedResult{
		Data:       users,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}

	r.log.Debug("paginated event participants retrieved successfully",
		"event_id", eventID,
		"page", params.Page,
		"page_size", params.PageSize,
		"total", total,
		"total_pages", totalPages,
		"returned_count", len(users))

	return result, nil
}

// EmailExists checks if a user with the given email already exists
func (r *PostgresUserRepository) EmailExists(email string) (bool, error) {
	r.log.Debug("checking if email exists", "email", email)

	if email == "" {
		return false, errors.New("email cannot be empty")
	}

	var count int64
	if err := r.db.Model(&participant.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		r.log.Error("failed to check email existence", "email", email, "error", err)
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	exists := count > 0
	r.log.Debug("email existence check completed", "email", email, "exists", exists)
	return exists, nil
}

// IsEventParticipant checks if a user is a participant in the given event
func (r *PostgresUserRepository) IsEventParticipant(userID, eventID string) (bool, error) {
	r.log.Debug("checking if user is event participant", "user_id", userID, "event_id", eventID)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID format", "user_id", userID, "error", err)
		return false, errors.New("invalid user ID format")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return false, errors.New("invalid event ID format")
	}

	var count int64
	if err := r.db.Table("event_participants").
		Where("user_id = ? AND event_id = ?", userUUID, eventUUID).
		Count(&count).Error; err != nil {
		r.log.Error("failed to check event participation", "user_id", userID, "event_id", eventID, "error", err)
		return false, fmt.Errorf("failed to check event participation: %w", err)
	}

	isParticipant := count > 0
	r.log.Debug("event participation check completed", "user_id", userID, "event_id", eventID, "is_participant", isParticipant)
	return isParticipant, nil
}

// GetUserStatistics returns statistics about a user's participation
func (r *PostgresUserRepository) GetUserStatistics(userID string) (map[string]interface{}, error) {
	r.log.Debug("getting user statistics", "user_id", userID)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID format", "user_id", userID, "error", err)
		return nil, errors.New("invalid user ID format")
	}

	// Check if user exists
	var user participant.User
	if err := r.db.First(&user, userUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("user not found for statistics", "user_id", userID)
			return nil, errors.New("user not found")
		}
		r.log.Error("failed to check user existence for statistics", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	stats := make(map[string]interface{})

	// Count events participated in
	var eventCount int64
	if err := r.db.Table("event_participants").
		Where("user_id = ?", userUUID).
		Count(&eventCount).Error; err != nil {
		r.log.Error("failed to count user events", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to count user events: %w", err)
	}
	stats["total_events"] = eventCount

	// Count attachments uploaded
	var attachmentCount int64
	if err := r.db.Table("attachments").
		Where("participant_id = ?", userUUID).
		Count(&attachmentCount).Error; err != nil {
		r.log.Error("failed to count user attachments", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to count user attachments: %w", err)
	}
	stats["total_attachments"] = attachmentCount

	// Count votes cast
	var voteCount int64
	if err := r.db.Table("votes").
		Where("voter_id = ?", userUUID).
		Count(&voteCount).Error; err != nil {
		r.log.Error("failed to count user votes", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to count user votes: %w", err)
	}
	stats["total_votes"] = voteCount

	// Count assignments received
	var assignmentCount int64
	if err := r.db.Table("assignments").
		Where("participant_id = ?", userUUID).
		Count(&assignmentCount).Error; err != nil {
		r.log.Error("failed to count user assignments", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to count user assignments: %w", err)
	}
	stats["total_assignments"] = assignmentCount

	stats["user_id"] = userID
	stats["email"] = user.Email
	stats["name"] = user.Name

	r.log.Debug("user statistics retrieved successfully", "user_id", userID, "stats", stats)
	return stats, nil
}
