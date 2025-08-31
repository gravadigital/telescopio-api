package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/migrations"
)

// DB holds the database connection
var DB *gorm.DB

// Connect establishes a connection to the PostgreSQL database
func Connect(cfg *config.Config) (*gorm.DB, error) {
	log := logger.Database()
	dsn := cfg.GetDatabaseURL()

	var gormLoggerInstance gormLogger.Interface
	if cfg.Server.GinMode == "debug" {
		gormLoggerInstance = gormLogger.Default.LogMode(gormLogger.Info)
	} else {
		gormLoggerInstance = gormLogger.Default.LogMode(gormLogger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLoggerInstance,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db

	log.Info("Successfully connected to PostgreSQL database")
	return db, nil
}

// AutoMigrate runs the structured migrations for all models
func AutoMigrate(db *gorm.DB) error {
	log := logger.Migration()
	log.Info("Running database migrations...")

	// Run all migrations using the new migration system
	if err := migrations.RunMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
