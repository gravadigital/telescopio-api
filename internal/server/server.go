package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/handlers"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	config     *config.Config
	db         *gorm.DB
}

// New creates a new server instance
func New(cfg *config.Config, db *gorm.DB) *Server {
	return &Server{
		config: cfg,
		db:     db,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := s.setupRouter()

	s.httpServer = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: router,

		// Timeouts seguros según estándares de Go
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Get().Info("Starting HTTP server", "port", s.config.Port)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	logger.Get().Info("Shutting down HTTP server...")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// setupRouter configures the HTTP router with middleware and routes
func (s *Server) setupRouter() *gin.Engine {
	// Configurar Gin
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware básico
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Inicializar repositorios
	eventRepo := postgres.NewEventRepository(s.db)
	userRepo := postgres.NewUserRepository(s.db)
	attachmentRepo := postgres.NewAttachmentRepository(s.db)
	voteRepo := postgres.NewVoteRepository(s.db)

	// Inicializar handlers
	eventHandler := handlers.NewEventHandler(eventRepo, userRepo)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentRepo, eventRepo, userRepo)
	voteHandler := handlers.NewVoteHandler(voteRepo, eventRepo, attachmentRepo, userRepo)

	// Health check
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Telescopio API is running",
			"status":  "healthy",
		})
	})

	// API routes
	s.setupAPIRoutes(router, eventHandler, attachmentHandler, voteHandler)

	return router
}

// setupAPIRoutes configures all API routes
func (s *Server) setupAPIRoutes(
	router *gin.Engine,
	eventHandler *handlers.EventHandler,
	attachmentHandler *handlers.AttachmentHandler,
	voteHandler *handlers.VoteHandler,
) {
	api := router.Group("/api")
	{
		// Event routes (solo las que existen actualmente)
		events := api.Group("/events")
		{
			events.GET("", eventHandler.GetAllEvents)
			events.POST("", eventHandler.CreateEvent)
			events.POST("/:id/register", eventHandler.RegisterParticipant)
			events.GET("/:id/results", voteHandler.GetEventResults)
		}

		// Attachment routes (solo las que existen)
		attachments := api.Group("/attachments")
		{
			attachments.POST("/upload", attachmentHandler.UploadAttachment)
		}

		// Vote routes (solo las que existen)
		votes := api.Group("/votes")
		{
			votes.POST("", voteHandler.SubmitVote)
		}
	}
}
