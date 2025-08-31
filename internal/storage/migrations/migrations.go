package migrations

import (
	"fmt"

	"github.com/gravadigital/telescopio-api/internal/logger"
	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID   string
	Name string
	Up   func(*gorm.DB) error
	Down func(*gorm.DB) error
}

// GetMigrations returns all available migrations in order
func GetMigrations() []Migration {
	return []Migration{
		{
			ID:   "001",
			Name: "create_extensions_and_types",
			Up:   migration001Up,
			Down: migration001Down,
		},
		{
			ID:   "002",
			Name: "create_core_tables",
			Up:   migration002Up,
			Down: migration002Down,
		},
		{
			ID:   "003",
			Name: "create_indexes",
			Up:   migration003Up,
			Down: migration003Down,
		},
		{
			ID:   "004",
			Name: "create_constraints_and_triggers",
			Up:   migration004Up,
			Down: migration004Down,
		},
		{
			ID:   "005",
			Name: "create_views_and_functions",
			Up:   migration005Up,
			Down: migration005Down,
		},
		{
			ID:   "006",
			Name: "insert_sample_data",
			Up:   migration006Up,
			Down: migration006Down,
		},
	}
}

// RunMigrations executes all pending migrations
func RunMigrations(db *gorm.DB) error {
	log := logger.Migration()

	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrations := GetMigrations()

	for _, migration := range migrations {
		if hasBeenRun(db, migration.ID) {
			log.Debug("Migration already applied, skipping", "id", migration.ID, "name", migration.Name)
			continue
		}

		log.Info("Running migration", "id", migration.ID, "name", migration.Name)

		err := db.Transaction(func(tx *gorm.DB) error {
			if err := migration.Up(tx); err != nil {
				return fmt.Errorf("failed to run migration %s: %w", migration.ID, err)
			}

			return recordMigration(tx, migration.ID, migration.Name)
		})
		if err != nil {
			return err
		}

		log.Info("Successfully applied migration", "id", migration.ID)
	}

	log.Info("All migrations completed successfully")
	return nil
}

// createMigrationsTable creates the migrations tracking table
func createMigrationsTable(db *gorm.DB) error {
	return db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            id VARCHAR(10) PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        )
    `).Error
}

// hasBeenRun checks if a migration has already been applied
func hasBeenRun(db *gorm.DB, migrationID string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE id = ?", migrationID).Scan(&count)
	return count > 0
}

// recordMigration records that a migration has been applied
func recordMigration(db *gorm.DB, migrationID, name string) error {
	return db.Exec("INSERT INTO schema_migrations (id, name) VALUES (?, ?)", migrationID, name).Error
}

// RollbackMigration rolls back the last applied migration
func RollbackMigration(db *gorm.DB) error {
	log := logger.Migration()

	var lastMigration struct {
		ID   string
		Name string
	}

	err := db.Raw(`
        SELECT id, name FROM schema_migrations 
        ORDER BY applied_at DESC 
        LIMIT 1
    `).Scan(&lastMigration).Error
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	if lastMigration.ID == "" {
		return fmt.Errorf("no migrations to rollback")
	}

	migrations := GetMigrations()
	var targetMigration *Migration

	for _, migration := range migrations {
		if migration.ID == lastMigration.ID {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration %s not found", lastMigration.ID)
	}

	log.Info("Rolling back migration", "id", targetMigration.ID, "name", targetMigration.Name)

	err = db.Transaction(func(tx *gorm.DB) error {
		if err = targetMigration.Down(tx); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", targetMigration.ID, err)
		}

		return tx.Exec("DELETE FROM schema_migrations WHERE id = ?", targetMigration.ID).Error
	})
	if err != nil {
		return err
	}

	log.Info("Successfully rolled back migration", "id", targetMigration.ID)
	return nil
}
