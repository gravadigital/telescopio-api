package vote

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/domain/common"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Vote represents a ranking vote for an attachment by a participant
type Vote struct {
	ID                    uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EventID               uuid.UUID `json:"event_id" gorm:"type:uuid;not null"`
	AssignmentID          uuid.UUID `json:"assignment_id" gorm:"type:uuid;not null"`
	VoterID               uuid.UUID `json:"voter_id" gorm:"type:uuid;not null"`
	AttachmentID          uuid.UUID `json:"attachment_id" gorm:"type:uuid;not null"`
	RankPosition          int       `json:"rank_position" gorm:"not null"` // 1 = best, higher numbers = worse
	Score                 *float64  `json:"score" gorm:"type:decimal(10,4)"`
	Confidence            *float64  `json:"confidence" gorm:"type:decimal(3,2)"`
	EvaluationTimeSeconds *int      `json:"evaluation_time_seconds"`
	Notes                 string    `json:"notes" gorm:"type:text"`
	IsQualityVote         *bool     `json:"is_quality_vote"`
	VotedAt               time.Time `json:"voted_at" gorm:"autoCreateTime"`

	// Relations - using shared types to avoid circular imports
	Event      common.SharedEvent      `json:"event,omitempty" gorm:"foreignKey:EventID"`
	Assignment Assignment              `json:"assignment,omitempty" gorm:"foreignKey:AssignmentID"`
	Voter      common.SharedUser       `json:"voter,omitempty" gorm:"foreignKey:VoterID"`
	Attachment common.SharedAttachment `json:"attachment,omitempty" gorm:"foreignKey:AttachmentID"`
}

// Assignment represents the distributed assignment of attachments to evaluators
type Assignment struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EventID             uuid.UUID      `json:"event_id" gorm:"type:uuid;not null"`
	ParticipantID       uuid.UUID      `json:"participant_id" gorm:"type:uuid;not null"`
	AttachmentIDs       pq.StringArray `json:"attachment_ids" gorm:"type:uuid[]"` // Array of UUIDs
	AssignmentRound     int            `json:"assignment_round" gorm:"default:1"`
	IsCompleted         bool           `json:"is_completed" gorm:"default:false"`
	CompletedAt         *time.Time     `json:"completed_at"`
	QualityScore        *float64       `json:"quality_score" gorm:"type:decimal(5,4)"` // Q_i score
	ExpertiseMatchScore *float64       `json:"expertise_match_score" gorm:"type:decimal(3,2)"`
	ConflictOfInterest  bool           `json:"conflict_of_interest" gorm:"default:false"`
	CreatedAt           time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations - using shared types to avoid circular imports
	Event       common.SharedEvent `json:"event,omitempty" gorm:"foreignKey:EventID"`
	Participant common.SharedUser  `json:"participant,omitempty" gorm:"foreignKey:ParticipantID"`
	Votes       []Vote             `json:"votes,omitempty" gorm:"foreignKey:AssignmentID"`
}

