package postgres

import (
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresVotingResultsRepository implements VotingResultsRepository using GORM
type PostgresVotingResultsRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresVotingResultsRepository creates a new PostgreSQL voting results repository
func NewPostgresVotingResultsRepository(db *gorm.DB) *PostgresVotingResultsRepository {
	return &PostgresVotingResultsRepository{
		db:  db,
		log: logger.Repository("voting_results"),
	}
}

func (r *PostgresVotingResultsRepository) Create(results *vote.VotingResults) error {
	r.log.Debug("creating new voting results", "results_id", results.ID, "event_id", results.EventID)

	if err := r.db.Create(results).Error; err != nil {
		r.log.Error("failed to create voting results", "error", err, "results_id", results.ID)
		return err
	}

	r.log.Info("voting results created successfully", "results_id", results.ID, "event_id", results.EventID)
	return nil
}

func (r *PostgresVotingResultsRepository) GetByEventID(eventID string) (*vote.VotingResults, error) {
	r.log.Debug("retrieving voting results by event ID", "event_id", eventID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, errors.New("invalid event ID format")
	}

	var results vote.VotingResults
	if err := r.db.Preload("Event").Where("event_id = ?", eventUUID).First(&results).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("voting results not found", "event_id", eventID)
			return nil, errors.New("voting results not found")
		}
		r.log.Error("failed to retrieve voting results", "event_id", eventID, "error", err)
		return nil, err
	}

	r.log.Debug("voting results retrieved successfully", "event_id", eventID, "results_id", results.ID)
	return &results, nil
}

// GetByID retrieves voting results by their unique ID
func (r *PostgresVotingResultsRepository) GetByID(id string) (*vote.VotingResults, error) {
	r.log.Debug("retrieving voting results by ID", "results_id", id)
	resultsID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid results ID format", "results_id", id, "error", err)
		return nil, errors.New("invalid results ID format")
	}

	var results vote.VotingResults
	if err := r.db.Preload("Event").Where("id = ?", resultsID).First(&results).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("voting results not found", "results_id", id)
			return nil, errors.New("voting results not found")
		}
		r.log.Error("failed to retrieve voting results", "results_id", id, "error", err)
		return nil, err
	}

	r.log.Debug("voting results retrieved successfully", "results_id", id, "event_id", results.EventID)
	return &results, nil
}

// CalculateResults computes and stores the voting results for an event
func (r *PostgresVotingResultsRepository) CalculateResults(eventID string) (*vote.VotingResults, error) {
	r.log.Debug("calculating voting results", "event_id", eventID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, errors.New("invalid event ID format")
	}

	// Placeholder: Aggregate votes and calculate rankings
	var attachments []vote.AttachmentResult
	// TODO: Implement aggregation logic for global and adjusted rankings

	results := &vote.VotingResults{
		ID:                      uuid.New(),
		EventID:                 eventUUID,
		GlobalRanking:           attachments,
		ParticipantQualities:    make(map[string]float64),
		AdjustedRanking:         attachments,
		TotalParticipants:       0, // TODO: Query participant count
		AttachmentsPerEvaluator: 0, // TODO: Query config
		CalculatedAt:            r.db.NowFunc(),
	}

	if err := r.Create(results); err != nil {
		r.log.Error("failed to store calculated voting results", "event_id", eventID, "error", err)
		return nil, err
	}

	r.log.Info("voting results calculated and stored", "event_id", eventID, "results_id", results.ID)
	return results, nil
}

// GetRankingByEvent returns the ranking of attachments for an event
func (r *PostgresVotingResultsRepository) GetRankingByEvent(eventID string) ([]vote.AttachmentResult, error) {
	r.log.Debug("retrieving ranking by event", "event_id", eventID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return nil, errors.New("invalid event ID format")
	}

	var results vote.VotingResults
	if err := r.db.Where("event_id = ?", eventUUID).First(&results).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("voting results not found for ranking", "event_id", eventID)
			return nil, errors.New("voting results not found")
		}
		r.log.Error("failed to retrieve voting results for ranking", "event_id", eventID, "error", err)
		return nil, err
	}

	r.log.Debug("ranking retrieved successfully", "event_id", eventID, "count", len(results.GlobalRanking))
	return results.GlobalRanking, nil
}

func (r *PostgresVotingResultsRepository) Update(results *vote.VotingResults) error {
	r.log.Debug("updating voting results", "results_id", results.ID)

	if err := r.db.Save(results).Error; err != nil {
		r.log.Error("failed to update voting results", "error", err, "results_id", results.ID)
		return err
	}

	r.log.Info("voting results updated successfully", "results_id", results.ID)
	return nil
}

func (r *PostgresVotingResultsRepository) Delete(eventID string) error {
	r.log.Debug("deleting voting results", "event_id", eventID)

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		r.log.Error("invalid event ID format", "event_id", eventID, "error", err)
		return errors.New("invalid event ID format")
	}

	if err := r.db.Where("event_id = ?", eventUUID).Delete(&vote.VotingResults{}).Error; err != nil {
		r.log.Error("failed to delete voting results", "error", err, "event_id", eventID)
		return err
	}

	r.log.Info("voting results deleted successfully", "event_id", eventID)
	return nil
}
