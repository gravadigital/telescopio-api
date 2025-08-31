package migrations

import "gorm.io/gorm"

// migration002Up creates all core tables using GORM AutoMigrate
func migration002Up(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// migration002Down drops all core tables
func migration002Down(db *gorm.DB) error {
	tables := []string{
		"voting_results",
		"votes",
		"assignments",
		"voting_configurations",
		"attachments",
		"event_participants",
		"events",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}
