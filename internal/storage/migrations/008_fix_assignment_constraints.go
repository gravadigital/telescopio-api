package migrations

import "gorm.io/gorm"

// migration008Up adds unique constraint for assignments and improves validation
func migration008Up(db *gorm.DB) error {
	// Add unique constraint to prevent duplicate assignments for same participant in same event
	constraints := []string{
		// Ensure each participant can only have ONE assignment per event
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'unique_participant_event_assignment'
			) THEN
				ALTER TABLE assignments
				ADD CONSTRAINT unique_participant_event_assignment
				UNIQUE (event_id, participant_id);
			END IF;
		END $$`,
	}

	for _, constraintSQL := range constraints {
		if err := db.Exec(constraintSQL).Error; err != nil {
			return err
		}
	}

	return nil
}

// migration008Down removes the unique constraint
func migration008Down(db *gorm.DB) error {
	constraints := []string{
		"ALTER TABLE assignments DROP CONSTRAINT IF EXISTS unique_participant_event_assignment",
	}

	for _, constraintSQL := range constraints {
		if err := db.Exec(constraintSQL).Error; err != nil {
			return err
		}
	}

	return nil
}
