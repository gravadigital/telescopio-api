package postgres

import (
	"errors"
	"fmt"
	"math"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresVotingConfigurationRepository implements VotingConfigurationRepository using GORM
type PostgresVotingConfigurationRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresVotingConfigurationRepository creates a new PostgreSQL voting configuration repository
func NewPostgresVotingConfigurationRepository(db *gorm.DB) *PostgresVotingConfigurationRepository {
	return &PostgresVotingConfigurationRepository{
		db:  db,
		log: logger.Repository("voting_configuration"),
	}
}

func (r *PostgresVotingConfigurationRepository) Create(config *vote.VotingConfiguration) error {
	r.log.Debug("creating new voting configuration", "config_id", config.ID, "event_id", config.EventID)

	// Check if configuration already exists for this event
	var existingConfig vote.VotingConfiguration
	if err := r.db.Where("event_id = ?", config.EventID).First(&existingConfig).Error; err == nil {
		r.log.Error("voting configuration already exists for event", "event_id", config.EventID, "existing_config_id", existingConfig.ID)
		return fmt.Errorf("voting configuration already exists for event %s", config.EventID.String())
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Error("failed to check existing configuration", "event_id", config.EventID, "error", err)
		return fmt.Errorf("failed to check existing configuration: %w", err)
	}

	// Validate configuration before creating
	if err := r.ValidateConfiguration(config); err != nil {
		r.log.Error("voting configuration validation failed", "error", err, "config_id", config.ID)
		return fmt.Errorf("voting configuration validation failed: %w", err)
	}

	if err := r.db.Create(config).Error; err != nil {
		r.log.Error("failed to create voting configuration", "error", err, "config_id", config.ID)
		return fmt.Errorf("failed to create voting configuration: %w", err)
	}

	r.log.Info("voting configuration created successfully", "config_id", config.ID, "event_id", config.EventID)
	return nil
}

func (r *PostgresVotingConfigurationRepository) GetByID(id string) (*vote.VotingConfiguration, error) {
	r.log.Debug("retrieving voting configuration by ID", "config_id", id)

	configID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid configuration ID format", "config_id", id, "error", err)
		return nil, errors.New("invalid configuration ID format")
	}

	var config vote.VotingConfiguration
	if err := r.db.Preload("Event").First(&config, configID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("voting configuration not found", "config_id", id)
			return nil, errors.New("voting configuration not found")
		}
		r.log.Error("failed to retrieve voting configuration", "config_id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve voting configuration: %w", err)
	}

	r.log.Debug("voting configuration retrieved successfully", "config_id", id, "event_id", config.EventID)
	return &config, nil
}

func (r *PostgresVotingConfigurationRepository) GetByEventID(eventID string) (*vote.VotingConfiguration, error) {
	r.log.Debug("retrieving voting configuration by event ID", "event_id", eventID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return nil, errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, errors.New("invalid event ID format")
	}

	var config vote.VotingConfiguration
	if err := r.db.Preload("Event").Where("event_id = ?", eventUUID).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("voting configuration not found", "event_id", eventID)
			return nil, errors.New("voting configuration not found")
		}
		r.log.Error("failed to retrieve voting configuration", "event_id", eventID, "error", err)
		return nil, fmt.Errorf("failed to retrieve voting configuration: %w", err)
	}

	r.log.Debug("voting configuration retrieved successfully", "event_id", eventID, "config_id", config.ID)
	return &config, nil
}

func (r *PostgresVotingConfigurationRepository) Update(config *vote.VotingConfiguration) error {
	r.log.Debug("updating voting configuration", "config_id", config.ID)

	// Validate configuration before updating
	if err := r.ValidateConfiguration(config); err != nil {
		r.log.Error("voting configuration validation failed", "error", err, "config_id", config.ID)
		return fmt.Errorf("voting configuration validation failed: %w", err)
	}

	// Check if configuration exists
	var existingConfig vote.VotingConfiguration
	if err := r.db.First(&existingConfig, config.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("voting configuration not found for update", "config_id", config.ID)
			return errors.New("voting configuration not found")
		}
		r.log.Error("failed to check configuration existence for update", "config_id", config.ID, "error", err)
		return fmt.Errorf("failed to check configuration existence: %w", err)
	}

	if err := r.db.Save(config).Error; err != nil {
		r.log.Error("failed to update voting configuration", "error", err, "config_id", config.ID)
		return fmt.Errorf("failed to update voting configuration: %w", err)
	}

	r.log.Info("voting configuration updated successfully", "config_id", config.ID)
	return nil
}

