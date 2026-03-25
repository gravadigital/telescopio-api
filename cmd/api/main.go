package main

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/email"
	"github.com/gravadigital/telescopio-api/internal/handlers"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/middleware/auth"
	"github.com/gravadigital/telescopio-api/internal/middleware/events"
	"github.com/gravadigital/telescopio-api/internal/storage"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

func main() {
	cfg := config.Load()

	logLevel := "info"
	if cfg.Server.GinMode == "debug" {
		logLevel = "debug"
	}
	logger.Initialize(logLevel)
	log := logger.Get()

	gin.SetMode(cfg.Server.GinMode)

	db, err := postgres.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}

	if err := postgres.AutoMigrate(db); err != nil {
		log.Fatal("Failed to migrate database", "error", err)
	}

	router := gin.Default()

	corsConfig := cors.DefaultConfig()
	if cfg.CORS.AllowOrigins == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = strings.Split(cfg.CORS.AllowOrigins, ",")
	}
	corsConfig.AllowMethods = strings.Split(cfg.CORS.AllowMethods, ",")
	corsConfig.AllowHeaders = strings.Split(cfg.CORS.AllowHeaders, ",")
	router.Use(cors.New(corsConfig))

	router.Use(events.CreateEvent())

	eventRepo := postgres.NewPostgresEventRepository(db)
	userRepo := postgres.NewPostgresUserRepository(db)
	attachmentRepo := postgres.NewPostgresAttachmentRepository(db)
	voteRepo := postgres.NewPostgresVoteRepository(db)

	// Initialize file storage
	fileStorage, err := storage.NewFileStorage(cfg)
	if err != nil {
		log.Fatal("Failed to initialize file storage", "error", err)
	}
	log.Info("File storage initialized", "provider", cfg.Storage.Provider)

	emailService := email.NewEmailService(cfg)
	log.Info("Email service initialized", "enabled", cfg.Email.Enabled)

	eventHandler := handlers.NewEventHandler(eventRepo, userRepo, attachmentRepo, emailService, cfg)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentRepo, eventRepo, userRepo, fileStorage, cfg)
	userHandler := handlers.NewUserHandler(userRepo, eventRepo, emailService, cfg)
	googleAuthHandler := handlers.NewGoogleAuthHandler(userRepo, cfg)

	configRepo := postgres.NewPostgresVotingConfigurationRepository(db)
	resultsRepo := postgres.NewPostgresVotingResultsRepository(db)
	distributedVoteHandler := handlers.NewDistributedVoteHandler(voteRepo, eventRepo, attachmentRepo, userRepo, configRepo, resultsRepo, cfg)

	voteDraftRepo := postgres.NewPostgresVoteDraftRepository(db)
	voteDraftHandler := handlers.NewVoteDraftHandler(voteDraftRepo, voteRepo)

	// Test database connection
	router.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"service": "telescopio-api",
				"error":   "database connection failed",
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"service": "telescopio-api",
				"error":   "database ping failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"service":  "telescopio-api",
			"version":  "1.0.0",
			"database": "connected",
		})
	})

	api := router.Group("/api/v1")
	{
		// User management - Public endpoints (no auth required)
		users := api.Group("/users")
		{
			users.POST("", userHandler.CreateUser)                         // Register new user (returns JWT)
			users.POST("/authenticate", userHandler.AuthenticateUser)      // Login (returns JWT)
			users.POST("/forgot-password", userHandler.ForgotPassword)     // Request password reset email
			users.POST("/reset-password", userHandler.ResetPassword)       // Set new password with token
		}

		// Google OAuth - Public endpoints (no auth required)
		googleAuth := api.Group("/auth/google")
		{
			googleAuth.POST("/verify", googleAuthHandler.VerifyGoogleToken)
			googleAuth.POST("/register", googleAuthHandler.RegisterGoogleUser)
		}

	// Protected user endpoints (require authentication)
	usersProtected := api.Group("/users")
	usersProtected.Use(auth.JWTAuthMiddleware())
	{
		usersProtected.GET("/:user_id", userHandler.GetUser)
		usersProtected.GET("/:user_id/events", userHandler.GetUserEvents) // Get events where user participates
	}

		// Event management - Public endpoints (no authentication required)
		eventsPublic := api.Group("/events")
		{
			eventsPublic.GET("", eventHandler.GetAllEvents)                            // List all events
			eventsPublic.GET("/:event_id", eventHandler.GetEvent)                      // Get event details
			eventsPublic.GET("/:event_id/share", eventHandler.GetShareableEventInfo)   // Get shareable metadata
			eventsPublic.POST("/:event_id/register", eventHandler.RegisterParticipant) // Register for event (creates user if doesn't exist)
		}

		// Event management - Protected endpoints (require authentication)
		events := api.Group("/events")
		events.Use(auth.JWTAuthMiddleware())
		{
			// Create event - Any authenticated user can create events
			events.POST("", eventHandler.CreateEvent)

			// Update event stage - Only event owner or admin
			events.PATCH("/:event_id/stage",
				auth.RequireEventOwner(eventRepo),
				eventHandler.UpdateEventStage)

			// Update estimated end date - Only event owner
			events.PATCH("/:event_id/estimated-end-date",
				auth.RequireEventOwner(eventRepo),
				eventHandler.UpdateEstimatedEndDate)

			// Cancel event - Only event owner or admin
			events.PATCH("/:event_id/cancel",
				auth.RequireEventOwner(eventRepo),
				eventHandler.CancelEvent)

			// Pause/resume event - Only event owner or admin
			events.PATCH("/:event_id/pause",
				auth.RequireEventOwner(eventRepo),
				eventHandler.PauseEvent)

			// Get event participants - Any authenticated user
			events.GET("/:event_id/participants", eventHandler.GetEventParticipants)

			// Attachment management - Participant or event owner
			events.POST("/:event_id/participant/:participant_id/attachment",
				auth.RequireParticipantOrOwner(eventRepo),
				attachmentHandler.UploadAttachment)

			// Get event attachments - Any authenticated user
			events.GET("/:event_id/attachments", attachmentHandler.GetEventAttachments)

			// Voting configuration - Only event owner/organizer/admin
			events.POST("/:event_id/voting-config",
				auth.RequireEventOwnerOrOrganizer(eventRepo),
				distributedVoteHandler.CreateVotingConfiguration)

			// Generate assignments - Only event owner/organizer/admin
			events.POST("/:event_id/generate-assignments",
				auth.RequireEventOwnerOrOrganizer(eventRepo),
				distributedVoteHandler.GenerateAssignments)

			// Get participant assignment - Participant themselves or event owner
			events.GET("/:event_id/participants/:participant_id/assignment",
				auth.RequireParticipantOrOwner(eventRepo),
				distributedVoteHandler.GetParticipantAssignment)

			// Submit ranking votes - Participant themselves or event owner
			events.POST("/:event_id/participants/:participant_id/ranking-votes",
				auth.RequireParticipantOrOwner(eventRepo),
				distributedVoteHandler.SubmitRankingVotes)

			// Vote draft - save/restore partial ranking selections before submit
			events.PUT("/:event_id/participants/:participant_id/vote-draft",
				auth.RequireParticipantOrOwner(eventRepo),
				voteDraftHandler.SaveDraft)
			events.GET("/:event_id/participants/:participant_id/vote-draft",
				auth.RequireParticipantOrOwner(eventRepo),
				voteDraftHandler.GetDraft)

			// Get results - Any authenticated user
			events.GET("/:event_id/distributed-results", distributedVoteHandler.GetDistributedResults)

			// Get voting statistics - Any authenticated user
			events.GET("/:event_id/voting-statistics", distributedVoteHandler.GetVotingStatistics)
		}

		// Attachment download - Available to authenticated users
		api.GET("/attachments/:attachment_id/download", attachmentHandler.DownloadAttachment)
	}

	log.Info("Starting Telescopio API server", "port", cfg.Server.Port)

	log.Debug("Configuration",
		"database_url", cfg.GetDatabaseURL(),
		"uploads_dir", cfg.Upload.Dir,
		"gin_mode", cfg.Server.GinMode,
	)

	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server", "error", err)
	}
}
