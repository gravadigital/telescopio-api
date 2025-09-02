package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/migrations"
)

// DB holds the database connection
var DB *gorm.DB

// ConnectionConfig holds database connection configuration
type ConnectionConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConnectionConfig returns default connection configuration
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
	}
}

// DatabaseMetrics holds database connection metrics
type DatabaseMetrics struct {
	OpenConnections  int
	InUseConnections int
	IdleConnections  int
}

// Connect establishes a connection to the PostgreSQL database with enhanced configuration
func Connect(cfg *config.Config) (*gorm.DB, error) {
	return ConnectWithConfig(cfg, DefaultConnectionConfig())
}

// ConnectWithConfig establishes a connection with custom configuration
func ConnectWithConfig(cfg *config.Config, connCfg *ConnectionConfig) (*gorm.DB, error) {
	log := logger.Database()

	// Validate configuration
	if err := validateDatabaseConfig(cfg); err != nil {
		log.Error("Database configuration validation failed", "error", err)
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	dsn := cfg.GetDatabaseURL()
	log.Debug("Connecting to database", "host", cfg.DB.Host, "port", cfg.DB.Port, "database", cfg.DB.Name)

	// Configure GORM logger based on environment
	var gormLoggerInstance gormLogger.Interface
	if cfg.Server.GinMode == "debug" {
		gormLoggerInstance = gormLogger.Default.LogMode(gormLogger.Info)
		log.Debug("GORM logging enabled (debug mode)")
	} else {
		gormLoggerInstance = gormLogger.Default.LogMode(gormLogger.Silent)
		log.Debug("GORM logging disabled (production mode)")
	}

	// GORM configuration with timeouts
	gormConfig := &gorm.Config{
		Logger: gormLoggerInstance,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true, // Enable prepared statements for better performance
	}

	// Attempt connection with retry logic
	var db *gorm.DB
	var err error
	maxRetries := 3
	retryDelay := time.Second * 2

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Debug("Database connection attempt", "attempt", attempt, "max_retries", maxRetries)

		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			break
		}

		log.Warn("Database connection failed", "attempt", attempt, "error", err)
		if attempt < maxRetries {
			log.Debug("Retrying database connection", "delay", retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	if err != nil {
		log.Error("Failed to connect to database after retries", "error", err, "attempts", maxRetries)
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool
	if err := configureConnectionPool(db, connCfg); err != nil {
		log.Error("Failed to configure connection pool", "error", err)
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	// Test the connection
	if err := testConnection(db); err != nil {
		log.Error("Database connection test failed", "error", err)
		return nil, fmt.Errorf("database connection test failed: %w", err)
	}

	// Set global DB variable
	DB = db

	// Log connection success with metrics
	metrics := GetDatabaseMetrics(db)
	log.Info("Successfully connected to PostgreSQL database",
		"host", cfg.DB.Host,
		"database", cfg.DB.Name,
		"max_open_conns", connCfg.MaxOpenConns,
		"max_idle_conns", connCfg.MaxIdleConns,
		"open_connections", metrics.OpenConnections)

	return db, nil
}

// validateDatabaseConfig validates the database configuration
func validateDatabaseConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if cfg.DB.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}

	if cfg.DB.Port == "" {
		return fmt.Errorf("database port cannot be empty")
	}

	if cfg.DB.Name == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if cfg.DB.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}

	// Password can be empty for local development
	// if cfg.DB.Password == "" {
	//     return fmt.Errorf("database password cannot be empty")
	// }

	return nil
}

// configureConnectionPool configures the database connection pool
func configureConnectionPool(db *gorm.DB, cfg *ConnectionConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set maximum number of open connections to the database
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// Set maximum number of idle connections in the pool
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	// Set maximum lifetime of a connection
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Set maximum idle time of a connection
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return nil
}

// testConnection tests the database connection
func testConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// GetDatabaseMetrics returns current database connection metrics
func GetDatabaseMetrics(db *gorm.DB) *DatabaseMetrics {
	sqlDB, err := db.DB()
	if err != nil {
		return &DatabaseMetrics{}
	}

	stats := sqlDB.Stats()
	return &DatabaseMetrics{
		OpenConnections:  stats.OpenConnections,
		InUseConnections: stats.InUse,
		IdleConnections:  stats.Idle,
	}
}

// HealthCheck performs a health check on the database connection
func HealthCheck(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	return testConnection(db)
}

// HealthCheckWithTimeout performs a health check with custom timeout
func HealthCheckWithTimeout(db *gorm.DB, timeout time.Duration) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// AutoMigrate runs the structured migrations for all models with enhanced error handling
func AutoMigrate(db *gorm.DB) error {
	log := logger.Migration()
	log.Info("Starting database migrations...")

	if db == nil {
		log.Error("Database connection is nil")
		return fmt.Errorf("database connection is nil")
	}

	// Test connection before running migrations
	if err := HealthCheck(db); err != nil {
		log.Error("Database health check failed before migrations", "error", err)
		return fmt.Errorf("database health check failed: %w", err)
	}

	startTime := time.Now()

	// Run all migrations using the new migration system
	if err := migrations.RunMigrations(db); err != nil {
		log.Error("Database migrations failed", "error", err, "duration", time.Since(startTime))
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	duration := time.Since(startTime)
	log.Info("Database migrations completed successfully", "duration", duration)
	return nil
}

// Close closes the database connection with cleanup
func Close() error {
	log := logger.Database()

	if DB == nil {
		log.Warn("Attempted to close nil database connection")
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Error("Failed to get database instance for closing", "error", err)
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Get final metrics before closing
	metrics := GetDatabaseMetrics(DB)
	log.Debug("Database metrics before closing",
		"open_connections", metrics.OpenConnections,
		"in_use_connections", metrics.InUseConnections,
		"idle_connections", metrics.IdleConnections)

	if err := sqlDB.Close(); err != nil {
		log.Error("Failed to close database connection", "error", err)
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	// Clear global DB variable
	DB = nil

	log.Info("Database connection closed successfully")
	return nil
}

// CloseWithTimeout closes the database connection with a timeout
func CloseWithTimeout(timeout time.Duration) error {
	done := make(chan error, 1)

	go func() {
		done <- Close()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("database close operation timed out after %v", timeout)
	}
}

// GetConnectionInfo returns information about the current database connection
func GetConnectionInfo() map[string]interface{} {
	if DB == nil {
		return map[string]interface{}{
			"connected": false,
			"error":     "no database connection",
		}
	}

	metrics := GetDatabaseMetrics(DB)
	sqlDB, err := DB.DB()
	if err != nil {
		return map[string]interface{}{
			"connected": false,
			"error":     err.Error(),
		}
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"connected":            true,
		"open_connections":     metrics.OpenConnections,
		"in_use_connections":   metrics.InUseConnections,
		"idle_connections":     metrics.IdleConnections,
		"max_open_connections": stats.MaxOpenConnections,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
