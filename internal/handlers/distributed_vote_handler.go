package handlers

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type DistributedVoteHandler struct {
	voteRepo       postgres.VoteRepository
	eventRepo      postgres.EventRepository
	attachmentRepo postgres.AttachmentRepository
	userRepo       postgres.UserRepository
	configRepo     postgres.VotingConfigurationRepository
	resultsRepo    postgres.VotingResultsRepository
	votingService  *vote.VotingService
	config         *config.Config
	log            *log.Logger
}

func NewDistributedVoteHandler(
	voteRepo postgres.VoteRepository,
	eventRepo postgres.EventRepository,
	attachmentRepo postgres.AttachmentRepository,
	userRepo postgres.UserRepository,
	configRepo postgres.VotingConfigurationRepository,
	resultsRepo postgres.VotingResultsRepository,
	cfg *config.Config,
) *DistributedVoteHandler {
	// Create adapters to bridge interface differences
	voteAdapter := NewVoteRepositoryAdapter(voteRepo)
	attachmentAdapter := NewAttachmentRepositoryAdapter(attachmentRepo)
	userAdapter := NewUserRepositoryAdapter(userRepo)

	votingService := vote.NewVotingService(voteAdapter, attachmentAdapter, userAdapter)

	return &DistributedVoteHandler{
		voteRepo:       voteRepo,
		eventRepo:      eventRepo,
		attachmentRepo: attachmentRepo,
		userRepo:       userRepo,
		configRepo:     configRepo,
		resultsRepo:    resultsRepo,
		votingService:  votingService,
		config:         cfg,
		log:            logger.Handler("distributed_vote"),
	}
}

