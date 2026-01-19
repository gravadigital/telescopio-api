package migrations

import "gorm.io/gorm"

// migration014Up adds estimated end date fields to events table
// These fields store the estimated closing dates for participation and voting stages
// Both fields are nullable to maintain compatibility with existing events
func migration014Up(db *gorm.DB) error {
	// Add participation_estimated_end_date column
	if err := db.Exec(`
		ALTER TABLE events
		ADD COLUMN IF NOT EXISTS participation_estimated_end_date DATE NULL
	`).Error; err != nil {
		return err
	}

	// Add voting_estimated_end_date column
	if err := db.Exec(`
		ALTER TABLE events
		ADD COLUMN IF NOT EXISTS voting_estimated_end_date DATE NULL
	`).Error; err != nil {
		return err
	}

	// Add comments for documentation
	if err := db.Exec(`
		COMMENT ON COLUMN events.participation_estimated_end_date IS 'Estimated end date for participation stage (informative, not enforced)'
	`).Error; err != nil {
		return err
	}

	return db.Exec(`
		COMMENT ON COLUMN events.voting_estimated_end_date IS 'Estimated end date for voting stage (informative, not enforced)'
	`).Error
}

// migration014Down removes estimated end date fields from events table
func migration014Down(db *gorm.DB) error {
	if err := db.Exec(`
		ALTER TABLE events
		DROP COLUMN IF EXISTS participation_estimated_end_date
	`).Error; err != nil {
		return err
	}

	return db.Exec(`
		ALTER TABLE events
		DROP COLUMN IF EXISTS voting_estimated_end_date
	`).Error
}

