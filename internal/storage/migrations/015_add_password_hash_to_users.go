package migrations

import "gorm.io/gorm"

// migration015Up adds password_hash column to users table
func migration015Up(db *gorm.DB) error {
	if err := db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT ''
	`).Error; err != nil {
		return err
	}

	// Remove the default once the column exists (it's only needed for existing rows)
	return db.Exec(`
		ALTER TABLE users
		ALTER COLUMN password_hash DROP DEFAULT
	`).Error
}

// migration015Down removes password_hash column from users table
func migration015Down(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS password_hash
	`).Error
}
