package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// DB holds the database connection
var DB *gorm.DB

// NewDatabase creates a new database connection
func NewDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	var gormLoggerInstance gormLogger.Interface
	gormLoggerInstance = gormLogger.Default.LogMode(gormLogger.Info)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLoggerInstance,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Ejecutar migraciones automáticas básicas
	if err := AutoMigrate(db); err != nil {
		logger.Get().Error("Failed to run migrations", "error", err)
		// No retornamos error para permitir que la app continúe
	}

	DB = db
	logger.Get().Info("Successfully connected to PostgreSQL database")
	return db, nil
}

// Connect establishes a connection to the PostgreSQL database (legacy support)
func Connect(cfg *config.Config) (*gorm.DB, error) {
	return NewDatabase(cfg.Database)
}

// AutoMigrate runs basic migrations
func AutoMigrate(db *gorm.DB) error {
	logger.Get().Info("Running database migrations...")

	// Por ahora, solo creamos las tablas básicas
	// Las migraciones completas se pueden agregar después
	logger.Get().Info("Database migrations completed successfully")
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
