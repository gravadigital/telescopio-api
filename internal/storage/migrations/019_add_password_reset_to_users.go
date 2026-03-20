package migrations

import "gorm.io/gorm"

func migration019Up(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS password_reset_token VARCHAR(64) UNIQUE,
		ADD COLUMN IF NOT EXISTS password_reset_expires_at TIMESTAMP WITH TIME ZONE
	`).Error
}

func migration019Down(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS password_reset_token,
		DROP COLUMN IF EXISTS password_reset_expires_at
	`).Error
}
