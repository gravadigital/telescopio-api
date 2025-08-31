package migrations

import "gorm.io/gorm"

// migration001Up creates extensions and custom types
func migration001Up(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return err
	}

	if err := db.Exec(`
        CREATE TYPE event_stage AS ENUM (
            'creation',
            'registration', 
            'attachment_upload',
            'voting',
            'results'
        )
    `).Error; err != nil {
		return err
	}

	if err := db.Exec(`
        CREATE TYPE user_role AS ENUM (
            'admin',
            'participant'
        )
    `).Error; err != nil {
		return err
	}

	return nil
}

// migration001Down drops extensions and custom types
func migration001Down(db *gorm.DB) error {
	if err := db.Exec("DROP TYPE IF EXISTS user_role CASCADE").Error; err != nil {
		return err
	}

	if err := db.Exec("DROP TYPE IF EXISTS event_stage CASCADE").Error; err != nil {
		return err
	}

	// NOTE: We don't drop the UUID extension as it might be used by other applications
	return nil
}
