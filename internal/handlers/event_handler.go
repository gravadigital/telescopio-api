package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type EventHandler struct {
	eventRepo postgres.EventRepository
	userRepo  postgres.UserRepository
	config    *config.Config
	log       *log.Logger
}

func NewEventHandler(eventRepo postgres.EventRepository, userRepo postgres.UserRepository, cfg *config.Config) *EventHandler {
	return &EventHandler{
		eventRepo: eventRepo,
		userRepo:  userRepo,
		config:    cfg,
		log:       logger.Handler("event"),
	}
}

type CreateEventRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=200"`
	Description string `json:"description" binding:"required,min=10,max=2000"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
}

// CreateEvent handles POST /api/events
func (h *EventHandler) CreateEvent(c *gin.Context) {
	h.log.Debug("received create event request")

	// TODO: Add authentication check
	// userID := c.GetString("user_id") // From JWT middleware
	// if userID == "" {
	//     h.log.Warn("unauthenticated create event attempt")
	//     c.JSON(http.StatusUnauthorized, gin.H{
	//         "error": "Authentication required",
	//         "code":  "UNAUTHORIZED",
	//     })
	//     return
	// }

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("invalid request payload for create event", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Parse and validate dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		h.log.Warn("invalid start_date format", "start_date", req.StartDate, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start_date format",
			"code":    "INVALID_START_DATE",
			"details": "Expected format: YYYY-MM-DD",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		h.log.Warn("invalid end_date format", "end_date", req.EndDate, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"code":    "INVALID_END_DATE",
			"details": "Expected format: YYYY-MM-DD",
		})
		return
	}

	// Business validation for dates
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if startDate.Before(today) {
		h.log.Warn("start_date is in the past", "start_date", req.StartDate)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Start date cannot be in the past",
			"code":  "PAST_START_DATE",
		})
		return
	}

	if endDate.Before(startDate) {
		h.log.Warn("end_date before start_date", "start_date", req.StartDate, "end_date", req.EndDate)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "End date must be after start date",
			"code":  "INVALID_DATE_RANGE",
		})
		return
	}

	// Validate event duration (minimum 1 day, maximum 1 year)
	duration := endDate.Sub(startDate)
	minDuration := 24 * time.Hour
	maxDuration := 365 * 24 * time.Hour

	if duration < minDuration {
		h.log.Warn("event duration too short", "duration", duration)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Event duration must be at least 1 day",
			"code":  "DURATION_TOO_SHORT",
		})
		return
	}

	if duration > maxDuration {
		h.log.Warn("event duration too long", "duration", duration)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Event duration cannot exceed 1 year",
			"code":  "DURATION_TOO_LONG",
		})
		return
	}

	// TODO: Get authorID from authentication
	// user, err := h.userRepo.GetByID(userID)
	// if err != nil {
	//     h.log.Error("user not found", "user_id", userID, "error", err)
	//     c.JSON(http.StatusNotFound, gin.H{
	//         "error": "User not found",
	//         "code":  "USER_NOT_FOUND",
	//     })
	//     return
	// }

	// For now, use a placeholder author ID
	authorID := uuid.New() // TODO: Replace with authenticated user ID

	// Check for duplicate event names (optional business rule)
	existingEvents, err := h.eventRepo.GetAll()
	if err == nil {
		for _, existingEvent := range existingEvents {
			if existingEvent.Name == req.Name {
				h.log.Warn("duplicate event name", "event_name", req.Name)
				c.JSON(http.StatusConflict, gin.H{
					"error": "An event with this name already exists",
					"code":  "DUPLICATE_EVENT_NAME",
				})
				return
			}
		}
	}

	newEvent := event.NewEvent(req.Name, req.Description, authorID, startDate, endDate)

	// Validate the event domain entity
	if err := newEvent.Validate(); err != nil {
		h.log.Error("event validation failed", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Event validation failed",
			"code":    "VALIDATION_FAILED",
			"details": err.Error(),
		})
		return
	}

	if err := h.eventRepo.Create(newEvent); err != nil {
		h.log.Error("failed to create event", "error", err, "event_name", req.Name)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create event",
			"code":  "DB_CREATE_ERROR",
		})
		return
	}

	h.log.Info("event created successfully", "event_id", newEvent.ID, "event_name", newEvent.Name, "author_id", authorID)

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"id":          newEvent.ID.String(),
			"name":        newEvent.Name,
			"description": newEvent.Description,
			"start_date":  newEvent.StartDate.Format("2006-01-02"),
			"end_date":    newEvent.EndDate.Format("2006-01-02"),
			"stage":       newEvent.Stage.String(),
			"author_id":   newEvent.AuthorID.String(),
			"created_at":  newEvent.CreatedAt,
		},
		"message": "Event created successfully",
		"code":    "EVENT_CREATED",
	})
}

type UpdateStageRequest struct {
	Stage string `json:"stage" binding:"required"`
}

// UpdateEventStage handles PATCH /api/events/{event_id}/stage
func (h *EventHandler) UpdateEventStage(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("updating event stage", "event_id", eventID)

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
	if _, err := uuid.Parse(eventID); err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	var req UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid request payload for stage update", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// TODO: Check if user is admin or event owner
	// userID := c.GetString("user_id") // From JWT middleware
	// user, err := h.userRepo.GetByID(userID)
	// if err != nil || (!user.HasRole(participant.RoleAdmin) && event.AuthorID.String() != userID) {
	//     h.log.Warn("unauthorized stage update attempt", "user_id", userID, "event_id", eventID)
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Insufficient permissions to update event stage",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	// Get the event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Parse and validate the new stage
	newStage, valid := event.StageFromString(req.Stage)
	if !valid {
		h.log.Warn("invalid stage requested", "requested_stage", req.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "Invalid stage",
			"code":         "INVALID_STAGE",
			"valid_stages": []string{"creation", "registration", "attachment_upload", "voting", "results"},
		})
		return
	}

	// Check if transition is valid
	if !existingEvent.CanTransitionTo(newStage) {
		h.log.Warn("invalid stage transition",
			"event_id", eventID,
			"current_stage", existingEvent.Stage.String(),
			"requested_stage", req.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Invalid stage transition",
			"code":            "INVALID_TRANSITION",
			"current_stage":   existingEvent.Stage.String(),
			"requested_stage": req.Stage,
		})
		return
	}

	// Additional business rules validation before stage transitions
	switch newStage {
	case event.StageSubmission:
		// Check if there are participants registered
		participants, err := h.userRepo.GetEventParticipants(eventID)
		if err != nil || len(participants) == 0 {
			h.log.Warn("attempting to move to submission stage without participants", "event_id", eventID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cannot move to submission stage without registered participants",
				"code":  "NO_PARTICIPANTS",
			})
			return
		}

	case event.StageVoting:
		// Check if there are attachments submitted
		// This would require an attachment repository check
		// For now, we'll skip this validation

	case event.StageResult:
		// Check if voting is complete
		// This would require a vote repository check
		// For now, we'll skip this validation
	}

	// Update the stage
	if err := h.eventRepo.UpdateStage(eventID, newStage); err != nil {
		h.log.Error("failed to update event stage", "event_id", eventID, "new_stage", req.Stage, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update event stage",
			"code":  "DB_UPDATE_ERROR",
		})
		return
	}

	// Get updated event
	updatedEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("failed to retrieve updated event", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve updated event",
			"code":  "RETRIEVAL_ERROR",
		})
		return
	}

	h.log.Info("event stage updated successfully",
		"event_id", eventID,
		"old_stage", existingEvent.Stage.String(),
		"new_stage", updatedEvent.Stage.String())

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":          updatedEvent.ID.String(),
			"name":        updatedEvent.Name,
			"description": updatedEvent.Description,
			"start_date":  updatedEvent.StartDate.Format("2006-01-02"),
			"end_date":    updatedEvent.EndDate.Format("2006-01-02"),
			"stage":       updatedEvent.Stage.String(),
			"author_id":   updatedEvent.AuthorID.String(),
			"updated_at":  updatedEvent.UpdatedAt,
		},
		"message": "Event stage updated successfully",
		"code":    "STAGE_UPDATED",
		"transition": gin.H{
			"from": existingEvent.Stage.String(),
			"to":   updatedEvent.Stage.String(),
		},
	})
}

type RegisterParticipantRequest struct {
	ParticipantName  string `json:"participant_name" binding:"required,min=2,max=100"`
	ParticipantEmail string `json:"participant_email" binding:"required,email"`
}

// RegisterParticipant handles POST /api/events/{event_id}/register
func (h *EventHandler) RegisterParticipant(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("registering participant", "event_id", eventID)

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

	var req RegisterParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid request payload for participant registration", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Check if event exists and is in registration stage
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Only allow registration during registration stage
	if eventObj.Stage != event.StageRegistration {
		h.log.Warn("registration attempt outside registration stage",
			"event_id", eventID,
			"current_stage", eventObj.Stage.String())
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Participant registration is only allowed during registration stage",
			"code":          "INVALID_REGISTRATION_STAGE",
			"current_stage": eventObj.Stage.String(),
		})
		return
	}

	// Check if registration is still open (event hasn't started)
	now := time.Now()
	if now.After(eventObj.StartDate) {
		h.log.Warn("registration attempt after event start", "event_id", eventID, "start_date", eventObj.StartDate)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Registration is closed - event has already started",
			"code":  "REGISTRATION_CLOSED",
		})
		return
	}

	// Check if user already exists by email
	existingUser, err := h.userRepo.GetByEmail(req.ParticipantEmail)
	if err != nil {
		// User doesn't exist, create a new one
		h.log.Debug("creating new user", "email", req.ParticipantEmail, "name", req.ParticipantName)

		newUser := &participant.User{
			ID:    uuid.New(),
			Name:  req.ParticipantName,
			Email: req.ParticipantEmail,
			Role:  participant.RoleParticipant,
		}

		if err := h.userRepo.Create(newUser); err != nil {
			h.log.Error("failed to create participant", "email", req.ParticipantEmail, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create participant",
				"code":  "USER_CREATE_ERROR",
			})
			return
		}
		existingUser = newUser
		h.log.Info("new user created", "user_id", newUser.ID, "email", newUser.Email)
	} else {
		h.log.Debug("using existing user", "user_id", existingUser.ID, "email", existingUser.Email)
	}

	// Check if participant is already registered for this event
	participantEvents, err := h.eventRepo.GetByParticipant(existingUser.ID.String())
	if err == nil && len(participantEvents) > 0 {
		for _, evt := range participantEvents {
			if evt.ID == eventUUID {
				h.log.Warn("duplicate registration attempt",
					"event_id", eventID,
					"user_id", existingUser.ID.String())
				c.JSON(http.StatusConflict, gin.H{
					"error": "Participant is already registered for this event",
					"code":  "ALREADY_REGISTERED",
				})
				return
			}
		}
	}

	// Check maximum participants limit (optional business rule)
	currentParticipants, err := h.userRepo.GetEventParticipants(eventID)
	maxParticipants := 100 // This could come from configuration or event settings
	if err == nil && len(currentParticipants) >= maxParticipants {
		h.log.Warn("maximum participants reached",
			"event_id", eventID,
			"current_count", len(currentParticipants),
			"max_participants", maxParticipants)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":            "Maximum number of participants reached for this event",
			"code":             "MAX_PARTICIPANTS_REACHED",
			"current_count":    len(currentParticipants),
			"max_participants": maxParticipants,
		})
		return
	}

	// Add participant to event
	if err := h.eventRepo.AddParticipant(eventID, existingUser.ID.String()); err != nil {
		h.log.Error("failed to register participant",
			"event_id", eventID,
			"user_id", existingUser.ID.String(),
			"error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register participant",
			"code":  "REGISTRATION_ERROR",
		})
		return
	}

	h.log.Info("participant registered successfully",
		"event_id", eventID,
		"user_id", existingUser.ID.String(),
		"email", existingUser.Email)

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"participant_id":    existingUser.ID.String(),
			"participant_name":  existingUser.Name,
			"participant_email": existingUser.Email,
			"event_id":          eventID,
			"event_name":        eventObj.Name,
			"registered_at":     time.Now(),
		},
		"message": "Participant registered successfully",
		"code":    "PARTICIPANT_REGISTERED",
	})
}

// GetEventParticipants handles GET /api/events/{event_id}/participants
func (h *EventHandler) GetEventParticipants(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("retrieving event participants", "event_id", eventID)

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
	if _, err := uuid.Parse(eventID); err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
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

	// TODO: Check if user has permission to view participants
	// This might depend on the event stage or user role
	// userID := c.GetString("user_id")
	// if eventObj.Stage == event.StageCreation && eventObj.AuthorID.String() != userID {
	//     // Only event owner can see participants during creation
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Insufficient permissions",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	participants, err := h.userRepo.GetEventParticipants(eventID)
	if err != nil {
		h.log.Error("failed to retrieve participants", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve participants",
			"code":  "RETRIEVAL_ERROR",
		})
		return
	}

	// Transform participants data
	participantData := make([]gin.H, len(participants))
	for i, p := range participants {
		participantData[i] = gin.H{
			"id":         p.ID.String(),
			"name":       p.Name,
			"email":      p.Email,
			"role":       p.Role.String(),
			"created_at": p.CreatedAt,
		}
	}

	h.log.Debug("participants retrieved successfully", "event_id", eventID, "count", len(participants))

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"event": gin.H{
				"id":    eventObj.ID.String(),
				"name":  eventObj.Name,
				"stage": eventObj.Stage.String(),
			},
			"participants": participantData,
		},
		"count": len(participants),
	})
}

// GetAllEvents handles GET /api/events
func (h *EventHandler) GetAllEvents(c *gin.Context) {
	h.log.Debug("retrieving all events")

	// Add pagination support
	page := 1
	limit := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Add filtering support
	stage := c.Query("stage")

	events, err := h.eventRepo.GetAll()
	if err != nil {
		h.log.Error("failed to retrieve events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve events",
			"code":  "RETRIEVAL_ERROR",
		})
		return
	}

	// Filter by stage if specified
	var filteredEvents []*event.Event
	if stage != "" {
		if stageEnum, valid := event.StageFromString(stage); valid {
			for _, evt := range events {
				if evt.Stage == stageEnum {
					filteredEvents = append(filteredEvents, evt)
				}
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        "Invalid stage filter",
				"code":         "INVALID_STAGE_FILTER",
				"valid_stages": []string{"creation", "registration", "attachment_upload", "voting", "results"},
			})
			return
		}
	} else {
		filteredEvents = events
	}

	// Apply pagination
	total := len(filteredEvents)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		filteredEvents = []*event.Event{}
	} else {
		if end > total {
			end = total
		}
		filteredEvents = filteredEvents[start:end]
	}

	// Transform events data
	eventData := make([]gin.H, len(filteredEvents))
	for i, evt := range filteredEvents {
		eventData[i] = gin.H{
			"id":          evt.ID.String(),
			"name":        evt.Name,
			"description": evt.Description,
			"start_date":  evt.StartDate.Format("2006-01-02"),
			"end_date":    evt.EndDate.Format("2006-01-02"),
			"stage":       evt.Stage.String(),
			"author_id":   evt.AuthorID.String(),
			"created_at":  evt.CreatedAt,
			"updated_at":  evt.UpdatedAt,
		}
	}

	h.log.Debug("events retrieved successfully", "total", total, "page", page, "limit", limit)

	c.JSON(http.StatusOK, gin.H{
		"data": eventData,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
		"filters": gin.H{
			"stage": stage,
		},
	})
}

// GetEvent handles GET /api/events/{event_id}
func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("retrieving event", "event_id", eventID)

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
	if _, err := uuid.Parse(eventID); err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	// Get the event
	eventObj, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Get additional statistics if requested
	includeStats := c.Query("include_stats") == "true"
	response := gin.H{
		"data": gin.H{
			"id":          eventObj.ID.String(),
			"name":        eventObj.Name,
			"description": eventObj.Description,
			"start_date":  eventObj.StartDate.Format("2006-01-02"),
			"end_date":    eventObj.EndDate.Format("2006-01-02"),
			"stage":       eventObj.Stage.String(),
			"author_id":   eventObj.AuthorID.String(),
			"created_at":  eventObj.CreatedAt,
			"updated_at":  eventObj.UpdatedAt,
		},
	}

	if includeStats {
		// Get participant count
		participants, err := h.userRepo.GetEventParticipants(eventID)
		participantCount := 0
		if err == nil {
			participantCount = len(participants)
		}

		response["statistics"] = gin.H{
			"participants_count": participantCount,
			"duration_days":      int(eventObj.EndDate.Sub(eventObj.StartDate).Hours() / 24),
		}
	}

	h.log.Debug("event retrieved successfully", "event_id", eventID)
	c.JSON(http.StatusOK, response)
}

// UpdateEvent handles PUT /api/events/{event_id}
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("updating event", "event_id", eventID)

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
	if _, err := uuid.Parse(eventID); err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	// Get existing event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Only allow updates in creation stage
	if existingEvent.Stage != event.StageCreation {
		h.log.Warn("update attempt outside creation stage", "event_id", eventID, "current_stage", existingEvent.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Event can only be updated during creation stage",
			"code":          "INVALID_UPDATE_STAGE",
			"current_stage": existingEvent.Stage.String(),
		})
		return
	}

	// TODO: Check if user is event owner
	// userID := c.GetString("user_id")
	// if existingEvent.AuthorID.String() != userID {
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Only event owner can update the event",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	var req CreateEventRequest // Reuse the same struct
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid request payload for event update", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Parse and validate dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		h.log.Warn("invalid start_date format", "start_date", req.StartDate, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start_date format",
			"code":    "INVALID_START_DATE",
			"details": "Expected format: YYYY-MM-DD",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		h.log.Warn("invalid end_date format", "end_date", req.EndDate, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"code":    "INVALID_END_DATE",
			"details": "Expected format: YYYY-MM-DD",
		})
		return
	}

	// Validate business rules
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if startDate.Before(today) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Start date cannot be in the past",
			"code":  "PAST_START_DATE",
		})
		return
	}

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "End date must be after start date",
			"code":  "INVALID_DATE_RANGE",
		})
		return
	}

	// Update event fields - since there's no Update method, we'll comment this out for now
	// TODO: Add Update method to EventRepository interface
	// existingEvent.Name = req.Name
	// existingEvent.Description = req.Description
	// existingEvent.StartDate = startDate
	// existingEvent.EndDate = endDate

	// For now, return an error indicating this feature is not implemented
	h.log.Warn("event update feature not implemented", "event_id", eventID)
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Event update feature is not yet implemented",
		"code":    "NOT_IMPLEMENTED",
		"details": "EventRepository.Update method needs to be added",
	})
}

// DeleteEvent handles DELETE /api/events/{event_id}
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("deleting event", "event_id", eventID)

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
	if _, err := uuid.Parse(eventID); err != nil {
		h.log.Warn("invalid event_id format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	// Get existing event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Only allow deletion in creation stage
	if existingEvent.Stage != event.StageCreation {
		h.log.Warn("deletion attempt outside creation stage", "event_id", eventID, "current_stage", existingEvent.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Event can only be deleted during creation stage",
			"code":          "INVALID_DELETE_STAGE",
			"current_stage": existingEvent.Stage.String(),
		})
		return
	}

	// TODO: Check if user is event owner or admin
	// userID := c.GetString("user_id")
	// user, err := h.userRepo.GetByID(userID)
	// if err != nil || (!user.HasRole(participant.RoleAdmin) && existingEvent.AuthorID.String() != userID) {
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Only event owner or admin can delete the event",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	// Delete the event - since there's no Delete method, we'll comment this out for now
	// TODO: Add Delete method to EventRepository interface
	// if err := h.eventRepo.Delete(eventID); err != nil {
	//     h.log.Error("failed to delete event", "event_id", eventID, "error", err)
	//     c.JSON(http.StatusInternalServerError, gin.H{
	//         "error": "Failed to delete event",
	//         "code":  "DB_DELETE_ERROR",
	//     })
	//     return
	// }

	// For now, return an error indicating this feature is not implemented
	h.log.Warn("event delete feature not implemented", "event_id", eventID)
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Event delete feature is not yet implemented",
		"code":    "NOT_IMPLEMENTED",
		"details": "EventRepository.Delete method needs to be added",
	})
}

// RemoveParticipant handles DELETE /api/events/{event_id}/participants/{participant_id}
func (h *EventHandler) RemoveParticipant(c *gin.Context) {
	eventID := c.Param("event_id")
	participantID := c.Param("participant_id")

	h.log.Debug("removing participant", "event_id", eventID, "participant_id", participantID)

	// Validate required parameters
	if eventID == "" || participantID == "" {
		h.log.Warn("missing required parameters", "event_id", eventID, "participant_id", participantID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id and participant_id are required",
			"code":  "MISSING_PARAMETERS",
		})
		return
	}

	// Validate UUID formats
	if _, err := uuid.Parse(eventID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event_id format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	if _, err := uuid.Parse(participantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid participant_id format",
			"code":  "INVALID_PARTICIPANT_ID",
		})
		return
	}

	// Get existing event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Only allow removal during registration stage
	if existingEvent.Stage != event.StageRegistration {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Participants can only be removed during registration stage",
			"code":          "INVALID_REMOVAL_STAGE",
			"current_stage": existingEvent.Stage.String(),
		})
		return
	}

	// TODO: Check permissions - only admin or event owner can remove participants
	// userID := c.GetString("user_id")
	// user, err := h.userRepo.GetByID(userID)
	// if err != nil || (!user.HasRole(participant.RoleAdmin) && existingEvent.AuthorID.String() != userID) {
	//     c.JSON(http.StatusForbidden, gin.H{
	//         "error": "Insufficient permissions to remove participant",
	//         "code":  "INSUFFICIENT_PERMISSIONS",
	//     })
	//     return
	// }

	// Verify participant exists and is registered for this event
	participantEvents, err := h.eventRepo.GetByParticipant(participantID)
	isRegistered := false
	if err == nil {
		for _, evt := range participantEvents {
			if evt.ID.String() == eventID {
				isRegistered = true
				break
			}
		}
	}

	if !isRegistered {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Participant is not registered for this event",
			"code":  "NOT_REGISTERED",
		})
		return
	}

	// Remove participant from event
	if err := h.eventRepo.RemoveParticipant(eventID, participantID); err != nil {
		h.log.Error("failed to remove participant", "event_id", eventID, "participant_id", participantID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove participant",
			"code":  "REMOVAL_ERROR",
		})
		return
	}

	h.log.Info("participant removed successfully", "event_id", eventID, "participant_id", participantID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Participant removed successfully",
		"code":    "PARTICIPANT_REMOVED",
	})
}
