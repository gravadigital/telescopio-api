package handlers

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/email"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/middleware/auth"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type UserHandler struct {
	userRepo     postgres.UserRepository
	eventRepo    postgres.EventRepository
	emailService *email.EmailService
	config       *config.Config
	log          *log.Logger
}

func NewUserHandler(userRepo postgres.UserRepository, eventRepo postgres.EventRepository, emailService *email.EmailService, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userRepo:     userRepo,
		eventRepo:    eventRepo,
		emailService: emailService,
		config:       cfg,
		log:          logger.Handler("user"),
	}
}

type CreateUserRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	LastName string `json:"lastname,omitempty"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type AuthenticateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// CreateUser handles POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	h.log.Debug("received create user request")

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("invalid request payload for create user", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Check if user already exists
	existingUser, err := h.userRepo.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		h.log.Warn("attempt to register with existing email", "email", req.Email)
		c.JSON(http.StatusConflict, gin.H{
			"error": "User with this email already exists",
			"code":  "EMAIL_ALREADY_EXISTS",
		})
		return
	}

	// Create new user
	user := &participant.User{
		Name:     req.Name,
		LastName: req.LastName,
		Email:    req.Email,
		Role:     participant.RoleParticipant, // All users are participants by default
	}

	// Hash password
	if err := user.SetPassword(req.Password); err != nil {
		h.log.Error("failed to hash password", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "INVALID_PASSWORD",
		})
		return
	}

	err = h.userRepo.Create(user)
	if err != nil {
		h.log.Error("failed to create user", "error", err, "email", req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
			"code":  "CREATION_ERROR",
		})
		return
	}

	h.log.Info("user created successfully", "id", user.ID, "email", user.Email)

	// Generate JWT token for new user
	token, err := auth.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		h.log.Error("failed to generate token for new user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
			"code":  "TOKEN_GENERATION_ERROR",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"token":   token,
		"user": gin.H{
			"id":         user.ID.String(),
			"name":       user.Name,
			"lastname":   user.LastName,
			"email":      user.Email,
			"role":       user.Role.String(),
			"created_at": user.CreatedAt,
		},
	})
}

// AuthenticateUser handles POST /api/v1/users/authenticate
func (h *UserHandler) AuthenticateUser(c *gin.Context) {
	h.log.Debug("received authenticate user request")

	var req AuthenticateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("invalid request payload for authenticate user", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	// Try to find existing user
	existingUser, err := h.userRepo.GetByEmail(req.Email)
	if err != nil || existingUser == nil {
		h.log.Warn("authentication failed: user not found", "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
			"code":  "INVALID_CREDENTIALS",
		})
		return
	}

	// Check if the user has a password (OAuth accounts don't)
	if existingUser.PasswordHash == nil {
		h.log.Warn("authentication failed: user has no password (OAuth account)", "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "This account was created with Google. Please use the 'Continue with Google' button to sign in.",
			"code":  "OAUTH_ACCOUNT_NO_PASSWORD",
		})
		return
	}

	// Verify password
	if !existingUser.CheckPassword(req.Password) {
		h.log.Warn("authentication failed: invalid password", "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
			"code":  "INVALID_CREDENTIALS",
		})
		return
	}

	h.log.Info("user authenticated successfully", "email", req.Email, "user_id", existingUser.ID)

	// Generate JWT token
	token, err := auth.GenerateToken(existingUser.ID, existingUser.Email, existingUser.Role)
	if err != nil {
		h.log.Error("failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
			"code":  "TOKEN_GENERATION_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User authenticated successfully",
		"token":   token,
		"user": gin.H{
			"id":         existingUser.ID.String(),
			"name":       existingUser.Name,
			"lastname":   existingUser.LastName,
			"email":      existingUser.Email,
			"role":       existingUser.Role.String(),
			"created_at": existingUser.CreatedAt,
		},
	})
}

// GetUser handles GET /api/v1/users/:user_id
func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("user_id")

	user, err := h.userRepo.GetByID(userIDStr)
	if err != nil {
		h.log.Error("failed to get user", "error", err, "user_id", userIDStr)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID.String(),
			"name":       user.Name,
			"lastname":   user.LastName,
			"email":      user.Email,
			"role":       user.Role.String(),
			"created_at": user.CreatedAt,
		},
	})
}

// GetUserEvents handles GET /api/v1/users/:user_id/events
// Returns all events where the user is a participant (not creator)
func (h *UserHandler) GetUserEvents(c *gin.Context) {
	requestedUserID := c.Param("user_id")
	h.log.Debug("received get user events request", "user_id", requestedUserID)

	// Get authenticated user from JWT
	authenticatedUserID, exists := c.Get("user_id")
	if !exists {
		h.log.Warn("no user_id in context (missing authentication)")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: No valid authentication token",
			"code":  "NO_AUTH_TOKEN",
		})
		return
	}

	// Verify user can only access their own events
	if authenticatedUserID.(string) != requestedUserID {
		h.log.Warn("user attempting to access another user's events",
			"authenticated_user", authenticatedUserID,
			"requested_user", requestedUserID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: You can only view your own events",
			"code":  "UNAUTHORIZED_ACCESS",
		})
		return
	}

	// Get events from repository
	events, err := h.eventRepo.GetUserParticipatingEvents(requestedUserID)
	if err != nil {
		h.log.Error("failed to retrieve user events", "user_id", requestedUserID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user events",
			"code":  "RETRIEVAL_ERROR",
		})
		return
	}

	// Transform events to response format
	response := make([]gin.H, 0, len(events))
	for _, evt := range events {
		response = append(response, gin.H{
			"id":          evt.ID.String(),
			"name":        evt.Name,
			"title":       evt.Name, // For frontend compatibility
			"description": evt.Description,
			"stage":       evt.Stage.String(),
			"start_date":  evt.StartDate,
			"date":        evt.StartDate, // For frontend compatibility
			"end_date":    evt.EndDate,
			"organizer":   evt.Organizer,
			"author_id":   evt.AuthorID.String(),
			"creator_id":  evt.AuthorID.String(), // For frontend compatibility
			"created_at":  evt.CreatedAt,
			"updated_at":  evt.UpdatedAt,
		})
	}

	h.log.Info("user events retrieved successfully", "user_id", requestedUserID, "count", len(events))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword handles POST /api/v1/users/forgot-password
// Generates a reset token, saves it to the user, and sends a reset email.
// Always returns 200 to avoid leaking whether an email is registered.
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload",
			"code":  "INVALID_PAYLOAD",
		})
		return
	}

	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil || user == nil {
		// Don't reveal whether the email exists
		h.log.Debug("forgot-password: email not found, returning 200 anyway", "email", req.Email)
		c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent."})
		return
	}

	token, err := user.GeneratePasswordResetToken()
	if err != nil {
		h.log.Error("failed to generate password reset token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate reset token",
			"code":  "TOKEN_GENERATION_ERROR",
		})
		return
	}

	if err := h.userRepo.SavePasswordResetToken(user); err != nil {
		h.log.Error("failed to save password reset token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save reset token",
			"code":  "DB_ERROR",
		})
		return
	}

	resetURL := h.config.Server.FrontendURL + "/reset-password?token=" + token
	if err := h.emailService.SendPasswordResetEmail(user.Email, resetURL); err != nil {
		h.log.Error("failed to send password reset email", "email", user.Email, "error", err)
	} else {
		h.log.Info("password reset email sent", "email", user.Email)
	}
	c.JSON(http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent."})
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ResetPassword handles POST /api/v1/users/reset-password
// Validates the token and sets the new password.
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"code":    "INVALID_PAYLOAD",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userRepo.GetByResetToken(req.Token)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired reset token",
			"code":  "INVALID_RESET_TOKEN",
		})
		return
	}

	if !user.IsPasswordResetTokenValid() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Reset token has expired",
			"code":  "EXPIRED_RESET_TOKEN",
		})
		return
	}

	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "INVALID_PASSWORD",
		})
		return
	}

	user.ClearPasswordResetToken()

	if err := h.userRepo.ClearPasswordResetToken(user); err != nil {
		h.log.Error("failed to save new password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update password",
			"code":  "DB_ERROR",
		})
		return
	}

	h.log.Info("password reset successfully", "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully."})
}