// CreateVotingConfiguration handles POST /api/events/{event_id}/voting-config
func (h *DistributedVoteHandler) CreateVotingConfiguration(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("creating voting configuration", "event_id", eventID)

	// Validate required parameters
	if eventID == "" {
		h.log.Warn("missing event_id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
			"code":  "MISSING_EVENT_ID",
		})
		return
	}

	// Validate UUID format
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	// TODO: Add authentication check
	// userID := c.GetString("user_id") // From JWT middleware
	// if userID == "" {
	//     c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required", "code": "UNAUTHORIZED"})
	//     return
	// }

	var req struct {
		AttachmentsPerEvaluator int     `json:"attachments_per_evaluator" binding:"required,min=1,max=50"`
		QualityGoodThreshold    float64 `json:"quality_good_threshold" binding:"min=0,max=1"`
		QualityBadThreshold     float64 `json:"quality_bad_threshold" binding:"min=0,max=1"`
		AdjustmentMagnitude     int     `json:"adjustment_magnitude" binding:"min=1,max=10"`
		MinEvaluationsPerFile   int     `json:"min_evaluations_per_file" binding:"min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid request payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Additional business validation
	if req.QualityGoodThreshold != 0 && req.QualityBadThreshold != 0 {
		if req.QualityGoodThreshold <= req.QualityBadThreshold {
			h.log.Warn("invalid quality thresholds", "good", req.QualityGoodThreshold, "bad", req.QualityBadThreshold)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Quality good threshold must be higher than bad threshold",
				"code":  "INVALID_THRESHOLDS",
			})
			return
		}
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// TODO: Check if user has permission to configure this event
	// user, err := h.userRepo.GetByID(userID)
	// if err != nil || (!user.HasRole(participant.RoleAdmin) && eventObj.AuthorID.String() != userID) {
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Insufficient permissions to configure this event",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	// Only allow configuration in registration stage
	if eventObj.Stage != event.StageRegistration {
		h.log.Warn("voting configuration attempt in wrong stage", "event_id", eventID, "current_stage", eventObj.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Voting configuration can only be set during registration stage",
			"code":          "INVALID_EVENT_STAGE",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Check if configuration already exists
	existingConfig, err := h.configRepo.GetByEventID(eventID)
	if err == nil && existingConfig != nil {
		h.log.Warn("voting configuration already exists", "event_id", eventID, "existing_config_id", existingConfig.ID)
		c.JSON(http.StatusConflict, gin.H{
			"error": "Voting configuration already exists for this event",
			"code":  "CONFIG_EXISTS",
			"existing_config": gin.H{
				"id":         existingConfig.ID.String(),
				"created_at": existingConfig.CreatedAt,
			},
		})
		return
	}

	// Get participants and attachments to validate configuration
	participants, err := h.userRepo.GetEventParticipants(eventID)
	if err != nil {
		h.log.Error("failed to get participants", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get participants",
			"code":  "PARTICIPANTS_ERROR",
		})
		return
	}

	if len(participants) < 2 {
		h.log.Warn("insufficient participants for voting", "event_id", eventID, "participant_count", len(participants))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "At least 2 participants are required for distributed voting",
			"code":          "INSUFFICIENT_PARTICIPANTS",
			"current_count": len(participants),
		})
		return
	}

	attachments, err := h.attachmentRepo.GetByEventID(eventID)
	if err != nil {
		h.log.Error("failed to get attachments", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get attachments",
			"code":  "ATTACHMENTS_ERROR",
		})
		return
	}

	if len(attachments) == 0 {
		h.log.Warn("no attachments for voting configuration", "event_id", eventID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least 1 attachment is required for voting configuration",
			"code":  "NO_ATTACHMENTS",
		})
		return
	}

	// Create voting configuration
	config := &vote.VotingConfiguration{
		ID:                      uuid.New(),
		EventID:                 eventUUID,
		AttachmentsPerEvaluator: req.AttachmentsPerEvaluator,
		QualityGoodThreshold:    req.QualityGoodThreshold,
		QualityBadThreshold:     req.QualityBadThreshold,
		AdjustmentMagnitude:     req.AdjustmentMagnitude,
		MinEvaluationsPerFile:   req.MinEvaluationsPerFile,
		CreatedAt:               time.Now(),
	}

	// Set smart defaults if not provided
	if config.QualityGoodThreshold == 0 {
		config.QualityGoodThreshold = 0.6
	}
	if config.QualityBadThreshold == 0 {
		config.QualityBadThreshold = 0.3
	}
	if config.AdjustmentMagnitude == 0 {
		config.AdjustmentMagnitude = 3
	}
	if config.MinEvaluationsPerFile == 0 {
		config.MinEvaluationsPerFile = 3
	}

	// Validate configuration with current data
	if err := h.votingService.ValidateVotingConfiguration(config, len(attachments), len(participants)); err != nil {
		h.log.Error("voting configuration validation failed", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid voting configuration",
			"code":    "VALIDATION_FAILED",
			"details": err.Error(),
		})
		return
	}

	// Validate mathematical constraints
	maxPossibleAssignments := req.AttachmentsPerEvaluator * len(participants)
	minRequiredAssignments := req.MinEvaluationsPerFile * len(attachments)
	if maxPossibleAssignments < minRequiredAssignments {
		h.log.Warn("mathematical constraint violation",
			"max_possible", maxPossibleAssignments,
			"min_required", minRequiredAssignments,
			"participants", len(participants),
			"attachments", len(attachments))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":                    "Configuration violates mathematical constraints",
			"code":                     "MATH_CONSTRAINT_VIOLATION",
			"details":                  "Not enough evaluation capacity to meet minimum evaluations per file",
			"max_possible_evaluations": maxPossibleAssignments,
			"min_required_evaluations": minRequiredAssignments,
		})
		return
	}

	// Save configuration
	if err := h.configRepo.Create(config); err != nil {
		h.log.Error("failed to save voting configuration", "event_id", eventID, "config_id", config.ID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save voting configuration",
			"code":    "DB_SAVE_ERROR",
			"details": err.Error(),
		})
		return
	}

	h.log.Info("voting configuration created successfully",
		"event_id", eventID,
		"config_id", config.ID,
		"participants", len(participants),
		"attachments", len(attachments))

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"id":                        config.ID.String(),
			"event_id":                  eventID,
			"attachments_per_evaluator": config.AttachmentsPerEvaluator,
			"quality_good_threshold":    config.QualityGoodThreshold,
			"quality_bad_threshold":     config.QualityBadThreshold,
			"adjustment_magnitude":      config.AdjustmentMagnitude,
			"min_evaluations_per_file":  config.MinEvaluationsPerFile,
			"created_at":                config.CreatedAt,
		},
		"message": "Voting configuration created successfully",
		"code":    "CONFIG_CREATED",
		"statistics": gin.H{
			"participants_count": len(participants),
			"attachments_count":  len(attachments),
			"max_evaluations":    maxPossibleAssignments,
			"min_evaluations":    minRequiredAssignments,
		},
	})
}

// GenerateAssignments handles POST /api/events/{event_id}/generate-assignments
func (h *DistributedVoteHandler) GenerateAssignments(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("generating assignments", "event_id", eventID)

	// Validate required parameters
	if eventID == "" {
		h.log.Warn("missing event_id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
			"code":  "MISSING_EVENT_ID",
		})
		return
	}

	// Validate UUID format
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	// TODO: Add authentication and authorization check
	// userID := c.GetString("user_id")
	// user, err := h.userRepo.GetByID(userID)
	// if err != nil || !user.HasRole(participant.RoleAdmin) {
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Insufficient permissions to generate assignments",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	// Check if event exists and is in voting stage
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	if eventObj.Stage != event.StageVoting {
		h.log.Warn("assignment generation attempt in wrong stage", "event_id", eventID, "current_stage", eventObj.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Assignments can only be generated during voting stage",
			"code":          "INVALID_EVENT_STAGE",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Check if assignments already exist
	existingAssignments, err := h.voteRepo.GetAssignmentsByEventID(eventID)
	if err == nil && len(existingAssignments) > 0 {
		h.log.Warn("assignments already exist", "event_id", eventID, "existing_count", len(existingAssignments))
		c.JSON(http.StatusConflict, gin.H{
			"error":                "Assignments already exist for this event",
			"code":                 "ASSIGNMENTS_EXIST",
			"existing_assignments": len(existingAssignments),
		})
		return
	}

	// Get participants and attachments
	participantUsers, err := h.userRepo.GetEventParticipants(eventID)
	if err != nil {
		h.log.Error("failed to get participants", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get participants",
			"code":  "PARTICIPANTS_ERROR",
		})
		return
	}

	if len(participantUsers) < 2 {
		h.log.Warn("insufficient participants for assignment generation", "event_id", eventID, "participant_count", len(participantUsers))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "At least 2 participants are required for distributed voting",
			"code":          "INSUFFICIENT_PARTICIPANTS",
			"current_count": len(participantUsers),
		})
		return
	}

	participants := make([]uuid.UUID, len(participantUsers))
	for i, p := range participantUsers {
		participants[i] = p.ID
	}

	attachments, err := h.attachmentRepo.GetByEventID(eventID)
	if err != nil {
		h.log.Error("failed to get attachments", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get attachments",
			"code":  "ATTACHMENTS_ERROR",
		})
		return
	}

	if len(attachments) == 0 {
		h.log.Warn("no attachments for assignment generation", "event_id", eventID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least 1 attachment is required for assignment generation",
			"code":  "NO_ATTACHMENTS",
		})
		return
	}

	attachmentIDs := make([]uuid.UUID, len(attachments))
	for i, a := range attachments {
		attachmentIDs[i] = a.ID
	}

	// Get voting configuration
	config, err := h.configRepo.GetByEventID(eventID)
	if err != nil {
		h.log.Error("voting configuration not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Voting configuration not found for this event",
			"code":    "CONFIG_NOT_FOUND",
			"details": "Please create a voting configuration before generating assignments",
		})
		return
	}

	// Validate configuration is still valid with current data
	if err := h.votingService.ValidateVotingConfiguration(config, len(attachments), len(participants)); err != nil {
		h.log.Error("voting configuration is no longer valid", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Voting configuration is no longer valid with current data",
			"code":    "CONFIG_INVALID",
			"details": err.Error(),
		})
		return
	}

	// Generate assignments
	h.log.Info("generating assignments",
		"event_id", eventID,
		"participants", len(participants),
		"attachments", len(attachmentIDs))

	assignments, err := h.votingService.GenerateAssignments(eventUUID, participants, attachmentIDs, config)
	if err != nil {
		h.log.Error("failed to generate assignments", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate assignments",
			"code":    "GENERATION_FAILED",
			"details": err.Error(),
		})
		return
	}

	// Save assignments to database
	savedCount := 0
	for _, assignment := range assignments {
		if err := h.voteRepo.CreateAssignment(assignment); err != nil {
			h.log.Error("failed to save assignment",
				"event_id", eventID,
				"assignment_id", assignment.ID,
				"participant_id", assignment.ParticipantID,
				"error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to save assignment",
				"code":    "DB_SAVE_ERROR",
				"details": err.Error(),
			})
			return
		}
		savedCount++
	}

	h.log.Info("assignments generated and saved successfully",
		"event_id", eventID,
		"assignments_count", savedCount,
		"participants", len(participants),
		"attachments", len(attachmentIDs))

	// Calculate assignment statistics
	totalEvaluations := 0
	for _, assignment := range assignments {
		totalEvaluations += len(assignment.GetAttachmentUUIDs())
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"assignments_count":         len(assignments),
			"total_participants":        len(participants),
			"total_attachments":         len(attachmentIDs),
			"total_evaluations":         totalEvaluations,
			"attachments_per_evaluator": config.AttachmentsPerEvaluator,
		},
		"message": "Assignments generated successfully",
		"code":    "ASSIGNMENTS_GENERATED",
		"config": gin.H{
			"id":                        config.ID.String(),
			"attachments_per_evaluator": config.AttachmentsPerEvaluator,
			"min_evaluations_per_file":  config.MinEvaluationsPerFile,
		},
	})
}

// GetParticipantAssignment handles GET /api/events/{event_id}/participants/{participant_id}/assignment
func (h *DistributedVoteHandler) GetParticipantAssignment(c *gin.Context) {
	eventID := c.Param("event_id")
	participantID := c.Param("participant_id")

	if eventID == "" || participantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_id and participant_id are required"})
		return
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Check if event is in voting stage
	if eventObj.Stage != event.StageVoting {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Assignments are only available during voting stage",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Check if participant is registered for this event
	participantEvents, err := h.eventRepo.GetByParticipant(participantID)
	isParticipant := false
	if err == nil {
		for _, evt := range participantEvents {
			if evt.ID.String() == eventID {
				isParticipant = true
				break
			}
		}
	}

	if !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Participant is not registered for this event"})
		return
	}

	// Get assignment for this participant
	assignment, err := h.voteRepo.GetAssignmentByParticipant(eventID, participantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found for this participant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assignment":     assignment,
		"event_name":     eventObj.Name,
		"participant_id": participantID,
	})
}

// SubmitRankingVotes handles POST /api/events/{event_id}/participants/{participant_id}/ranking-votes
func (h *DistributedVoteHandler) SubmitRankingVotes(c *gin.Context) {
	eventID := c.Param("event_id")
	participantID := c.Param("participant_id")

	if eventID == "" || participantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_id and participant_id are required"})
		return
	}

	var req struct {
		AssignmentID string `json:"assignment_id" binding:"required"`
		Rankings     []struct {
			AttachmentID string `json:"attachment_id" binding:"required"`
			Rank         int    `json:"rank" binding:"required,min=1"`
		} `json:"rankings" binding:"required,dive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Check if event exists and is in voting stage
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	if eventObj.Stage != event.StageVoting {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Voting is only allowed during voting stage",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Check if participant is registered
	participantEvents, err := h.eventRepo.GetByParticipant(participantID)
	isParticipant := false
	if err == nil {
		for _, evt := range participantEvents {
			if evt.ID.String() == eventID {
				isParticipant = true
				break
			}
		}
	}

	if !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Participant is not registered for this event"})
		return
	}

	eventUUID := uuid.MustParse(eventID)
	participantUUID := uuid.MustParse(participantID)
	assignmentUUID, err := uuid.Parse(req.AssignmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment_id format"})
		return
	}

	// Verify assignment belongs to participant and event
	assignment, err := h.voteRepo.GetAssignmentByParticipant(eventID, participantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found for this participant"})
		return
	}

	if assignment.ID != assignmentUUID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assignment ID does not match participant's assignment"})
		return
	}

	// Get assigned attachments to validate the vote
	assignedAttachments := assignment.GetAttachmentUUIDs()
	assignedMap := make(map[uuid.UUID]bool)
	for _, attachmentID := range assignedAttachments {
		assignedMap[attachmentID] = true
	}

	// Validate rankings (should be consecutive integers starting from 1)
	rankSet := make(map[int]bool)
	for _, ranking := range req.Rankings {
		if rankSet[ranking.Rank] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate rank found"})
			return
		}
		rankSet[ranking.Rank] = true
	}

	// Check that ranks form a complete sequence
	for i := 1; i <= len(req.Rankings); i++ {
		if !rankSet[i] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Rankings must be consecutive integers starting from 1",
			})
			return
		}
	}

	// Create vote records
	var votes []*vote.Vote
	for _, ranking := range req.Rankings {
		attachmentUUID, err := uuid.Parse(ranking.AttachmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid attachment_id format: " + ranking.AttachmentID,
			})
			return
		}

		// Check if attachment is in participant's assignment
		if !assignedMap[attachmentUUID] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Attachment is not assigned to this participant: " + ranking.AttachmentID,
			})
			return
		}

		// Check if attachment exists and belongs to this event
		attachment, err := h.attachmentRepo.GetByID(ranking.AttachmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Attachment not found: " + ranking.AttachmentID,
			})
			return
		}

		if attachment.EventID != eventUUID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Attachment does not belong to this event: " + ranking.AttachmentID,
			})
			return
		}

		vote := &vote.Vote{
			ID:           uuid.New(),
			EventID:      eventUUID,
			AssignmentID: assignmentUUID,
			VoterID:      participantUUID,
			AttachmentID: attachmentUUID,
			RankPosition: ranking.Rank,
		}
		votes = append(votes, vote)
	}

	// Save votes
	for _, v := range votes {
		if err := h.voteRepo.Create(v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to save vote",
			})
			return
		}
	}

	// Mark assignment as completed if all attachments are ranked
	if len(votes) == len(assignedAttachments) {
		assignment.IsCompleted = true
		now := time.Now()
		assignment.CompletedAt = &now
		if err := h.voteRepo.UpdateAssignment(assignment); err != nil {
			// Log error but don't fail the request - just ignore logging for now
			// TODO: Improve error handling
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Ranking votes submitted successfully",
		"event_id":       eventID,
		"participant_id": participantID,
		"votes_count":    len(votes),
	})
}

