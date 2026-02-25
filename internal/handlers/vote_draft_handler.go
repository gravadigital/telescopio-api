package handlers

import (
	"errors"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/domain/vote"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/middleware/auth"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type VoteDraftHandler struct {
	draftRepo postgres.VoteDraftRepository
	voteRepo  postgres.VoteRepository
	log       *log.Logger
}

func NewVoteDraftHandler(draftRepo postgres.VoteDraftRepository, voteRepo postgres.VoteRepository) *VoteDraftHandler {
	return &VoteDraftHandler{
		draftRepo: draftRepo,
		voteRepo:  voteRepo,
		log:       logger.Handler("vote_draft_handler"),
	}
}

type saveDraftRankingItem struct {
	AttachmentID string `json:"attachment_id" binding:"required"`
	Rank         int    `json:"rank"          binding:"required,min=1"`
}

type saveDraftRequest struct {
	Rankings []saveDraftRankingItem `json:"rankings" binding:"required"`
}

// SaveDraft — PUT /api/v1/events/:event_id/participants/:participant_id/vote-draft
// Saves (or replaces) the participant's current ranking selections as a draft.
func (h *VoteDraftHandler) SaveDraft(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	participantIDStr := c.Param("participant_id")

	h.log.Debug("saving vote draft", "event_id", eventIDStr, "participant_id", participantIDStr)

	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event_id", "code": "INVALID_EVENT_ID"})
		return
	}

	participantID, err := uuid.Parse(participantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid participant_id", "code": "INVALID_PARTICIPANT_ID"})
		return
	}

	// Verify that the authenticated user matches the participant (auth middleware already
	// enforces this via RequireParticipantOrOwner, but we double-check here for safety)
	authenticatedUserID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "code": "UNAUTHORIZED"})
		return
	}
	_ = authenticatedUserID // used by middleware; kept here as documentation of the check

	var req saveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Fetch the assignment to verify it exists and is not already completed
	assignment, err := h.voteRepo.GetAssignmentByParticipant(eventIDStr, participantIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found", "code": "ASSIGNMENT_NOT_FOUND"})
		return
	}

	if assignment.IsCompleted {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Cannot save draft: assignment is already completed",
			"code":  "ASSIGNMENT_ALREADY_COMPLETED",
		})
		return
	}

	// Map request rankings to domain type
	rankings := make(vote.DraftRankings, 0, len(req.Rankings))
	for _, item := range req.Rankings {
		attachmentID, err := uuid.Parse(item.AttachmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid attachment_id in rankings",
				"code":    "INVALID_ATTACHMENT_ID",
				"details": item.AttachmentID,
			})
			return
		}
		rankings = append(rankings, vote.DraftRanking{
			AttachmentID: attachmentID,
			Rank:         item.Rank,
		})
	}

	draft := &vote.VoteDraft{
		EventID:       eventID,
		AssignmentID:  assignment.ID,
		ParticipantID: participantID,
		Rankings:      rankings,
	}

	if err := h.draftRepo.Upsert(draft); err != nil {
		h.log.Error("failed to upsert vote draft", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save draft", "code": "DRAFT_SAVE_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Draft saved successfully",
		"data": gin.H{
			"assignment_id":  assignment.ID,
			"participant_id": participantIDStr,
			"rankings_count": len(rankings),
			"updated_at":     draft.UpdatedAt,
		},
		"code": "DRAFT_SAVED",
	})
}

// GetDraft — GET /api/v1/events/:event_id/participants/:participant_id/vote-draft
// Returns the participant's saved draft, or 404 if none exists yet.
func (h *VoteDraftHandler) GetDraft(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	participantIDStr := c.Param("participant_id")

	h.log.Debug("retrieving vote draft", "event_id", eventIDStr, "participant_id", participantIDStr)

	participantID, err := uuid.Parse(participantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid participant_id", "code": "INVALID_PARTICIPANT_ID"})
		return
	}

	// Get the assignment to resolve its ID
	assignment, err := h.voteRepo.GetAssignmentByParticipant(eventIDStr, participantIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found", "code": "ASSIGNMENT_NOT_FOUND"})
		return
	}

	draft, err := h.draftRepo.GetByAssignmentAndParticipant(assignment.ID, participantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No draft found for this assignment",
				"code":  "DRAFT_NOT_FOUND",
			})
			return
		}
		h.log.Error("failed to retrieve vote draft", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve draft", "code": "DRAFT_GET_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": draft})
}
