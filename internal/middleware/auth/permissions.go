package auth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

// RequireRole is a middleware that checks if the user has one of the required roles
func RequireRole(roles ...participant.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, err := GetUserRoleFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User role not found in context",
			})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, role := range roles {
			if userRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(403, gin.H{
				"error": "FORBIDDEN",
				"message": fmt.Sprintf("This action requires one of the following roles: %v", roles),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireEventOwner is a middleware that checks if the user is the owner of the event
// or has admin privileges
func RequireEventOwner(eventRepo postgres.EventRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context
		userID, err := GetUserIDFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User ID not found in context",
			})
			c.Abort()
			return
		}

		userRole, err := GetUserRoleFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User role not found in context",
			})
			c.Abort()
			return
		}

		// Admins can do anything
		if userRole == participant.RoleAdmin {
			c.Next()
			return
		}

		// Get event ID from URL parameter
		eventIDStr := c.Param("event_id")
		if eventIDStr == "" {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Event ID is required",
			})
			c.Abort()
			return
		}

		eventID, err := uuid.Parse(eventIDStr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Invalid event ID format",
			})
			c.Abort()
			return
		}

		// Get event from database
		event, err := eventRepo.GetByID(eventID.String())
		if err != nil {
			c.JSON(404, gin.H{
				"error": "NOT_FOUND",
				"message": "Event not found",
			})
			c.Abort()
			return
		}

		// Check if user is the event author
		if event.AuthorID != userID {
			c.JSON(403, gin.H{
				"error": "FORBIDDEN",
				"message": "Only the event creator or an admin can perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireEventOwnerOrOrganizer is a middleware that checks if the user is the owner of the event
// or has admin/organizer privileges
func RequireEventOwnerOrOrganizer(eventRepo postgres.EventRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context
		userID, err := GetUserIDFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User ID not found in context",
			})
			c.Abort()
			return
		}

		userRole, err := GetUserRoleFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User role not found in context",
			})
			c.Abort()
			return
		}

		// Admins and organizers can manage events
		if userRole == participant.RoleAdmin || userRole == participant.RoleOrganizer {
			c.Next()
			return
		}

		// Get event ID from URL parameter
		eventIDStr := c.Param("event_id")
		if eventIDStr == "" {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Event ID is required",
			})
			c.Abort()
			return
		}

		eventID, err := uuid.Parse(eventIDStr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Invalid event ID format",
			})
			c.Abort()
			return
		}

		// Get event from database
		event, err := eventRepo.GetByID(eventID.String())
		if err != nil {
			c.JSON(404, gin.H{
				"error": "NOT_FOUND",
				"message": "Event not found",
			})
			c.Abort()
			return
		}

		// Check if user is the event author
		if event.AuthorID != userID {
			c.JSON(403, gin.H{
				"error": "FORBIDDEN",
				"message": "Only the event creator, organizers, or admins can perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireParticipantOrOwner is a middleware that checks if the user is the participant
// specified in the URL or the event owner/admin
func RequireParticipantOrOwner(eventRepo postgres.EventRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context
		userID, err := GetUserIDFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User ID not found in context",
			})
			c.Abort()
			return
		}

		userRole, err := GetUserRoleFromContext(c)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "User role not found in context",
			})
			c.Abort()
			return
		}

		// Admins can do anything
		if userRole == participant.RoleAdmin {
			c.Next()
			return
		}

		// Get participant ID from URL parameter
		participantIDStr := c.Param("participant_id")
		if participantIDStr == "" {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Participant ID is required",
			})
			c.Abort()
			return
		}

		participantID, err := uuid.Parse(participantIDStr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Invalid participant ID format",
			})
			c.Abort()
			return
		}

		// If user is the participant themselves, allow
		if userID == participantID {
			c.Next()
			return
		}

		// Otherwise, check if user is event owner
		eventIDStr := c.Param("event_id")
		if eventIDStr == "" {
			c.JSON(403, gin.H{
				"error": "FORBIDDEN",
				"message": "You can only access your own resources",
			})
			c.Abort()
			return
		}

		eventID, err := uuid.Parse(eventIDStr)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "BAD_REQUEST",
				"message": "Invalid event ID format",
			})
			c.Abort()
			return
		}

		// Get event from database
		event, err := eventRepo.GetByID(eventID.String())
		if err != nil {
			c.JSON(404, gin.H{
				"error": "NOT_FOUND",
				"message": "Event not found",
			})
			c.Abort()
			return
		}

		// Check if user is the event author
		if event.AuthorID != userID {
			c.JSON(403, gin.H{
				"error": "FORBIDDEN",
				"message": "You can only access your own resources or resources from events you created",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