// VotingResults represents the calculated results
type VotingResults struct {
	ID                      uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EventID                 uuid.UUID          `json:"event_id" gorm:"type:uuid;not null;uniqueIndex"`
	GlobalRanking           []AttachmentResult `json:"global_ranking" gorm:"type:jsonb"`
	ParticipantQualities    map[string]float64 `json:"participant_qualities" gorm:"type:jsonb"`
	AdjustedRanking         []AttachmentResult `json:"adjusted_ranking" gorm:"type:jsonb"`
	TotalParticipants       int                `json:"total_participants"`
	AttachmentsPerEvaluator int                `json:"attachments_per_evaluator"` // m parameter
	CalculatedAt            time.Time          `json:"calculated_at" gorm:"autoCreateTime"`
	UpdatedAt               time.Time          `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations - using shared types to avoid circular imports
	Event common.SharedEvent `json:"event,omitempty" gorm:"foreignKey:EventID"`
}

// VotingConfiguration represents the mathematical parameters for the voting system
type VotingConfiguration struct {
	ID                      uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	EventID                 uuid.UUID `json:"event_id" gorm:"type:uuid;not null;uniqueIndex"`
	AttachmentsPerEvaluator int       `json:"attachments_per_evaluator"` // m parameter
	QualityGoodThreshold    float64   `json:"quality_good_threshold"`    // Q_good
	QualityBadThreshold     float64   `json:"quality_bad_threshold"`     // Q_bad
	AdjustmentMagnitude     int       `json:"adjustment_magnitude"`      // n parameter
	MinEvaluationsPerFile   int       `json:"min_evaluations_per_file"`
	CreatedAt               time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt               time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations - using shared types to avoid circular imports
	Event common.SharedEvent `json:"event,omitempty" gorm:"foreignKey:EventID"`
}

// AttachmentResult represents the MBC score and ranking for an attachment
type AttachmentResult struct {
	AttachmentID  uuid.UUID `json:"attachment_id"`
	Filename      string    `json:"filename"`
	ParticipantID uuid.UUID `json:"participant_id"`
	MBCScore      float64   `json:"mbc_score"`
	GlobalRank    int       `json:"global_rank"`
	AdjustedRank  int       `json:"adjusted_rank"`
	VoteCount     int       `json:"vote_count"`
	AverageRank   float64   `json:"average_rank"`
}

// TableName overrides the table name
func (Vote) TableName() string {
	return "votes"
}

// TableName overrides the table name
func (Assignment) TableName() string {
	return "assignments"
}

// TableName overrides the table name
func (VotingResults) TableName() string {
	return "voting_results"
}

// TableName overrides the table name
func (VotingConfiguration) TableName() string {
	return "voting_configurations"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (v *Vote) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// BeforeCreate will set a UUID rather than numeric ID.
func (a *Assignment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// BeforeCreate will set a UUID rather than numeric ID.
func (vr *VotingResults) BeforeCreate(tx *gorm.DB) error {
	if vr.ID == uuid.Nil {
		vr.ID = uuid.New()
	}
	return nil
}

// BeforeCreate will set a UUID rather than numeric ID.
func (vc *VotingConfiguration) BeforeCreate(tx *gorm.DB) error {
	if vc.ID == uuid.Nil {
		vc.ID = uuid.New()
	}
	return nil
}

func NewVote(eventID, voterID, attachmentID uuid.UUID, rank int) *Vote {
	return &Vote{
		ID:           uuid.New(),
		EventID:      eventID,
		VoterID:      voterID,
		AttachmentID: attachmentID,
		RankPosition: rank,
		VotedAt:      time.Now(),
	}
}

// Validate checks if the vote data is valid
func (v *Vote) Validate() error {
	if v.EventID == uuid.Nil {
		return fmt.Errorf("event_id is required")
	}
	if v.VoterID == uuid.Nil {
		return fmt.Errorf("voter_id is required")
	}
	if v.AttachmentID == uuid.Nil {
		return fmt.Errorf("attachment_id is required")
	}
	if v.RankPosition <= 0 {
		return fmt.Errorf("rank_position must be positive")
	}
	// TODO: Add validation for rank_position <= max_assignments_per_evaluator
	return nil
}

func NewAssignment(eventID, participantID uuid.UUID, attachmentIDs []uuid.UUID) *Assignment {
	stringIDs := make(pq.StringArray, len(attachmentIDs))
	for i, id := range attachmentIDs {
		stringIDs[i] = id.String()
	}

	return &Assignment{
		ID:            uuid.New(),
		EventID:       eventID,
		ParticipantID: participantID,
		AttachmentIDs: stringIDs,
		IsCompleted:   false,
		CreatedAt:     time.Now(),
	}
}

// GetAttachmentUUIDs converts pq.StringArray back to []uuid.UUID
func (a *Assignment) GetAttachmentUUIDs() []uuid.UUID {
	uuids := make([]uuid.UUID, len(a.AttachmentIDs))
	for i, idStr := range a.AttachmentIDs {
		if uuid, err := uuid.Parse(idStr); err == nil {
			uuids[i] = uuid
		}
	}
	return uuids
}

// MarkCompleted marks the assignment as completed
func (a *Assignment) MarkCompleted() {
	a.IsCompleted = true
	now := time.Now()
	a.CompletedAt = &now
	a.UpdatedAt = now
}

// Validate checks if the assignment data is valid
func (a *Assignment) Validate() error {
	if a.EventID == uuid.Nil {
		return fmt.Errorf("event_id is required")
	}
	if a.ParticipantID == uuid.Nil {
		return fmt.Errorf("participant_id is required")
	}
	if len(a.AttachmentIDs) == 0 {
		return fmt.Errorf("at least one attachment must be assigned")
	}
	// TODO: Add validation for maximum attachments per evaluator
	return nil
}

// GetProgress returns the completion progress of the assignment
func (a *Assignment) GetProgress() float64 {
	if len(a.AttachmentIDs) == 0 {
		return 0.0
	}
	if a.IsCompleted {
		return 1.0
	}
	// TODO: Calculate based on actual votes submitted vs total assignments
	return 0.0
}

func NewVotingConfiguration(eventID uuid.UUID, m int) *VotingConfiguration {
	return &VotingConfiguration{
		ID:                      uuid.New(),
		EventID:                 eventID,
		AttachmentsPerEvaluator: m,
		QualityGoodThreshold:    0.6,
		QualityBadThreshold:     0.3,
		AdjustmentMagnitude:     3,
		MinEvaluationsPerFile:   3,
		CreatedAt:               time.Now(),
	}
}

// Validate checks if the voting configuration is mathematically valid
func (vc *VotingConfiguration) Validate() error {
	if vc.EventID == uuid.Nil {
		return fmt.Errorf("event_id is required")
	}
	if vc.AttachmentsPerEvaluator <= 0 {
		return fmt.Errorf("attachments_per_evaluator must be positive")
	}
	if vc.QualityGoodThreshold <= vc.QualityBadThreshold {
		return fmt.Errorf("quality_good_threshold must be higher than quality_bad_threshold")
	}
	if vc.QualityGoodThreshold > 1.0 || vc.QualityBadThreshold < 0.0 {
		return fmt.Errorf("quality thresholds must be in [0, 1] range")
	}
	if vc.AdjustmentMagnitude < 0 {
		return fmt.Errorf("adjustment_magnitude must be non-negative")
	}
	if vc.MinEvaluationsPerFile <= 0 {
		return fmt.Errorf("min_evaluations_per_file must be positive")
	}
	return nil
}

// IsOptimal checks if the configuration follows mathematical recommendations
func (vc *VotingConfiguration) IsOptimal(totalAttachments int) bool {
	// TODO: Implement optimality checks based on mathematical constraints
	// m ≥ 2*log₂(k) for convergence
	// Sufficient evaluations per file
	// Balanced thresholds
	return true // Placeholder
}
