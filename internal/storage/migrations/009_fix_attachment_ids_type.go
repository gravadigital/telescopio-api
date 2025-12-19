package migrations

import "gorm.io/gorm"

func migration009Up(db *gorm.DB) error {
	// Change attachment_ids column type from uuid[] to text[]
	// This fixes the "operator does not exist: text = uuid" error
	sql := `
		-- First, cast existing uuid[] data to text[]
		ALTER TABLE assignments 
		ALTER COLUMN attachment_ids TYPE text[] 
		USING attachment_ids::text[];
	`
	
	return db.Exec(sql).Error
}

func migration009Down(db *gorm.DB) error {
	// Revert back to uuid[] (if needed)
	sql := `
		ALTER TABLE assignments 
		ALTER COLUMN attachment_ids TYPE uuid[] 
		USING attachment_ids::uuid[];
	`
	
	return db.Exec(sql).Error
}
