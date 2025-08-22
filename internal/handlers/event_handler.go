package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type EventHandler struct {
	eventRepo postgres.EventRepository
	userRepo  postgres.UserRepository
}

func NewEventHandler(eventRepo postgres.EventRepository, userRepo postgres.UserRepository) *EventHandler {
	return &EventHandler{
		eventRepo: eventRepo,
		userRepo:  userRepo,
	}
}

type CreateEventRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
}

// CreateEvent handles POST /api/events
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start_date format",
			"details": "Expected format: YYYY-MM-DD",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"details": "Expected format: YYYY-MM-DD",
		})
		return
	}

	// Validate dates
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "end_date must be after start_date",
		})
		return
	}

	// For now, we'll use a placeholder user ID. In a real app, this would come from authentication
	authorID := "user_1" // TODO: Get from authentication middleware

	newEvent := event.NewEvent(req.Name, req.Description, authorID, startDate, endDate)

	if err := h.eventRepo.Create(newEvent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create event",
		})
		return
	}

	c.JSON(http.StatusCreated, newEvent)
}

type UpdateStageRequest struct {
	Stage string `json:"stage" binding:"required"`
}

// UpdateEventStage handles PATCH /api/events/{event_id}/stage
func (h *EventHandler) UpdateEventStage(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
		})
		return
	}

	var req UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Check if user is admin (TODO: implement proper authorization)
	// For now, we'll skip this check

	// Get the event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
		})
		return
	}

	// Parse and validate the new stage
	newStage, valid := event.StageFromString(req.Stage)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "Invalid stage",
			"valid_stages": []string{"registration", "attachment_upload", "voting", "results"},
		})
		return
	}

	// Check if transition is valid
	if !existingEvent.CanTransitionTo(newStage) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Invalid stage transition",
			"current_stage":   existingEvent.Stage.String(),
			"requested_stage": req.Stage,
		})
		return
	}

	// Update the stage
	if err := h.eventRepo.UpdateStage(eventID, newStage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update event stage",
		})
		return
	}

	// Get updated event
	updatedEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve updated event",
		})
		return
	}

	c.JSON(http.StatusOK, updatedEvent)
}

type RegisterParticipantRequest struct {
	ParticipantName  string `json:"participant_name" binding:"required"`
	ParticipantEmail string `json:"participant_email" binding:"required,email"`
}

// RegisterParticipant handles POST /api/events/{event_id}/register
func (h *EventHandler) RegisterParticipant(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
		})
		return
	}

	var req RegisterParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Check if event exists
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
		})
		return
	}

	// Check if user already exists by email
	existingUser, err := h.userRepo.GetByEmail(req.ParticipantEmail)
	if err != nil {
		// User doesn't exist, create a new one
		newUser := &participant.User{
			ID:              generateUserID(), // TODO: Implement proper ID generation
			Name:            req.ParticipantName,
			Role:            "participant",
			JoinedEventIDs:  make([]string, 0),
			CreatedEventIDs: make([]string, 0),
		}

		if err := h.userRepo.Create(newUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create participant",
			})
			return
		}
		existingUser = newUser
	}

	// Check if participant is already registered
	if existingEvent.IsParticipant(existingUser.ID) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Participant is already registered for this event",
		})
		return
	}

	// Add participant to event
	if err := h.eventRepo.AddParticipant(eventID, existingUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register participant",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"participant_id":   existingUser.ID,
		"participant_name": existingUser.Name,
		"event_id":         eventID,
		"message":          "Participant registered successfully",
	})
}

// GetEventParticipants handles GET /api/events/{event_id}/participants
func (h *EventHandler) GetEventParticipants(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id is required",
		})
		return
	}

	// Check if event exists
	_, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
		})
		return
	}

	// TODO: Check if user is admin
	// For now, we'll skip this check

	participants, err := h.userRepo.GetEventParticipants(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve participants",
		})
		return
	}

	c.JSON(http.StatusOK, participants)
}

// GetAllEvents handles GET /api/events
func (h *EventHandler) GetAllEvents(c *gin.Context) {
	events, err := h.eventRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve events",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

func generateUserID() string {
	return "user_" + time.Now().Format("20060102150405")
}