func (r *PostgresVotingConfigurationRepository) Delete(eventID string) error {
	r.log.Debug("deleting voting configuration", "event_id", eventID)

	if eventID == "" {
		r.log.Error("event ID cannot be empty")
		return errors.New("event ID cannot be empty")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	// Check if configuration exists before deletion
	var config vote.VotingConfiguration
	if err := r.db.Where("event_id = ?", eventUUID).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("attempted to delete non-existent voting configuration", "event_id", eventID)
			return errors.New("voting configuration not found")
		}
		r.log.Error("failed to check configuration existence for deletion", "event_id", eventID, "error", err)
		return fmt.Errorf("failed to check configuration existence: %w", err)
	}

	if err := r.db.Where("event_id = ?", eventUUID).Delete(&vote.VotingConfiguration{}).Error; err != nil {
		r.log.Error("failed to delete voting configuration", "error", err, "event_id", eventID)
		return fmt.Errorf("failed to delete voting configuration: %w", err)
	}

	r.log.Info("voting configuration deleted successfully", "event_id", eventID, "config_id", config.ID)
	return nil
}

// ValidateConfiguration validates a voting configuration
func (r *PostgresVotingConfigurationRepository) ValidateConfiguration(config *vote.VotingConfiguration) error {
	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	r.log.Debug("validating voting configuration", "config_id", config.ID, "event_id", config.EventID)

	if config.EventID == uuid.Nil {
		return errors.New("event ID is required")
	}

	if config.AttachmentsPerEvaluator <= 0 {
		return fmt.Errorf("attachments per evaluator must be positive, got %d", config.AttachmentsPerEvaluator)
	}

	if config.MinEvaluationsPerFile <= 0 {
		return fmt.Errorf("minimum evaluations per file must be positive, got %d", config.MinEvaluationsPerFile)
	}

	if config.AttachmentsPerEvaluator > 50 {
		return fmt.Errorf("attachments per evaluator cannot exceed 50, got %d", config.AttachmentsPerEvaluator)
	}

	if config.MinEvaluationsPerFile > 20 {
		return fmt.Errorf("minimum evaluations per file cannot exceed 20, got %d", config.MinEvaluationsPerFile)
	}

	// Validate quality thresholds
	if config.QualityGoodThreshold < 0.0 || config.QualityGoodThreshold > 1.0 {
		return fmt.Errorf("quality good threshold must be between 0.0 and 1.0, got %.2f", config.QualityGoodThreshold)
	}

	if config.QualityBadThreshold < 0.0 || config.QualityBadThreshold > 1.0 {
		return fmt.Errorf("quality bad threshold must be between 0.0 and 1.0, got %.2f", config.QualityBadThreshold)
	}

	if config.QualityGoodThreshold <= config.QualityBadThreshold {
		return fmt.Errorf("quality good threshold (%.2f) must be higher than quality bad threshold (%.2f)", 
			config.QualityGoodThreshold, config.QualityBadThreshold)
	}

	// Validate adjustment magnitude
	if config.AdjustmentMagnitude < 1 || config.AdjustmentMagnitude > 10 {
		return fmt.Errorf("adjustment magnitude must be between 1 and 10, got %d", config.AdjustmentMagnitude)
	}

	r.log.Debug("voting configuration validation successful", "config_id", config.ID)
	return nil
}

