package migrations

import "gorm.io/gorm"

// migration007Up adds organizer column to events table
func migration007Up(db *gorm.DB) error {
	sql := `
		ALTER TABLE events
		ADD COLUMN IF NOT EXISTS organizer VARCHAR(200) DEFAULT '';
	`
	return db.Exec(sql).Error
}

// migration007Down removes organizer column from events table
func migration007Down(db *gorm.DB) error {
	sql := `
		ALTER TABLE events
		DROP COLUMN IF EXISTS organizer;
	`
	return db.Exec(sql).Error
}
