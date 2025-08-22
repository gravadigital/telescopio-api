package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type AttachmentHandler struct {
	attachmentRepo postgres.AttachmentRepository
	eventRepo      postgres.EventRepository
	userRepo       postgres.UserRepository
}

func NewAttachmentHandler(attachmentRepo postgres.AttachmentRepository, eventRepo postgres.EventRepository, userRepo postgres.UserRepository) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentRepo: attachmentRepo,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
	}
}

// UploadAttachment handles POST /api/events/{event_id}/participant/{participant_id}/attachment
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	eventID := c.Param("event_id")
	participantID := c.Param("participant_id")

	if eventID == "" || participantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id and participant_id are required",
		})
		return
	}

	// Check if event exists
	event, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
		})
		return
	}

	// Check if participant exists
	participant, err := h.userRepo.GetByID(participantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Participant not found",
		})
		return
	}

	// Check if participant is registered for this event
	if !event.IsParticipant(participantID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Participant is not registered for this event",
		})
		return
	}

	// Get the file from the form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No file provided",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file size (10MB limit)
	const maxFileSize = 10 << 20 // 10MB
	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File size exceeds 10MB limit",
		})
		return
	}

	// Validate file type (allow common document and image formats)
	allowedTypes := map[string]bool{
		"image/jpeg":         true,
		"image/png":          true,
		"image/gif":          true,
		"application/pdf":    true,
		"text/plain":         true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "File type not allowed",
			"allowed_types": []string{"JPEG", "PNG", "GIF", "PDF", "TXT", "DOC", "DOCX"},
		})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s_%s_%d%s", eventID, participantID, time.Now().Unix(), ext)

	// Create uploads directory if it doesn't exist
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create uploads directory",
		})
		return
	}

	// Save file to disk
	filePath := filepath.Join(uploadsDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create file",
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}

	// Create attachment record
	newAttachment := attachment.NewAttachment(
		eventID,
		participantID,
		filename,
		header.Filename,
		filePath,
		contentType,
		header.Size,
	)

	if err := h.attachmentRepo.Create(newAttachment); err != nil {
		// Clean up the file if database operation fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save attachment metadata",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          newAttachment.ID,
		"filename":    newAttachment.OriginalName,
		"size":        newAttachment.FileSize,
		"participant": participant.Name,
		"uploaded_at": newAttachment.UploadedAt,
		"message":     "File uploaded successfully",
	})
}
