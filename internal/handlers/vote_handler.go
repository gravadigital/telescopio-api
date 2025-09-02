package handlers

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type VoteHandler struct {
	voteRepo          postgres.VoteRepository
	eventRepo         postgres.EventRepository
	attachmentRepo    postgres.AttachmentRepository
	userRepo          postgres.UserRepository
	votingResultsRepo postgres.VotingResultsRepository
	config            *config.Config
	log               *log.Logger
}

func NewVoteHandler(voteRepo postgres.VoteRepository, eventRepo postgres.EventRepository, attachmentRepo postgres.AttachmentRepository, userRepo postgres.UserRepository, votingResultsRepo postgres.VotingResultsRepository, cfg *config.Config) *VoteHandler {
	return &VoteHandler{
		voteRepo:          voteRepo,
		eventRepo:         eventRepo,
		attachmentRepo:    attachmentRepo,
		userRepo:          userRepo,
		votingResultsRepo: votingResultsRepo,
		config:            cfg,
		log:               logger.Handler("vote_handler"),
	}
}

type SubmitVoteRequest struct {
	VoterID           string `json:"voter_id" binding:"required"`
	VotedAttachmentID string `json:"voted_attachment_id" binding:"required"`
}

// Standard response structures
type VoteResponse struct {
	VoteID   string `json:"vote_id"`
	Voter    string `json:"voter"`
	VotedFor string `json:"voted_for"`
	VotedAt  string `json:"voted_at"`
	Message  string `json:"message"`
}

type AttachmentResult struct {
	ID            string `json:"attachment_id"`
	Filename      string `json:"filename"`
	ParticipantID string `json:"participant_id"`
	VoteCount     int    `json:"vote_count"`
	Rank          int    `json:"rank"`
}

type ResultsResponse struct {
	EventID    string             `json:"event_id"`
	EventName  string             `json:"event_name"`
	TotalVotes int                `json:"total_votes"`
	Results    []AttachmentResult `json:"results"`
}

// Error response helper
func (h *VoteHandler) errorResponse(c *gin.Context, statusCode int, message string, err error, details map[string]interface{}) {
	response := gin.H{
		"error":   message,
		"code":    fmt.Sprintf("VOTE_%d", statusCode),
		"success": false,
	}

	if err != nil {
		h.log.Error(message, "error", err, "details", details)
		if h.config.Server.GinMode == "debug" {
			response["debug"] = err.Error()
		}
	} else {
		h.log.Warn(message, "details", details)
	}

	for k, v := range details {
		response[k] = v
	}

	c.JSON(statusCode, response)
}

// Success response helper
func (h *VoteHandler) successResponse(c *gin.Context, statusCode int, data interface{}, message string) {
	response := gin.H{
		"success": true,
		"message": message,
		"data":    data,
	}

	h.log.Info(message, "response", data)
	c.JSON(statusCode, response)
}

// Validation helpers
func (h *VoteHandler) validateUUID(id, fieldName string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid %s format: %w", fieldName, err)
	}
	return nil
}

func (h *VoteHandler) validateEventForVoting(eventObj *event.Event) error {
	if eventObj.Stage != event.StageVoting {
		return fmt.Errorf("event is not in voting stage, current stage: %s", eventObj.Stage.String())
	}
	return nil
}

func (h *VoteHandler) validateEventForResults(eventObj *event.Event) error {
	if eventObj.Stage != event.StageResult {
		return fmt.Errorf("event results are not yet available, current stage: %s", eventObj.Stage.String())
	}
	return nil
}

func (h *VoteHandler) checkParticipantRegistration(eventID, participantID string) (bool, error) {
	h.log.Debug("Checking participant registration", "event_id", eventID, "participant_id", participantID)

	participantEvents, err := h.eventRepo.GetByParticipant(participantID)
	if err != nil {
		return false, fmt.Errorf("failed to check participant registration: %w", err)
	}

	for _, evt := range participantEvents {
		if evt.ID.String() == eventID {
			h.log.Debug("Participant registration confirmed", "event_id", eventID, "participant_id", participantID)
			return true, nil
		}
	}

	h.log.Warn("Participant not registered for event", "event_id", eventID, "participant_id", participantID)
	return false, nil
}