// GetDistributedResults handles GET /api/events/{event_id}/distributed-results
func (h *DistributedVoteHandler) GetDistributedResults(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_id is required"})
		return
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id format"})
		return
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Check if event is in results stage
	if eventObj.Stage != event.StageResult {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         "Distributed results are not yet available",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Get voting configuration
	config, err := h.configRepo.GetByEventID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Voting configuration not found",
			"details": "Please create a voting configuration before calculating results",
		})
		return
	}

	// Calculate Modified Borda Count results
	results, err := h.votingService.CalculateModifiedBordaCount(eventUUID, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate results",
			"details": err.Error(),
		})
		return
	}

	// Save results to database
	if err := h.resultsRepo.Create(results); err != nil {
		// If results already exist, update them
		if err := h.resultsRepo.Update(results); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to save results",
				"details": err.Error(),
			})
			return
		}
	}

	// Get additional metrics
	includeMetrics := c.Query("include_metrics") == "true"
	response := gin.H{
		"event_id":                  eventID,
		"event_name":                eventObj.Name,
		"global_ranking":            results.GlobalRanking,
		"adjusted_ranking":          results.AdjustedRanking,
		"total_participants":        results.TotalParticipants,
		"attachments_per_evaluator": results.AttachmentsPerEvaluator,
		"calculated_at":             results.CalculatedAt,
	}

	if includeMetrics {
		response["participant_qualities"] = results.ParticipantQualities
		response["configuration"] = config
	}

	c.JSON(http.StatusOK, response)
}

