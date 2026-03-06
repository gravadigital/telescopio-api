package migrations

import "gorm.io/gorm"

func migration017Up(db *gorm.DB) error {
	if err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id VARCHAR UNIQUE`).Error; err != nil {
		return err
	}
	if err := db.Exec(`ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL`).Error
}

func migration017Down(db *gorm.DB) error {
	if err := db.Exec(`DROP INDEX IF EXISTS idx_users_google_id`).Error; err != nil {
		return err
	}
	// NOTE: NOT restoring password_hash NOT NULL to avoid breaking OAuth users already created
	return db.Exec(`ALTER TABLE users DROP COLUMN IF EXISTS google_id`).Error
}
