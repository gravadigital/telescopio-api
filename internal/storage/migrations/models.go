package migrations

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Custom types for GORM

type EventStage string

const (
	EventStageCreation         EventStage = "creation"
	EventStageRegistration     EventStage = "registration"
	EventStageAttachmentUpload EventStage = "attachment_upload"
	EventStageVoting           EventStage = "voting"
	EventStageResults          EventStage = "results"
)

func (es *EventStage) Scan(value any) error {
	if value == nil {
		*es = EventStageCreation
		return nil
	}
	if str, ok := value.(string); ok {
		*es = EventStage(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into EventStage", value)
}

func (es EventStage) Value() (driver.Value, error) {
	return string(es), nil
}

type UserRole string

const (
	UserRoleAdmin       UserRole = "admin"
	UserRoleParticipant UserRole = "participant"
)

func (ur *UserRole) Scan(value any) error {
	if value == nil {
		*ur = UserRoleParticipant
		return nil
	}
	if str, ok := value.(string); ok {
		*ur = UserRole(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into UserRole", value)
}

func (ur UserRole) Value() (driver.Value, error) {
	return string(ur), nil
}

// Core Models for the Voting System

// User represents participants in the voting system
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Lastname  string    `json:"lastname"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Role      UserRole  `gorm:"type:user_role;not null;default:'participant'" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	AuthoredEvents []Event            `gorm:"foreignKey:AuthorID" json:"authored_events,omitempty"`
	Participations []EventParticipant `gorm:"foreignKey:UserID" json:"participations,omitempty"`
	Attachments    []Attachment       `gorm:"foreignKey:ParticipantID" json:"attachments,omitempty"`
	Assignments    []Assignment       `gorm:"foreignKey:ParticipantID" json:"assignments,omitempty"`
	Votes          []Vote             `gorm:"foreignKey:VoterID" json:"votes,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// Event represents voting events for telescope time allocation
type Event struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	Description string     `gorm:"not null" json:"description"`
	AuthorID    uuid.UUID  `gorm:"type:uuid;not null" json:"author_id"`
	StartDate   time.Time  `gorm:"not null" json:"start_date"`
	EndDate     time.Time  `gorm:"not null" json:"end_date"`
	Stage       EventStage `gorm:"type:event_stage;not null;default:'creation'" json:"stage"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Author        User                 `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Participants  []EventParticipant   `gorm:"foreignKey:EventID" json:"participants,omitempty"`
	Attachments   []Attachment         `gorm:"foreignKey:EventID" json:"attachments,omitempty"`
	Configuration *VotingConfiguration `gorm:"foreignKey:EventID" json:"configuration,omitempty"`
	Assignments   []Assignment         `gorm:"foreignKey:EventID" json:"assignments,omitempty"`
	Votes         []Vote               `gorm:"foreignKey:EventID" json:"votes,omitempty"`
	Results       *VotingResult        `gorm:"foreignKey:EventID" json:"results,omitempty"`
}

func (Event) TableName() string {
	return "events"
}

// EventParticipant represents the many-to-many relationship between events and users
type EventParticipant struct {
	EventID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"event_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`

	// Relations
	Event Event `gorm:"foreignKey:EventID" json:"event,omitempty"`
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (EventParticipant) TableName() string {
	return "event_participants"
}

// Attachment represents proposals/files submitted for evaluation (set F)
type Attachment struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EventID       uuid.UUID `gorm:"type:uuid;not null" json:"event_id"`
	ParticipantID uuid.UUID `gorm:"type:uuid;not null" json:"participant_id"`
	Filename      string    `gorm:"not null" json:"filename"`
	OriginalName  string    `gorm:"not null" json:"original_name"`
	FilePath      string    `gorm:"not null" json:"file_path"`
	FileSize      int64     `gorm:"not null" json:"file_size"`
	MimeType      string    `gorm:"size:100;not null" json:"mime_type"`
	VoteCount     int       `gorm:"default:0" json:"vote_count"`
	UploadedAt    time.Time `gorm:"autoCreateTime" json:"uploaded_at"`

	// Relations
	Event       Event  `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Participant User   `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	Votes       []Vote `gorm:"foreignKey:AttachmentID" json:"votes,omitempty"`
}

func (Attachment) TableName() string {
	return "attachments"
}

// VotingConfiguration stores mathematical parameters
type VotingConfiguration struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EventID                 uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"event_id"`
	AttachmentsPerEvaluator int       `gorm:"not null" json:"attachments_per_evaluator"` // m parameter
	MinEvaluationsPerFile   int       `gorm:"not null;default:3" json:"min_evaluations_per_file"`
	QualityGoodThreshold    float64   `gorm:"type:decimal(3,2);not null;default:0.65" json:"quality_good_threshold"` // Q_good
	QualityBadThreshold     float64   `gorm:"type:decimal(3,2);not null;default:0.35" json:"quality_bad_threshold"`  // Q_bad
	AdjustmentMagnitude     int       `gorm:"not null;default:3" json:"adjustment_magnitude"`                        // n parameter
	UseExpertiseMatching    bool      `gorm:"default:false" json:"use_expertise_matching"`
	EnableCOIDetection      bool      `gorm:"column:enable_co_idetection;default:true" json:"enable_coi_detection"`
	RandomizationSeed       *int      `json:"randomization_seed"`
	AssignmentAlgorithm     string    `gorm:"size:50;default:'random_balanced'" json:"assignment_algorithm"`
	ScoringAlgorithm        string    `gorm:"size:50;default:'modified_borda_count'" json:"scoring_algorithm"`
	CreatedAt               time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Event Event `gorm:"foreignKey:EventID" json:"event,omitempty"`
}

func (VotingConfiguration) TableName() string {
	return "voting_configurations"
}

// Assignment represents the distributed assignment function A: P → 2^F
type Assignment struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EventID             uuid.UUID      `gorm:"type:uuid;not null" json:"event_id"`
	ParticipantID       uuid.UUID      `gorm:"type:uuid;not null" json:"participant_id"`
	AttachmentIDs       pq.StringArray `gorm:"type:uuid[]" json:"attachment_ids"` // Array of UUIDs
	AssignmentRound     int            `gorm:"default:1" json:"assignment_round"`
	IsCompleted         bool           `gorm:"default:false" json:"is_completed"`
	CompletedAt         *time.Time     `json:"completed_at"`
	QualityScore        *float64       `gorm:"type:decimal(5,4)" json:"quality_score"` // Q_i score
	ExpertiseMatchScore *float64       `gorm:"type:decimal(3,2)" json:"expertise_match_score"`
	ConflictOfInterest  bool           `gorm:"default:false" json:"conflict_of_interest"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Event       Event  `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Participant User   `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	Votes       []Vote `gorm:"foreignKey:AssignmentID" json:"votes,omitempty"`
}

func (Assignment) TableName() string {
	return "assignments"
}

// Vote represents individual rankings R_i: A(p_i) → {1, 2, ..., m}
type Vote struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EventID               uuid.UUID `gorm:"type:uuid;not null" json:"event_id"`
	AssignmentID          uuid.UUID `gorm:"type:uuid;not null" json:"assignment_id"`
	VoterID               uuid.UUID `gorm:"type:uuid;not null" json:"voter_id"`
	AttachmentID          uuid.UUID `gorm:"type:uuid;not null" json:"attachment_id"`
	RankPosition          int       `gorm:"not null" json:"rank_position"` // 1 = best
	Score                 *float64  `gorm:"type:decimal(10,4)" json:"score"`
	Confidence            *float64  `gorm:"type:decimal(3,2)" json:"confidence"`
	EvaluationTimeSeconds *int      `json:"evaluation_time_seconds"`
	Notes                 string    `gorm:"type:text" json:"notes"`
	IsQualityVote         *bool     `json:"is_quality_vote"`
	VotedAt               time.Time `gorm:"autoCreateTime" json:"voted_at"`

	// Relations
	Event      Event      `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Assignment Assignment `gorm:"foreignKey:AssignmentID" json:"assignment,omitempty"`
	Voter      User       `gorm:"foreignKey:VoterID" json:"voter,omitempty"`
	Attachment Attachment `gorm:"foreignKey:AttachmentID" json:"attachment,omitempty"`
}

func (Vote) TableName() string {
	return "votes"
}

// VotingResult stores final MBC calculations and global ranking G
type VotingResult struct {
	ID                        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EventID                   uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"event_id"`
	GlobalRanking             string    `gorm:"type:jsonb;not null" json:"global_ranking"`        // JSONB array
	ParticipantQualities      string    `gorm:"type:jsonb;not null" json:"participant_qualities"` // JSONB object
	AdjustedRanking           string    `gorm:"type:jsonb;not null" json:"adjusted_ranking"`      // JSONB array
	StatisticalMetrics        string    `gorm:"type:jsonb" json:"statistical_metrics"`            // JSONB object
	TotalParticipants         int       `gorm:"not null" json:"total_participants"`
	TotalVotes                int       `gorm:"not null" json:"total_votes"`
	AttachmentsPerEvaluator   int       `gorm:"not null" json:"attachments_per_evaluator"`
	AlgorithmUsed             string    `gorm:"size:100;not null;default:'modified_borda_count'" json:"algorithm_used"`
	QualityAdjustmentsApplied bool      `gorm:"default:true" json:"quality_adjustments_applied"`
	OverallQualityScore       *float64  `gorm:"type:decimal(5,4)" json:"overall_quality_score"`
	GoodEvaluatorCount        int       `gorm:"default:0" json:"good_evaluator_count"`
	BadEvaluatorCount         int       `gorm:"default:0" json:"bad_evaluator_count"`
	ConsensusStrength         *float64  `gorm:"type:decimal(3,2)" json:"consensus_strength"`
	CalculatedAt              time.Time `gorm:"autoCreateTime" json:"calculated_at"`
	UpdatedAt                 time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relations
	Event Event `gorm:"foreignKey:EventID" json:"event,omitempty"`
}

func (VotingResult) TableName() string {
	return "voting_results"
}

// AllModels returns a slice of all models for migration
func AllModels() []any {
	return []any{
		&User{},
		&Event{},
		&EventParticipant{},
		&Attachment{},
		&VotingConfiguration{},
		&Assignment{},
		&Vote{},
		&VotingResult{},
	}
}
