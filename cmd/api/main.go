package main

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/handlers"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/middleware/events"
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

	eventHandler := handlers.NewEventHandler(eventRepo, userRepo, cfg)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentRepo, eventRepo, userRepo, cfg)

	configRepo := postgres.NewPostgresVotingConfigurationRepository(db)
	resultsRepo := postgres.NewPostgresVotingResultsRepository(db)
	distributedVoteHandler := handlers.NewDistributedVoteHandler(voteRepo, eventRepo, attachmentRepo, userRepo, configRepo, resultsRepo, cfg)

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
		events := api.Group("/events")
		{
			// Standard event management
			events.GET("", eventHandler.GetAllEvents)
			events.POST("", eventHandler.CreateEvent)
			events.PATCH("/:event_id/stage", eventHandler.UpdateEventStage)
			events.POST("/:event_id/register", eventHandler.RegisterParticipant)
			events.GET("/:event_id/participants", eventHandler.GetEventParticipants)

			// Attachment management
			events.POST("/:event_id/participant/:participant_id/attachment", attachmentHandler.UploadAttachment)

			// Voting system
			events.POST("/:event_id/voting-config", distributedVoteHandler.CreateVotingConfiguration)
			events.POST("/:event_id/generate-assignments", distributedVoteHandler.GenerateAssignments)
			events.GET("/:event_id/participants/:participant_id/assignment", distributedVoteHandler.GetParticipantAssignment)
			events.POST("/:event_id/participants/:participant_id/ranking-votes", distributedVoteHandler.SubmitRankingVotes)
			events.GET("/:event_id/distributed-results", distributedVoteHandler.GetDistributedResults)
			events.GET("/:event_id/voting-statistics", distributedVoteHandler.GetVotingStatistics)
		}
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