// SubmitVote handles POST /api/events/{event_id}/vote
func (h *VoteHandler) SubmitVote(c *gin.Context) {
	eventID := c.Param("event_id")
	h.log.Info("Processing vote submission", "event_id", eventID)

	if eventID == "" {
		h.errorResponse(c, http.StatusBadRequest, "Event ID is required", nil, nil)
		return
	}

	// Validate event ID format
	if err := h.validateUUID(eventID, "event_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid event ID format", err, nil)
		return
	}

	var req SubmitVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid request payload", err, nil)
		return
	}

	// Validate request UUIDs
	if err := h.validateUUID(req.VoterID, "voter_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid voter ID format", err, nil)
		return
	}

	if err := h.validateUUID(req.VotedAttachmentID, "voted_attachment_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid attachment ID format", err, nil)
		return
	}

	h.log.Debug("Vote submission request validated",
		"event_id", eventID,
		"voter_id", req.VoterID,
		"attachment_id", req.VotedAttachmentID)

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Event not found", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	// Check if event is in voting stage
	if err := h.validateEventForVoting(eventObj); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Event not available for voting", err, map[string]interface{}{
			"current_stage": eventObj.Stage.String(),
			"event_id":      eventID,
		})
		return
	}

	// Check if voter exists
	voter, err := h.userRepo.GetByID(req.VoterID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Voter not found", err, map[string]interface{}{
			"voter_id": req.VoterID,
		})
		return
	}

	// Check if voter is registered for this event
	isParticipant, err := h.checkParticipantRegistration(eventID, req.VoterID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to verify participant registration", err, nil)
		return
	}

	if !isParticipant {
		h.errorResponse(c, http.StatusForbidden, "Voter is not registered for this event", nil, map[string]interface{}{
			"voter_id": req.VoterID,
			"event_id": eventID,
		})
		return
	}

	// Check if attachment exists and belongs to this event
	attachment, err := h.attachmentRepo.GetByID(req.VotedAttachmentID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Attachment not found", err, map[string]interface{}{
			"attachment_id": req.VotedAttachmentID,
		})
		return
	}

	if attachment.EventID.String() != eventID {
		h.errorResponse(c, http.StatusBadRequest, "Attachment does not belong to this event", nil, map[string]interface{}{
			"attachment_id":      req.VotedAttachmentID,
			"attachment_event":   attachment.EventID.String(),
			"requested_event_id": eventID,
		})
		return
	}

	// Check if voter has already voted in this event
	hasVoted, err := h.voteRepo.HasVoted(eventID, req.VoterID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to check voting status", err, nil)
		return
	}

	if hasVoted {
		h.errorResponse(c, http.StatusConflict, "Participant has already voted in this event", nil, map[string]interface{}{
			"voter_id": req.VoterID,
			"event_id": eventID,
		})
		return
	}

	// Parse UUIDs for creating the vote
	eventUUID, _ := uuid.Parse(eventID)
	voterUUID, _ := uuid.Parse(req.VoterID)
	attachmentUUID, _ := uuid.Parse(req.VotedAttachmentID)

	// Create and save the vote (using rank 1 for backward compatibility)
	newVote := vote.NewVote(eventUUID, voterUUID, attachmentUUID, 1)
	if err := h.voteRepo.Create(newVote); err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to submit vote", err, map[string]interface{}{
			"vote_id": newVote.ID.String(),
		})
		return
	}

	h.log.Info("Vote submitted successfully",
		"vote_id", newVote.ID.String(),
		"voter_id", req.VoterID,
		"event_id", eventID,
		"attachment_id", req.VotedAttachmentID)

	response := VoteResponse{
		VoteID:   newVote.ID.String(),
		Voter:    voter.Name,
		VotedFor: attachment.OriginalName,
		VotedAt:  newVote.VotedAt.Format("2006-01-02T15:04:05Z07:00"),
		Message:  "Vote submitted successfully",
	}

	h.successResponse(c, http.StatusCreated, response, "Vote submitted successfully")
}

// GetEventResults handles GET /api/events/{event_id}/results
func (h *VoteHandler) GetEventResults(c *gin.Context) {
	eventID := c.Param("event_id")
	h.log.Info("Processing results request", "event_id", eventID)

	if eventID == "" {
		h.errorResponse(c, http.StatusBadRequest, "Event ID is required", nil, nil)
		return
	}

	// Validate event ID format
	if err := h.validateUUID(eventID, "event_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid event ID format", err, nil)
		return
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Event not found", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	// Check if event is in results stage
	if err := h.validateEventForResults(eventObj); err != nil {
		h.errorResponse(c, http.StatusForbidden, "Event results not available", err, map[string]interface{}{
			"current_stage": eventObj.Stage.String(),
			"event_id":      eventID,
		})
		return
	}

	h.log.Debug("Event validation passed for results", "event_id", eventID, "stage", eventObj.Stage.String())

	// Get voting results
	results, err := h.votingResultsRepo.GetByEventID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve voting results", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	// Create a map for quick lookup of results by attachment ID
	resultsMap := make(map[string]int)
	totalVotes := 0
	for _, result := range results.GlobalRanking {
		resultsMap[result.AttachmentID.String()] = result.VoteCount
		totalVotes += result.VoteCount
	}

	h.log.Debug("Voting results retrieved", "event_id", eventID, "total_votes", totalVotes)

	// Get all attachments for this event to provide detailed results
	attachments, err := h.attachmentRepo.GetByEventID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve event attachments", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	// Build detailed results with attachment information
	var detailedResults []AttachmentResult
	for _, att := range attachments {
		voteCount, exists := resultsMap[att.ID.String()]
		if !exists {
			voteCount = 0
		}

		detailedResults = append(detailedResults, AttachmentResult{
			ID:            att.ID.String(),
			Filename:      att.OriginalName,
			ParticipantID: att.ParticipantID.String(),
			VoteCount:     voteCount,
		})
	}

	// Sort by vote count (highest first) and assign ranks
	for i := 0; i < len(detailedResults); i++ {
		for j := i + 1; j < len(detailedResults); j++ {
			if detailedResults[j].VoteCount > detailedResults[i].VoteCount {
				detailedResults[i], detailedResults[j] = detailedResults[j], detailedResults[i]
			}
		}
	}

	// Assign ranks with tie handling
	for i := range detailedResults {
		if i == 0 || detailedResults[i].VoteCount != detailedResults[i-1].VoteCount {
			detailedResults[i].Rank = i + 1
		} else {
			detailedResults[i].Rank = detailedResults[i-1].Rank
		}
	}

	h.log.Info("Results compiled successfully",
		"event_id", eventID,
		"total_attachments", len(detailedResults),
		"total_votes", totalVotes)

	response := ResultsResponse{
		EventID:    eventID,
		EventName:  eventObj.Name,
		TotalVotes: totalVotes,
		Results:    detailedResults,
	}

	h.successResponse(c, http.StatusOK, response, "Event results retrieved successfully")
}