// GetVotingStatistics handles GET /api/events/{event_id}/voting-statistics
func (h *DistributedVoteHandler) GetVotingStatistics(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_id is required"})
		return
	}

	// Get basic voting statistics
	votes, err := h.voteRepo.GetByEventID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get votes"})
		return
	}

	// Calculate statistics
	totalVotes := len(votes)
	participantVotes := make(map[uuid.UUID]int)
	attachmentVotes := make(map[uuid.UUID]int)

	for _, vote := range votes {
		participantVotes[vote.VoterID]++
		attachmentVotes[vote.AttachmentID]++
	}

	completedParticipants := 0
	for _, count := range participantVotes {
		if count > 0 { // In real implementation, check if assignment is complete
			completedParticipants++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":               eventID,
		"total_votes":            totalVotes,
		"completed_participants": completedParticipants,
		"total_participants":     len(participantVotes),
		"attachments_with_votes": len(attachmentVotes),
		"participation_rate":     float64(completedParticipants) / float64(len(participantVotes)),
	})
}

// GetVotingConfiguration handles GET /api/events/{event_id}/voting-config
func (h *DistributedVoteHandler) GetVotingConfiguration(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("retrieving voting configuration", "event_id", eventID)

	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
			"code":  "MISSING_EVENT_ID",
		})
		return
	}

	// Get voting configuration
	config, err := h.configRepo.GetByEventID(eventID)
	if err != nil {
		h.log.Error("voting configuration not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Voting configuration not found for this event",
			"code":  "CONFIG_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":                        config.ID.String(),
			"event_id":                  config.EventID.String(),
			"attachments_per_evaluator": config.AttachmentsPerEvaluator,
			"quality_good_threshold":    config.QualityGoodThreshold,
			"quality_bad_threshold":     config.QualityBadThreshold,
			"adjustment_magnitude":      config.AdjustmentMagnitude,
			"min_evaluations_per_file":  config.MinEvaluationsPerFile,
			"created_at":                config.CreatedAt,
		},
	})
}

