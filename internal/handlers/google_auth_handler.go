package handlers

import (
	"errors"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/middleware/auth"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)


// GoogleAuthHandler handles Google OAuth authentication endpoints.
type GoogleAuthHandler struct {
	userRepo postgres.UserRepository
	cfg      *config.Config
	log      *log.Logger
}

// NewGoogleAuthHandler creates a new GoogleAuthHandler.
func NewGoogleAuthHandler(userRepo postgres.UserRepository, cfg *config.Config) *GoogleAuthHandler {
	return &GoogleAuthHandler{
		userRepo: userRepo,
		cfg:      cfg,
		log:      logger.Handler("google_auth"),
	}
}

// VerifyGoogleTokenRequest is the request body for POST /auth/google/verify.
type VerifyGoogleTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// RegisterGoogleUserRequest is the request body for POST /auth/google/register.
type RegisterGoogleUserRequest struct {
	Token    string `json:"token" binding:"required"`
	Username string `json:"username" binding:"required,min=2,max=100"`
}

// VerifyGoogleToken handles POST /api/v1/auth/google/verify.
// It validates the Google id_token and determines if the user is new or existing.
func (h *GoogleAuthHandler) VerifyGoogleToken(c *gin.Context) {
	h.log.Debug("received google token verification request")

	var req VerifyGoogleTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	profile, err := verifyGoogleToken(req.Token, h.cfg.Google.ClientID)
	if err != nil {
		if errors.Is(err, ErrInvalidGoogleToken) {
			h.log.Warn("invalid google token received", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired Google token",
				"code":  "INVALID_GOOGLE_TOKEN",
			})
			return
		}
		h.log.Error("google API error during token verification", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to contact Google API",
			"code":  "GOOGLE_API_ERROR",
		})
		return
	}

	resolution, err := resolveUser(profile, h.userRepo)
	if err != nil {
		h.log.Error("failed to resolve user", "error", err, "email", profile.Email)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to resolve user",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	if resolution.Status == "existing_user" {
		token, err := auth.GenerateToken(resolution.User.ID, resolution.User.Email, resolution.User.Role)
		if err != nil {
			h.log.Error("failed to generate JWT", "error", err, "user_id", resolution.User.ID)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
				"code":  "INTERNAL_ERROR",
			})
			return
		}

		h.log.Info("existing google user authenticated", "user_id", resolution.User.ID, "email", resolution.User.Email)
		c.JSON(http.StatusOK, gin.H{
			"status": "existing_user",
			"token":  token,
			"user": gin.H{
				"id":       resolution.User.ID,
				"email":    resolution.User.Email,
				"username": resolution.User.Name,
			},
		})
		return
	}

	h.log.Info("new google user detected", "email", profile.Email)
	c.JSON(http.StatusOK, gin.H{
		"status":       "new_user",
		"google_token": req.Token,
		"profile": gin.H{
			"email":          profile.Email,
			"suggested_name": profile.Name,
		},
	})
}

// RegisterGoogleUser handles POST /api/v1/auth/google/register.
// It creates a new user account for a Google-authenticated user with a chosen username.
func (h *GoogleAuthHandler) RegisterGoogleUser(c *gin.Context) {
	h.log.Debug("received google user registration request")

	var req RegisterGoogleUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Re-validate the Google token to prevent account creation with intercepted tokens
	profile, err := verifyGoogleToken(req.Token, h.cfg.Google.ClientID)
	if err != nil {
		if errors.Is(err, ErrInvalidGoogleToken) {
			h.log.Warn("invalid google token in registration", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired Google token",
				"code":  "INVALID_GOOGLE_TOKEN",
			})
			return
		}
		h.log.Error("google API error during registration", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to contact Google API",
			"code":  "GOOGLE_API_ERROR",
		})
		return
	}

	// Check username uniqueness
	usernameExists, err := h.userRepo.UsernameExists(req.Username)
	if err != nil {
		h.log.Error("failed to check username existence", "error", err, "username", req.Username)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate username",
			"code":  "INTERNAL_ERROR",
		})
		return
	}
	if usernameExists {
		h.log.Warn("username already in use", "username", req.Username)
		c.JSON(http.StatusConflict, gin.H{
			"error": "Username is already in use",
			"code":  "USERNAME_ALREADY_EXISTS",
		})
		return
	}

	// Create the new user with no password (OAuth user)
	googleID := profile.GoogleID
	newUser := &participant.User{
		Name:     req.Username,
		Email:    profile.Email,
		GoogleID: &googleID,
		Role:     participant.RoleParticipant,
	}

	if err := h.userRepo.Create(newUser); err != nil {
		h.log.Error("failed to create google user", "error", err, "email", profile.Email)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	token, err := auth.GenerateToken(newUser.ID, newUser.Email, newUser.Role)
	if err != nil {
		h.log.Error("failed to generate JWT after registration", "error", err, "user_id", newUser.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	h.log.Info("google user registered successfully", "user_id", newUser.ID, "email", newUser.Email, "username", newUser.Name)
	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"id":       newUser.ID,
			"email":    newUser.Email,
			"username": newUser.Name,
		},
	})
}