// GetVotesByEvent handles GET /api/events/{event_id}/votes
func (h *VoteHandler) GetVotesByEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	h.log.Info("Processing votes by event request", "event_id", eventID)

	if eventID == "" {
		h.errorResponse(c, http.StatusBadRequest, "Event ID is required", nil, nil)
		return
	}

	// Validate event ID format
	if err := h.validateUUID(eventID, "event_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid event ID format", err, nil)
		return
	}

	// Check if event exists
	_, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Event not found", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	// Get votes by event ID
	votes, err := h.voteRepo.GetByEventID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve votes", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	h.log.Info("Votes retrieved successfully", "event_id", eventID, "vote_count", len(votes))
	h.successResponse(c, http.StatusOK, votes, "Votes retrieved successfully")
}

// GetVotesByVoter handles GET /api/votes/voter/{voter_id}
func (h *VoteHandler) GetVotesByVoter(c *gin.Context) {
	voterID := c.Param("voter_id")
	h.log.Info("Processing votes by voter request", "voter_id", voterID)

	if voterID == "" {
		h.errorResponse(c, http.StatusBadRequest, "Voter ID is required", nil, nil)
		return
	}

	// Validate voter ID format
	if err := h.validateUUID(voterID, "voter_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid voter ID format", err, nil)
		return
	}

	// Check if voter exists
	_, err := h.userRepo.GetByID(voterID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Voter not found", err, map[string]interface{}{
			"voter_id": voterID,
		})
		return
	}

	// Get votes by voter ID
	votes, err := h.voteRepo.GetByVoterID(voterID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to retrieve votes", err, map[string]interface{}{
			"voter_id": voterID,
		})
		return
	}

	h.log.Info("Votes retrieved successfully", "voter_id", voterID, "vote_count", len(votes))
	h.successResponse(c, http.StatusOK, votes, "Voter votes retrieved successfully")
}

// GetVotingStatus handles GET /api/events/{event_id}/voting-status/{voter_id}
func (h *VoteHandler) GetVotingStatus(c *gin.Context) {
	eventID := c.Param("event_id")
	voterID := c.Param("voter_id")
	h.log.Info("Processing voting status request", "event_id", eventID, "voter_id", voterID)

	if eventID == "" || voterID == "" {
		h.errorResponse(c, http.StatusBadRequest, "Event ID and voter ID are required", nil, nil)
		return
	}

	// Validate UUID formats
	if err := h.validateUUID(eventID, "event_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid event ID format", err, nil)
		return
	}

	if err := h.validateUUID(voterID, "voter_id"); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "Invalid voter ID format", err, nil)
		return
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Event not found", err, map[string]interface{}{
			"event_id": eventID,
		})
		return
	}

	// Check if voter exists
	_, err = h.userRepo.GetByID(voterID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "Voter not found", err, map[string]interface{}{
			"voter_id": voterID,
		})
		return
	}

	// Check if voter is registered for this event
	isParticipant, err := h.checkParticipantRegistration(eventID, voterID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to verify participant registration", err, nil)
		return
	}

	if !isParticipant {
		h.errorResponse(c, http.StatusForbidden, "Voter is not registered for this event", nil, map[string]interface{}{
			"voter_id": voterID,
			"event_id": eventID,
		})
		return
	}

	// Check voting status
	hasVoted, err := h.voteRepo.HasVoted(eventID, voterID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "Failed to check voting status", err, nil)
		return
	}

	status := map[string]interface{}{
		"event_id":    eventID,
		"voter_id":    voterID,
		"has_voted":   hasVoted,
		"event_stage": eventObj.Stage.String(),
		"can_vote":    eventObj.Stage == event.StageVoting && !hasVoted,
		"event_name":  eventObj.Name,
	}

	h.log.Info("Voting status retrieved", "event_id", eventID, "voter_id", voterID, "has_voted", hasVoted)
	h.successResponse(c, http.StatusOK, status, "Voting status retrieved successfully")
}
