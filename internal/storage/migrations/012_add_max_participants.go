package migrations

import "gorm.io/gorm"

// migration012Up adds max_participants column to events table
func migration012Up(db *gorm.DB) error {
	// Add column
	if err := db.Exec(`
		ALTER TABLE events
		ADD COLUMN IF NOT EXISTS max_participants INTEGER NOT NULL DEFAULT 20
	`).Error; err != nil {
		return err
	}

	// Add comment
	return db.Exec(`
		COMMENT ON COLUMN events.max_participants IS 'Maximum number of participants allowed for this event (default: 20)'
	`).Error
}

// migration012Down removes max_participants column from events table
func migration012Down(db *gorm.DB) error {
	sql := `
		ALTER TABLE events
		DROP COLUMN IF EXISTS max_participants;
	`
	return db.Exec(sql).Error
}
