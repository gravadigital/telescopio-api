package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/attachment"
	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type AttachmentHandler struct {
	attachmentRepo postgres.AttachmentRepository
	eventRepo      postgres.EventRepository
	userRepo       postgres.UserRepository
	config         *config.Config
	log            *log.Logger
}

func NewAttachmentHandler(attachmentRepo postgres.AttachmentRepository, eventRepo postgres.EventRepository, userRepo postgres.UserRepository, cfg *config.Config) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentRepo: attachmentRepo,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		config:         cfg,
		log:            logger.Handler("attachment"),
	}
}

// UploadAttachment handles POST /api/events/{event_id}/participant/{participant_id}/attachment
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	eventID := c.Param("event_id")
	participantID := c.Param("participant_id")

	h.log.Debug("processing attachment upload", "event_id", eventID, "participant_id", participantID)

	// Validate required parameters
	if eventID == "" || participantID == "" {
		h.log.Warn("missing required parameters", "event_id", eventID, "participant_id", participantID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event_id and participant_id are required",
			"code":  "MISSING_PARAMETERS",
		})
		return
	}

	// Validate UUID formats early
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		h.log.Warn("invalid event ID format", "event_id", eventID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID format",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	participantUUID, err := uuid.Parse(participantID)
	if err != nil {
		h.log.Warn("invalid participant ID format", "participant_id", participantID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid participant ID format",
			"code":  "INVALID_PARTICIPANT_ID",
		})
		return
	}

	// Check if event exists and validate its state
	eventEntity, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.log.Error("event not found", "event_id", eventID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Validate event stage - only allow uploads during submission stage
	if eventEntity.Stage != event.StageSubmission {
		h.log.Warn("upload attempt outside submission stage", "event_id", eventID, "current_stage", eventEntity.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "File uploads are only allowed during the submission stage",
			"code":          "INVALID_EVENT_STAGE",
			"current_stage": eventEntity.Stage,
		})
		return
	}

	// Check if participant exists
	participant, err := h.userRepo.GetByID(participantID)
	if err != nil {
		h.log.Error("participant not found", "participant_id", participantID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Participant not found",
			"code":  "PARTICIPANT_NOT_FOUND",
		})
		return
	}

	// Check if participant is registered for this event
	participantEvents, err := h.eventRepo.GetByParticipant(participantID)
	isParticipant := false
	if err == nil {
		for _, evt := range participantEvents {
			if evt.ID == eventUUID {
				isParticipant = true
				break
			}
		}
	}

	if !isParticipant {
		h.log.Warn("unauthorized upload attempt", "event_id", eventID, "participant_id", participantID)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Participant is not registered for this event",
			"code":  "NOT_REGISTERED",
		})
		return
	}

	// Check if participant already has an attachment for this event
	existingAttachments, err := h.attachmentRepo.GetByParticipantID(participantID)
	if err == nil {
		for _, att := range existingAttachments {
			if att.EventID == eventUUID {
				h.log.Warn("duplicate attachment attempt", "event_id", eventID, "participant_id", participantID, "existing_attachment", att.ID)
				c.JSON(http.StatusConflict, gin.H{
					"error":                  "Participant already has an attachment for this event",
					"code":                   "DUPLICATE_ATTACHMENT",
					"existing_attachment_id": att.ID.String(),
				})
				return
			}
		}
	}

	// Get the file from the form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.log.Warn("no file provided in request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No file provided",
			"code":    "NO_FILE",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	h.log.Debug("file received", "filename", header.Filename, "size", header.Size, "content_type", header.Header.Get("Content-Type"))

	// Validate file size using configuration
	if header.Size > h.config.Upload.MaxFileSize {
		h.log.Warn("file size exceeds limit", "filename", header.Filename, "size", header.Size, "max_size", h.config.Upload.MaxFileSize)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "File size exceeds limit",
			"code":          "FILE_TOO_LARGE",
			"max_size":      fmt.Sprintf("%d bytes", h.config.Upload.MaxFileSize),
			"received_size": header.Size,
		})
		return
	}

	// Enhanced file type validation
	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]string{
		"image/jpeg":         "JPEG Image",
		"image/png":          "PNG Image",
		"image/gif":          "GIF Image",
		"application/pdf":    "PDF Document",
		"text/plain":         "Text Document",
		"application/msword": "Word Document",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "Word Document (DOCX)",
	}

	if _, isAllowed := allowedTypes[contentType]; !isAllowed {
		h.log.Warn("file type not allowed", "filename", header.Filename, "content_type", contentType)
		allowedList := make([]string, 0, len(allowedTypes))
		for _, desc := range allowedTypes {
			allowedList = append(allowedList, desc)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "File type not allowed",
			"code":          "INVALID_FILE_TYPE",
			"received_type": contentType,
			"allowed_types": allowedList,
		})
		return
	}

	// Security: Validate filename to prevent path traversal
	cleanFilename := filepath.Base(header.Filename)
	if cleanFilename != header.Filename || strings.Contains(cleanFilename, "..") {
		h.log.Warn("suspicious filename detected", "original", header.Filename, "cleaned", cleanFilename)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
			"code":  "INVALID_FILENAME",
		})
		return
	}

	// Generate secure unique filename
	ext := filepath.Ext(cleanFilename)
	secureFilename := fmt.Sprintf("%s_%s_%d%s", eventID, participantID, time.Now().Unix(), ext)

	// Create uploads directory if it doesn't exist
	uploadsDir := h.config.Upload.Dir
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		h.log.Error("failed to create uploads directory", "dir", uploadsDir, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create uploads directory",
			"code":  "UPLOAD_DIR_ERROR",
		})
		return
	}

	// Save file to disk
	filePath := filepath.Join(uploadsDir, secureFilename)
	dst, err := os.Create(filePath)
	if err != nil {
		h.log.Error("failed to create file", "path", filePath, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create file",
			"code":  "FILE_CREATE_ERROR",
		})
		return
	}
	defer dst.Close()

	bytesWritten, err := io.Copy(dst, file)
	if err != nil {
		h.log.Error("failed to save file", "path", filePath, "error", err)
		// Clean up the partially written file
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
			"code":  "FILE_SAVE_ERROR",
		})
		return
	}

	// Verify written size matches expected size
	if bytesWritten != header.Size {
		h.log.Error("file size mismatch", "expected", header.Size, "written", bytesWritten)
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "File size mismatch during upload",
			"code":  "SIZE_MISMATCH",
		})
		return
	}

	// Create attachment record
	newAttachment := attachment.NewAttachment(
		eventUUID,
		participantUUID,
		secureFilename,
		cleanFilename,
		filePath,
		contentType,
		header.Size,
	)

	if err := h.attachmentRepo.Create(newAttachment); err != nil {
		h.log.Error("failed to save attachment metadata", "attachment_id", newAttachment.ID, "error", err)
		// Clean up the file if database operation fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save attachment metadata",
			"code":  "DB_SAVE_ERROR",
		})
		return
	}

	h.log.Info("attachment uploaded successfully",
		"attachment_id", newAttachment.ID,
		"event_id", eventID,
		"participant_id", participantID,
		"filename", cleanFilename,
		"size", header.Size)

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"id":          newAttachment.ID.String(),
			"filename":    newAttachment.OriginalName,
			"size":        newAttachment.FileSize,
			"mime_type":   newAttachment.MimeType,
			"participant": participant.Name,
			"uploaded_at": newAttachment.UploadedAt,
		},
		"message": "File uploaded successfully",
		"code":    "UPLOAD_SUCCESS",
	})
}