// GetConfigurationSummary returns a summary of the configuration with calculated metrics
func (r *PostgresVotingConfigurationRepository) GetConfigurationSummary(eventID string) (map[string]interface{}, error) {
	r.log.Debug("getting configuration summary", "event_id", eventID)

	config, err := r.GetByEventID(eventID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"id":                       config.ID.String(),
		"event_id":                 config.EventID.String(),
		"attachments_per_evaluator": config.AttachmentsPerEvaluator,
		"quality_good_threshold":   config.QualityGoodThreshold,
		"quality_bad_threshold":    config.QualityBadThreshold,
		"adjustment_magnitude":     config.AdjustmentMagnitude,
		"min_evaluations_per_file": config.MinEvaluationsPerFile,
		"created_at":               config.CreatedAt,
		"updated_at":               config.UpdatedAt,
	}

	// Calculate additional metrics if possible
	var participantCount, attachmentCount int64

	// Count participants
	if err := r.db.Table("event_participants").Where("event_id = ?", config.EventID).Count(&participantCount).Error; err == nil {
		summary["total_participants"] = participantCount
	}

	// Count attachments
	if err := r.db.Table("attachments").Where("event_id = ?", config.EventID).Count(&attachmentCount).Error; err == nil {
		summary["total_attachments"] = attachmentCount
	}

	// Calculate metrics if we have the data
	if participantCount > 0 && attachmentCount > 0 {
		maxPossibleEvaluations := int64(config.AttachmentsPerEvaluator) * participantCount
		minRequiredEvaluations := int64(config.MinEvaluationsPerFile) * attachmentCount
		
		summary["max_possible_evaluations"] = maxPossibleEvaluations
		summary["min_required_evaluations"] = minRequiredEvaluations
		summary["feasible"] = maxPossibleEvaluations >= minRequiredEvaluations
		
		if attachmentCount > 0 {
			summary["avg_evaluations_per_attachment"] = float64(maxPossibleEvaluations) / float64(attachmentCount)
		}
	}

	r.log.Debug("configuration summary retrieved successfully", "event_id", eventID, "config_id", config.ID)
	return summary, nil
}

// IsConfigurationOptimal checks if the configuration follows mathematical best practices
func (r *PostgresVotingConfigurationRepository) IsConfigurationOptimal(eventID string) (bool, []string, error) {
	r.log.Debug("checking configuration optimality", "event_id", eventID)

	config, err := r.GetByEventID(eventID)
	if err != nil {
		return false, nil, err
	}

	var warnings []string
	optimal := true

	// Count attachments for mathematical validation
	var attachmentCount int64
	if err := r.db.Table("attachments").Where("event_id = ?", config.EventID).Count(&attachmentCount).Error; err != nil {
		r.log.Warn("failed to count attachments for optimization check", "event_id", eventID, "error", err)
		warnings = append(warnings, "Could not verify attachment count for optimization checks")
	} else if attachmentCount > 0 {
		// Check mathematical recommendations
		// m ≥ 2*log₂(k) for convergence
		recommendedM := int(math.Ceil(2 * math.Log2(float64(attachmentCount))))
		
		if config.AttachmentsPerEvaluator < recommendedM {
			optimal = false
			warnings = append(warnings, fmt.Sprintf("Attachments per evaluator (%d) is below recommended minimum (%d) for optimal convergence", 
				config.AttachmentsPerEvaluator, recommendedM))
		}
	}

	// Check quality thresholds
	if config.QualityGoodThreshold - config.QualityBadThreshold < 0.2 {
		optimal = false
		warnings = append(warnings, "Quality threshold gap is too small (< 0.2), may affect result stability")
	}

	if config.QualityGoodThreshold < 0.6 {
		warnings = append(warnings, "Quality good threshold is relatively low, consider increasing for better results")
	}

	if config.MinEvaluationsPerFile < 3 {
		optimal = false
		warnings = append(warnings, "Minimum evaluations per file is below recommended minimum (3)")
	}

	r.log.Debug("configuration optimality check completed", "event_id", eventID, "optimal", optimal, "warnings_count", len(warnings))
	return optimal, warnings, nil
}

// GetConfigurationHistory returns configuration changes over time (if audit logging is implemented)
func (r *PostgresVotingConfigurationRepository) GetConfigurationHistory(eventID string) ([]map[string]interface{}, error) {
	r.log.Debug("getting configuration history", "event_id", eventID)

	// For now, just return the current configuration
	// In the future, this could be extended with audit logging
	config, err := r.GetByEventID(eventID)
	if err != nil {
		return nil, err
	}

	history := []map[string]interface{}{
		{
			"timestamp":                config.UpdatedAt,
			"action":                   "current",
			"attachments_per_evaluator": config.AttachmentsPerEvaluator,
			"quality_good_threshold":   config.QualityGoodThreshold,
			"quality_bad_threshold":    config.QualityBadThreshold,
			"adjustment_magnitude":     config.AdjustmentMagnitude,
			"min_evaluations_per_file": config.MinEvaluationsPerFile,
		},
	}

	r.log.Debug("configuration history retrieved", "event_id", eventID, "entries", len(history))
	return history, nil
}
