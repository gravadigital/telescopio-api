package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type VoteHandler struct {
	voteRepo       postgres.VoteRepository
	eventRepo      postgres.EventRepository
	attachmentRepo postgres.AttachmentRepository
	userRepo       postgres.UserRepository
}

func NewVoteHandler(voteRepo postgres.VoteRepository, eventRepo postgres.EventRepository, attachmentRepo postgres.AttachmentRepository, userRepo postgres.UserRepository) *VoteHandler {
	return &VoteHandler{
		voteRepo:       voteRepo,
		eventRepo:      eventRepo,
		attachmentRepo: attachmentRepo,
		userRepo:       userRepo,
	}
}

type SubmitVoteRequest struct {
	VoterID           string `json:"voter_id" binding:"required"`
	VotedAttachmentID string `json:"voted_attachment_id" binding:"required"`
}

// SubmitVote handles POST /api/events/{event_id}/vote
func (h *VoteHandler) SubmitVote(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
		})
		return
	}

	var req SubmitVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
		})
		return
	}

	// Check if event is in voting stage
	if eventObj.Stage != event.StageVoting {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Event is not in voting stage",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Check if voter exists and is a participant
	voter, err := h.userRepo.GetByID(req.VoterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Voter not found",
		})
		return
	}

	// Check if voter is registered for this event
	if !eventObj.IsParticipant(req.VoterID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Voter is not registered for this event",
		})
		return
	}

	// Check if attachment exists and belongs to this event
	attachment, err := h.attachmentRepo.GetByID(req.VotedAttachmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Attachment not found",
		})
		return
	}

	if attachment.EventID != eventID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Attachment does not belong to this event",
		})
		return
	}

	// Check if voter has already voted in this event
	hasVoted, err := h.voteRepo.HasVoted(eventID, req.VoterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check voting status",
		})
		return
	}

	if hasVoted {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Participant has already voted in this event",
		})
		return
	}

	// Create and save the vote
	newVote := vote.NewVote(eventID, req.VoterID, req.VotedAttachmentID)
	if err := h.voteRepo.Create(newVote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to submit vote",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"vote_id":   newVote.ID,
		"voter":     voter.Name,
		"voted_for": attachment.OriginalName,
		"voted_at":  newVote.CreatedAt,
		"message":   "Vote submitted successfully",
	})
}

// GetEventResults handles GET /api/events/{event_id}/results
func (h *VoteHandler) GetEventResults(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
		})
		return
	}

	// Check if event exists
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
		})
		return
	}

	// Check if event is in results stage
	if eventObj.Stage != event.StageResult {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         "Event results are not yet available",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Get voting results
	results, err := h.voteRepo.GetEventResults(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve results",
		})
		return
	}

	// Get all attachments for this event to provide detailed results
	attachments, err := h.attachmentRepo.GetByEventID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve attachments",
		})
		return
	}

	// Build detailed results with attachment information
	type AttachmentResult struct {
		ID            string `json:"attachment_id"`
		Filename      string `json:"filename"`
		ParticipantID string `json:"participant_id"`
		VoteCount     int    `json:"vote_count"`
		Rank          int    `json:"rank"`
	}

	var detailedResults []AttachmentResult
	for _, att := range attachments {
		voteCount, exists := results[att.ID]
		if !exists {
			voteCount = 0
		}

		detailedResults = append(detailedResults, AttachmentResult{
			ID:            att.ID,
			Filename:      att.OriginalName,
			ParticipantID: att.ParticipantID,
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

	// Assign ranks
	for i := range detailedResults {
		detailedResults[i].Rank = i + 1
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":    eventID,
		"event_name":  eventObj.Name,
		"total_votes": len(results),
		"results":     detailedResults,
	})
}
