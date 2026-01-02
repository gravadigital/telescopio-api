package migrations

import "gorm.io/gorm"

// migration011Up adds NOT NULL constraint and unique index to shareable_link
func migration011Up(db *gorm.DB) error {
	sqls := []string{
		// 1. Ensure all existing events have shareable_link populated
		`UPDATE events SET shareable_link = '/events/' || id::text WHERE shareable_link IS NULL OR shareable_link = ''`,

		// 2. Add NOT NULL constraint to shareable_link
		`ALTER TABLE events ALTER COLUMN shareable_link SET NOT NULL`,

		// 3. Add unique index on shareable_link
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_events_shareable_link ON events(shareable_link)`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}

// migration011Down removes constraints from shareable_link
func migration011Down(db *gorm.DB) error {
	sqls := []string{
		`DROP INDEX IF EXISTS idx_events_shareable_link`,
		`ALTER TABLE events ALTER COLUMN shareable_link DROP NOT NULL`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}
