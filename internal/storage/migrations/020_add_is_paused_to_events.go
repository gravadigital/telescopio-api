package migrations

import "gorm.io/gorm"

func migration020Up(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE events ADD COLUMN IF NOT EXISTS is_paused BOOLEAN NOT NULL DEFAULT false`).Error
}

func migration020Down(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE events DROP COLUMN IF EXISTS is_paused`).Error
}
