package migrations

import "gorm.io/gorm"

// migration017Up adds is_cancelled column to events table
func migration017Up(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE events
		ADD COLUMN IF NOT EXISTS is_cancelled BOOLEAN NOT NULL DEFAULT false
	`).Error
}

// migration017Down removes is_cancelled column from events table
func migration017Down(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE events
		DROP COLUMN IF EXISTS is_cancelled
	`).Error
}