// UpdateVotingConfiguration handles PUT /api/events/{event_id}/voting-config
func (h *DistributedVoteHandler) UpdateVotingConfiguration(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("updating voting configuration", "event_id", eventID)

	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
			"code":  "MISSING_EVENT_ID",
		})
		return
	}

	// Check if event exists and is still configurable
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Only allow updates in registration stage
	if eventObj.Stage != event.StageRegistration {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Voting configuration can only be updated during registration stage",
			"code":          "INVALID_EVENT_STAGE",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Get existing configuration
	config, err := h.configRepo.GetByEventID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Voting configuration not found for this event",
			"code":  "CONFIG_NOT_FOUND",
		})
		return
	}

	var req struct {
		AttachmentsPerEvaluator int     `json:"attachments_per_evaluator" binding:"required,min=1,max=50"`
		QualityGoodThreshold    float64 `json:"quality_good_threshold" binding:"min=0,max=1"`
		QualityBadThreshold     float64 `json:"quality_bad_threshold" binding:"min=0,max=1"`
		AdjustmentMagnitude     int     `json:"adjustment_magnitude" binding:"min=1,max=10"`
		MinEvaluationsPerFile   int     `json:"min_evaluations_per_file" binding:"min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Update configuration
	config.AttachmentsPerEvaluator = req.AttachmentsPerEvaluator
	config.QualityGoodThreshold = req.QualityGoodThreshold
	config.QualityBadThreshold = req.QualityBadThreshold
	config.AdjustmentMagnitude = req.AdjustmentMagnitude
	config.MinEvaluationsPerFile = req.MinEvaluationsPerFile

	// Re-validate configuration
	participants, _ := h.userRepo.GetEventParticipants(eventID)
	attachments, _ := h.attachmentRepo.GetByEventID(eventID)

	if err := h.votingService.ValidateVotingConfiguration(config, len(attachments), len(participants)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid voting configuration",
			"code":    "VALIDATION_FAILED",
			"details": err.Error(),
		})
		return
	}

	// Save updated configuration
	if err := h.configRepo.Update(config); err != nil {
		h.log.Error("failed to update voting configuration", "event_id", eventID, "config_id", config.ID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update voting configuration",
			"code":  "DB_UPDATE_ERROR",
		})
		return
	}

	h.log.Info("voting configuration updated successfully", "event_id", eventID, "config_id", config.ID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":                        config.ID.String(),
			"event_id":                  config.EventID.String(),
			"attachments_per_evaluator": config.AttachmentsPerEvaluator,
			"quality_good_threshold":    config.QualityGoodThreshold,
			"quality_bad_threshold":     config.QualityBadThreshold,
			"adjustment_magnitude":      config.AdjustmentMagnitude,
			"min_evaluations_per_file":  config.MinEvaluationsPerFile,
			"updated_at":                time.Now(),
		},
		"message": "Voting configuration updated successfully",
		"code":    "CONFIG_UPDATED",
	})
}

// DeleteVotingConfiguration handles DELETE /api/events/{event_id}/voting-config
func (h *DistributedVoteHandler) DeleteVotingConfiguration(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("deleting voting configuration", "event_id", eventID)

	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
			"code":  "MISSING_EVENT_ID",
		})
		return
	}

	// Check if event exists and is still configurable
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Only allow deletion in registration stage
	if eventObj.Stage != event.StageRegistration {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Voting configuration can only be deleted during registration stage",
			"code":          "INVALID_EVENT_STAGE",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Delete configuration
	if err := h.configRepo.Delete(eventID); err != nil {
		h.log.Error("failed to delete voting configuration", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete voting configuration",
			"code":  "DB_DELETE_ERROR",
		})
		return
	}

	h.log.Info("voting configuration deleted successfully", "event_id", eventID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Voting configuration deleted successfully",
		"code":    "CONFIG_DELETED",
	})
}

// PreviewVotingConfiguration handles POST /api/events/{event_id}/voting-config/preview
func (h *DistributedVoteHandler) PreviewVotingConfiguration(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("previewing voting configuration", "event_id", eventID)

	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
			"code":  "MISSING_EVENT_ID",
		})
		return
	}

	var req struct {
		AttachmentsPerEvaluator int     `json:"attachments_per_evaluator" binding:"required,min=1,max=50"`
		QualityGoodThreshold    float64 `json:"quality_good_threshold" binding:"min=0,max=1"`
		QualityBadThreshold     float64 `json:"quality_bad_threshold" binding:"min=0,max=1"`
		AdjustmentMagnitude     int     `json:"adjustment_magnitude" binding:"min=1,max=10"`
		MinEvaluationsPerFile   int     `json:"min_evaluations_per_file" binding:"min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Get current participants and attachments
	participants, err := h.userRepo.GetEventParticipants(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get participants",
			"code":  "PARTICIPANTS_ERROR",
		})
		return
	}

	attachments, err := h.attachmentRepo.GetByEventID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get attachments",
			"code":  "ATTACHMENTS_ERROR",
		})
		return
	}

	// Create temporary configuration for validation
	eventUUID, _ := uuid.Parse(eventID)
	tempConfig := &vote.VotingConfiguration{
		EventID:                 eventUUID,
		AttachmentsPerEvaluator: req.AttachmentsPerEvaluator,
		QualityGoodThreshold:    req.QualityGoodThreshold,
		QualityBadThreshold:     req.QualityBadThreshold,
		AdjustmentMagnitude:     req.AdjustmentMagnitude,
		MinEvaluationsPerFile:   req.MinEvaluationsPerFile,
	}

	// Set defaults
	if tempConfig.QualityGoodThreshold == 0 {
		tempConfig.QualityGoodThreshold = 0.6
	}
	if tempConfig.QualityBadThreshold == 0 {
		tempConfig.QualityBadThreshold = 0.3
	}

	// Validate and calculate metrics
	validationErr := h.votingService.ValidateVotingConfiguration(tempConfig, len(attachments), len(participants))

	maxPossibleAssignments := req.AttachmentsPerEvaluator * len(participants)
	minRequiredAssignments := req.MinEvaluationsPerFile * len(attachments)

	avgEvaluationsPerFile := float64(maxPossibleAssignments) / float64(len(attachments))
	workloadBalance := float64(req.AttachmentsPerEvaluator)

	response := gin.H{
		"configuration": gin.H{
			"attachments_per_evaluator": req.AttachmentsPerEvaluator,
			"quality_good_threshold":    tempConfig.QualityGoodThreshold,
			"quality_bad_threshold":     tempConfig.QualityBadThreshold,
			"adjustment_magnitude":      req.AdjustmentMagnitude,
			"min_evaluations_per_file":  req.MinEvaluationsPerFile,
		},
		"current_data": gin.H{
			"participants_count": len(participants),
			"attachments_count":  len(attachments),
		},
		"calculated_metrics": gin.H{
			"max_possible_evaluations":  maxPossibleAssignments,
			"min_required_evaluations":  minRequiredAssignments,
			"avg_evaluations_per_file":  avgEvaluationsPerFile,
			"workload_per_participant":  workloadBalance,
			"evaluation_coverage_ratio": avgEvaluationsPerFile / float64(req.MinEvaluationsPerFile),
		},
		"validation": gin.H{
			"is_valid": validationErr == nil,
		},
	}

	if validationErr != nil {
		response["validation"].(gin.H)["error"] = validationErr.Error()
		response["validation"].(gin.H)["feasible"] = false
	} else {
		response["validation"].(gin.H)["feasible"] = true
	}

	c.JSON(http.StatusOK, response)
}
