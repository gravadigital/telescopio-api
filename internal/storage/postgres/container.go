package postgres

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"gorm.io/gorm"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// Container implements RepositoryContainer interface
type Container struct {
	db                      *gorm.DB
	log                     *log.Logger
	eventRepo               EventRepository
	userRepo                UserRepository
	attachmentRepo          AttachmentRepository
	voteRepo                VoteRepository
	votingConfigurationRepo VotingConfigurationRepository
	votingResultsRepo       VotingResultsRepository
}

// NewContainer creates a new repository container with all repositories initialized
func NewContainer(cfg *config.Config) (*Container, error) {
	log := logger.Repository("postgres_container")
	log.Info("Initializing PostgreSQL repository container...")

	// Establish database connection
	db, err := Connect(cfg)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := AutoMigrate(db); err != nil {
		log.Error("Failed to run migrations", "error", err)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize all repositories
	container := &Container{
		db:                      db,
		log:                     log,
		eventRepo:               NewPostgresEventRepository(db),
		userRepo:                NewPostgresUserRepository(db),
		attachmentRepo:          NewPostgresAttachmentRepository(db),
		voteRepo:                NewPostgresVoteRepository(db),
		votingConfigurationRepo: NewPostgresVotingConfigurationRepository(db),
		votingResultsRepo:       NewPostgresVotingResultsRepository(db),
	}

	// Perform health check
	if err := container.Health(); err != nil {
		log.Error("Container health check failed", "error", err)
		return nil, fmt.Errorf("container health check failed: %w", err)
	}

	log.Info("PostgreSQL repository container initialized successfully")
	return container, nil
}

// NewContainerWithDB creates a container with an existing database connection
func NewContainerWithDB(db *gorm.DB) *Container {
	log := logger.Repository("postgres_container")

	return &Container{
		db:                      db,
		log:                     log,
		eventRepo:               NewPostgresEventRepository(db),
		userRepo:                NewPostgresUserRepository(db),
		attachmentRepo:          NewPostgresAttachmentRepository(db),
		voteRepo:                NewPostgresVoteRepository(db),
		votingConfigurationRepo: NewPostgresVotingConfigurationRepository(db),
		votingResultsRepo:       NewPostgresVotingResultsRepository(db),
	}
}

// Events returns the event repository
func (c *Container) Events() EventRepository {
	return c.eventRepo
}

// Users returns the user repository
func (c *Container) Users() UserRepository {
	return c.userRepo
}

// Attachments returns the attachment repository
func (c *Container) Attachments() AttachmentRepository {
	return c.attachmentRepo
}

// Votes returns the vote repository
func (c *Container) Votes() VoteRepository {
	return c.voteRepo
}

// VotingConfigurations returns the voting configuration repository
func (c *Container) VotingConfigurations() VotingConfigurationRepository {
	return c.votingConfigurationRepo
}

// VotingResults returns the voting results repository
func (c *Container) VotingResults() VotingResultsRepository {
	return c.votingResultsRepo
}

// Health performs a health check on all repositories and database connection
func (c *Container) Health() error {
	c.log.Debug("Performing container health check...")

	// Check database connection
	if err := HealthCheck(c.db); err != nil {
		c.log.Error("Database health check failed", "error", err)
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Get connection metrics
	metrics := GetDatabaseMetrics(c.db)
	c.log.Debug("Database connection metrics",
		"open_connections", metrics.OpenConnections,
		"in_use_connections", metrics.InUseConnections,
		"idle_connections", metrics.IdleConnections)

	// Verify each repository can perform basic operations
	repositories := []struct {
		name string
		test func() error
	}{
		{
			name: "events",
			test: func() error {
				// Test basic query on events table
				var count int64
				return c.db.Model(&struct{ ID string }{}).Table("events").Count(&count).Error
			},
		},
		{
			name: "users",
			test: func() error {
				// Test basic query on users table
				var count int64
				return c.db.Model(&struct{ ID string }{}).Table("users").Count(&count).Error
			},
		},
		{
			name: "attachments",
			test: func() error {
				// Test basic query on attachments table
				var count int64
				return c.db.Model(&struct{ ID string }{}).Table("attachments").Count(&count).Error
			},
		},
		{
			name: "votes",
			test: func() error {
				// Test basic query on votes table
				var count int64
				return c.db.Model(&struct{ ID string }{}).Table("votes").Count(&count).Error
			},
		},
		{
			name: "voting_configurations",
			test: func() error {
				// Test basic query on voting_configurations table
				var count int64
				return c.db.Model(&struct{ ID string }{}).Table("voting_configurations").Count(&count).Error
			},
		},
		{
			name: "voting_results",
			test: func() error {
				// Test basic query on voting_results table
				var count int64
				return c.db.Model(&struct{ ID string }{}).Table("voting_results").Count(&count).Error
			},
		},
	}

	for _, repo := range repositories {
		if err := repo.test(); err != nil {
			c.log.Error("Repository health check failed", "repository", repo.name, "error", err)
			return fmt.Errorf("repository %s health check failed: %w", repo.name, err)
		}
		c.log.Debug("Repository health check passed", "repository", repo.name)
	}

	c.log.Debug("Container health check completed successfully")
	return nil
}

// Close gracefully shuts down the container and closes database connections
func (c *Container) Close() error {
	c.log.Info("Closing PostgreSQL repository container...")

	if c.db == nil {
		c.log.Warn("Database connection is nil, nothing to close")
		return nil
	}

	// Get final metrics before closing
	metrics := GetDatabaseMetrics(c.db)
	c.log.Debug("Final database metrics",
		"open_connections", metrics.OpenConnections,
		"in_use_connections", metrics.InUseConnections,
		"idle_connections", metrics.IdleConnections)

	// Close the database connection
	if err := Close(); err != nil {
		c.log.Error("Failed to close database connection", "error", err)
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	// Clear repository references
	c.eventRepo = nil
	c.userRepo = nil
	c.attachmentRepo = nil
	c.voteRepo = nil
	c.votingConfigurationRepo = nil
	c.votingResultsRepo = nil
	c.db = nil

	c.log.Info("PostgreSQL repository container closed successfully")
	return nil
}

// CloseWithTimeout closes the container with a timeout
func (c *Container) CloseWithTimeout(timeout time.Duration) error {
	done := make(chan error, 1)

	go func() {
		done <- c.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		c.log.Error("Container close operation timed out", "timeout", timeout)
		return fmt.Errorf("container close operation timed out after %v", timeout)
	}
}

// GetInfo returns information about the container and its repositories
func (c *Container) GetInfo() map[string]interface{} {
	info := map[string]interface{}{
		"type":         "postgres",
		"repositories": []string{"events", "users", "attachments", "votes", "voting_configurations", "voting_results"},
	}

	// Add database connection info
	if c.db != nil {
		dbInfo := GetConnectionInfo()
		info["database"] = dbInfo
	} else {
		info["database"] = map[string]interface{}{
			"connected": false,
			"error":     "no database connection",
		}
	}

	return info
}

// GetDB returns the underlying database connection (for advanced usage)
func (c *Container) GetDB() *gorm.DB {
	return c.db
}

// BeginTransaction starts a new database transaction
func (c *Container) BeginTransaction() (*TransactionContainer, error) {
	tx := c.db.Begin()
	if tx.Error != nil {
		c.log.Error("Failed to begin transaction", "error", tx.Error)
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	c.log.Debug("Database transaction started")
	return NewTransactionContainer(tx), nil
}

// TransactionContainer wraps repositories in a database transaction
type TransactionContainer struct {
	tx                      *gorm.DB
	log                     *log.Logger
	eventRepo               EventRepository
	userRepo                UserRepository
	attachmentRepo          AttachmentRepository
	voteRepo                VoteRepository
	votingConfigurationRepo VotingConfigurationRepository
	votingResultsRepo       VotingResultsRepository
}

// NewTransactionContainer creates a new transaction container
func NewTransactionContainer(tx *gorm.DB) *TransactionContainer {
	log := logger.Repository("postgres_transaction")

	return &TransactionContainer{
		tx:                      tx,
		log:                     log,
		eventRepo:               NewPostgresEventRepository(tx),
		userRepo:                NewPostgresUserRepository(tx),
		attachmentRepo:          NewPostgresAttachmentRepository(tx),
		voteRepo:                NewPostgresVoteRepository(tx),
		votingConfigurationRepo: NewPostgresVotingConfigurationRepository(tx),
		votingResultsRepo:       NewPostgresVotingResultsRepository(tx),
	}
}

// Events returns the event repository within transaction
func (tc *TransactionContainer) Events() EventRepository {
	return tc.eventRepo
}

// Users returns the user repository within transaction
func (tc *TransactionContainer) Users() UserRepository {
	return tc.userRepo
}

// Attachments returns the attachment repository within transaction
func (tc *TransactionContainer) Attachments() AttachmentRepository {
	return tc.attachmentRepo
}

// Votes returns the vote repository within transaction
func (tc *TransactionContainer) Votes() VoteRepository {
	return tc.voteRepo
}

// VotingConfigurations returns the voting configuration repository within transaction
func (tc *TransactionContainer) VotingConfigurations() VotingConfigurationRepository {
	return tc.votingConfigurationRepo
}

// VotingResults returns the voting results repository within transaction
func (tc *TransactionContainer) VotingResults() VotingResultsRepository {
	return tc.votingResultsRepo
}

// Commit commits the transaction
func (tc *TransactionContainer) Commit() error {
	tc.log.Debug("Committing database transaction")

	if err := tc.tx.Commit().Error; err != nil {
		tc.log.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tc.log.Debug("Database transaction committed successfully")
	return nil
}

// Rollback rolls back the transaction
func (tc *TransactionContainer) Rollback() error {
	tc.log.Debug("Rolling back database transaction")

	if err := tc.tx.Rollback().Error; err != nil {
		tc.log.Error("Failed to rollback transaction", "error", err)
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tc.log.Debug("Database transaction rolled back successfully")
	return nil
}