// GetAttachment handles GET /api/attachments/{attachment_id}
func (h *AttachmentHandler) GetAttachment(c *gin.Context) {
	attachmentID := c.Param("attachment_id")

	h.log.Debug("retrieving attachment", "attachment_id", attachmentID)

	attachment, err := h.attachmentRepo.GetByID(attachmentID)
	if err != nil {
		h.log.Error("attachment not found", "attachment_id", attachmentID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Attachment not found",
			"code":  "ATTACHMENT_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":             attachment.ID.String(),
			"filename":       attachment.OriginalName,
			"size":           attachment.FileSize,
			"mime_type":      attachment.MimeType,
			"vote_count":     attachment.VoteCount,
			"uploaded_at":    attachment.UploadedAt,
			"event_id":       attachment.EventID.String(),
			"participant_id": attachment.ParticipantID.String(),
		},
	})
}

// GetEventAttachments handles GET /api/events/{event_id}/attachments
func (h *AttachmentHandler) GetEventAttachments(c *gin.Context) {
	eventID := c.Param("event_id")

	h.log.Debug("retrieving event attachments", "event_id", eventID)

	attachments, err := h.attachmentRepo.GetByEventID(eventID)
	if err != nil {
		h.log.Error("failed to retrieve event attachments", "event_id", eventID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve attachments",
			"code":  "RETRIEVAL_ERROR",
		})
		return
	}

	// Transform to response format
	attachmentData := make([]gin.H, len(attachments))
	for i, att := range attachments {
		attachmentData[i] = gin.H{
			"id":             att.ID.String(),
			"filename":       att.OriginalName,
			"size":           att.FileSize,
			"mime_type":      att.MimeType,
			"vote_count":     att.VoteCount,
			"uploaded_at":    att.UploadedAt,
			"participant_id": att.ParticipantID.String(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  attachmentData,
		"count": len(attachments),
	})
}

