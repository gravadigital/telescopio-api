package postgres

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// PostgresVoteDraftRepository implements VoteDraftRepository using GORM
type PostgresVoteDraftRepository struct {
	db  *gorm.DB
	log *log.Logger
}

// NewPostgresVoteDraftRepository creates a new PostgreSQL vote draft repository
func NewPostgresVoteDraftRepository(db *gorm.DB) *PostgresVoteDraftRepository {
	return &PostgresVoteDraftRepository{
		db:  db,
		log: logger.Repository("vote_draft"),
	}
}

// Upsert saves or replaces the draft for a given assignment+participant.
// If a draft already exists, only rankings and updated_at are overwritten.
func (r *PostgresVoteDraftRepository) Upsert(draft *vote.VoteDraft) error {
	r.log.Debug("upserting vote draft",
		"assignment_id", draft.AssignmentID,
		"participant_id", draft.ParticipantID,
		"rankings_count", len(draft.Rankings),
	)

	err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "assignment_id"}, {Name: "participant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"rankings", "updated_at"}),
	}).Create(draft).Error

	if err != nil {
		r.log.Error("failed to upsert vote draft",
			"assignment_id", draft.AssignmentID,
			"participant_id", draft.ParticipantID,
			"error", err,
		)
		return fmt.Errorf("failed to upsert vote draft: %w", err)
	}

	r.log.Info("vote draft upserted successfully",
		"assignment_id", draft.AssignmentID,
		"participant_id", draft.ParticipantID,
	)
	return nil
}

// GetByAssignmentAndParticipant retrieves the draft for a specific assignment and participant.
// Returns gorm.ErrRecordNotFound (wrapped) if no draft exists yet.
func (r *PostgresVoteDraftRepository) GetByAssignmentAndParticipant(assignmentID, participantID uuid.UUID) (*vote.VoteDraft, error) {
	r.log.Debug("retrieving vote draft",
		"assignment_id", assignmentID,
		"participant_id", participantID,
	)

	var draft vote.VoteDraft
	err := r.db.
		Where("assignment_id = ? AND participant_id = ?", assignmentID, participantID).
		First(&draft).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("vote draft not found",
				"assignment_id", assignmentID,
				"participant_id", participantID,
			)
			return nil, gorm.ErrRecordNotFound
		}
		r.log.Error("failed to retrieve vote draft",
			"assignment_id", assignmentID,
			"participant_id", participantID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to retrieve vote draft: %w", err)
	}

	r.log.Debug("vote draft retrieved successfully",
		"assignment_id", assignmentID,
		"participant_id", participantID,
		"rankings_count", len(draft.Rankings),
	)
	return &draft, nil
}
