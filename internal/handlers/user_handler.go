package handlers

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

type UserHandler struct {
	userRepo postgres.UserRepository
	config   *config.Config
	log      *log.Logger
}

func NewUserHandler(userRepo postgres.UserRepository, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		config:   cfg,
		log:      logger.Handler("user"),
	}
}

type CreateUserRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	LastName string `json:"lastname,omitempty"`
	Email    string `json:"email" binding:"required,email"`
}

type AuthenticateUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name,omitempty"`
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
		h.log.Info("user already exists", "email", req.Email)
		// Return existing user
		c.JSON(http.StatusOK, gin.H{
			"message": "User already exists",
			"user": gin.H{
				"id":         existingUser.ID.String(),
				"name":       existingUser.Name,
				"lastname":   existingUser.LastName,
				"email":      existingUser.Email,
				"role":       existingUser.Role.String(),
				"created_at": existingUser.CreatedAt,
			},
		})
		return
	}

	// Create new user
	user := &participant.User{
		Name:     req.Name,
		LastName: req.LastName,
		Email:    req.Email,
		Role:     participant.RoleParticipant,
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

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
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
	if err == nil && existingUser != nil {
		h.log.Info("user authenticated", "email", req.Email)
		c.JSON(http.StatusOK, gin.H{
			"message": "User authenticated successfully",
			"user": gin.H{
				"id":         existingUser.ID.String(),
				"name":       existingUser.Name,
				"lastname":   existingUser.LastName,
				"email":      existingUser.Email,
				"role":       existingUser.Role.String(),
				"created_at": existingUser.CreatedAt,
			},
		})
		return
	}

	// User doesn't exist, create new one if name is provided
	if req.Name == "" {
		h.log.Warn("user not found and no name provided", "email", req.Email)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found and name is required for registration",
			"code":  "USER_NOT_FOUND",
		})
		return
	}

	// Create new user
	user := &participant.User{
		Name:  req.Name,
		Email: req.Email,
		Role:  participant.RoleParticipant,
	}

	err = h.userRepo.Create(user)
	if err != nil {
		h.log.Error("failed to create user during authentication", "error", err, "email", req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
			"code":  "CREATION_ERROR",
		})
		return
	}

	h.log.Info("new user created during authentication", "id", user.ID, "email", user.Email)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created and authenticated successfully",
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
