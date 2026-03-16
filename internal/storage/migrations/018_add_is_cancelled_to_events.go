package migrations

import "gorm.io/gorm"

func migration018Up(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE events ADD COLUMN IF NOT EXISTS is_cancelled BOOLEAN NOT NULL DEFAULT false`).Error
}

func migration018Down(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE events DROP COLUMN IF EXISTS is_cancelled`).Error
}