// DownloadAttachment handles GET /api/attachments/{attachment_id}/download
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	attachmentID := c.Param("attachment_id")

	h.log.Debug("downloading attachment", "attachment_id", attachmentID)

	attachment, err := h.attachmentRepo.GetByID(attachmentID)
	if err != nil {
		h.log.Error("attachment not found for download", "attachment_id", attachmentID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Attachment not found",
			"code":  "ATTACHMENT_NOT_FOUND",
		})
		return
	}

	// Check if file exists on disk
	if _, err := os.Stat(attachment.FilePath); os.IsNotExist(err) {
		h.log.Error("file not found on disk", "attachment_id", attachmentID, "file_path", attachment.FilePath)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found on server",
			"code":  "FILE_NOT_FOUND",
		})
		return
	}

	// Set appropriate headers for file download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.OriginalName))
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", attachment.FileSize))

	h.log.Info("serving file download", "attachment_id", attachmentID, "filename", attachment.OriginalName)
	c.File(attachment.FilePath)
}

// DeleteAttachment handles DELETE /api/attachments/{attachment_id}
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	attachmentID := c.Param("attachment_id")

	h.log.Debug("deleting attachment", "attachment_id", attachmentID)

	// Get attachment details first
	attachment, err := h.attachmentRepo.GetByID(attachmentID)
	if err != nil {
		h.log.Error("attachment not found for deletion", "attachment_id", attachmentID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Attachment not found",
			"code":  "ATTACHMENT_NOT_FOUND",
		})
		return
	}

	// TODO: Add authorization check - only allow deletion by attachment owner or admin
	// participantID := c.GetString("user_id") // From JWT middleware
	// if attachment.ParticipantID.String() != participantID {
	//     c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this attachment"})
	//     return
	// }

	// Check event stage - only allow deletion during submission stage
	eventEntity, err := h.eventRepo.GetByID(attachment.EventID.String())
	if err == nil && eventEntity.Stage != event.StageSubmission {
		h.log.Warn("deletion attempt outside submission stage", "attachment_id", attachmentID, "event_stage", eventEntity.Stage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Attachments can only be deleted during the submission stage",
			"code":  "INVALID_EVENT_STAGE",
		})
		return
	}

	// Delete from database
	if err := h.attachmentRepo.Delete(attachmentID); err != nil {
		h.log.Error("failed to delete attachment from database", "attachment_id", attachmentID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete attachment",
			"code":  "DB_DELETE_ERROR",
		})
		return
	}

	// Delete file from disk
	if err := os.Remove(attachment.FilePath); err != nil {
		h.log.Warn("failed to delete file from disk", "attachment_id", attachmentID, "file_path", attachment.FilePath, "error", err)
		// Don't fail the request if file deletion fails, as DB record is already deleted
	}

	h.log.Info("attachment deleted successfully", "attachment_id", attachmentID, "filename", attachment.OriginalName)

	c.JSON(http.StatusOK, gin.H{
		"message": "Attachment deleted successfully",
		"code":    "DELETE_SUCCESS",
	})
}
