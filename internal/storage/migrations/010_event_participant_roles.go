package migrations

import "gorm.io/gorm"

// migration010Up adds role-based permissions to event participants
// and makes user.role nullable (only for admins)
func migration010Up(db *gorm.DB) error {
	// 1. Create enum type for event participant roles (only if it doesn't exist)
	err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'event_participant_role') THEN
				CREATE TYPE event_participant_role AS ENUM ('creator', 'participant');
			END IF;
		END $$;
	`).Error
	if err != nil {
		return err
	}

	sqls := []string{
		// 2. Add role column to event_participants
		`ALTER TABLE event_participants
		 ADD COLUMN IF NOT EXISTS role event_participant_role NOT NULL DEFAULT 'participant'`,

		// 3. Update existing creators based on events.author_id
		`UPDATE event_participants ep
		 SET role = 'creator'
		 FROM events e
		 WHERE ep.event_id = e.id
		   AND ep.user_id = e.author_id`,

		// 4. Make users.role nullable (will only contain 'admin' or NULL)
		`ALTER TABLE users ALTER COLUMN role DROP NOT NULL`,
		`ALTER TABLE users ALTER COLUMN role DROP DEFAULT`,

		// 5. Clear non-admin roles from users
		`UPDATE users SET role = NULL WHERE role != 'admin'`,

		// 6. Add shareable_link column to events
		`ALTER TABLE events ADD COLUMN IF NOT EXISTS shareable_link VARCHAR(255)`,

		// 7. Temporarily disable the constraint to allow updates
		`ALTER TABLE events DROP CONSTRAINT IF EXISTS future_start_date`,

		// 8. Generate shareable links for existing events
		`UPDATE events SET shareable_link = '/events/' || id::text WHERE shareable_link IS NULL`,

		// 9. Restore the constraint but don't validate existing rows (NOT VALID means it only applies to new/modified rows)
		`ALTER TABLE events ADD CONSTRAINT future_start_date CHECK (start_date >= CURRENT_TIMESTAMP - INTERVAL '1 day') NOT VALID`,

		// 10. Add comments for documentation
		`COMMENT ON COLUMN event_participants.role IS 'Role of the user in this specific event (creator: event owner, participant: regular participant)'`,
		`COMMENT ON COLUMN users.role IS 'Global system role (only ''admin'' or NULL for regular users)'`,
		`COMMENT ON COLUMN events.shareable_link IS 'URL path for sharing this event on social media'`,
	}

	for _, sql := range sqls {
		if err = db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}

// migration010Down reverts the role-based permissions changes
func migration010Down(db *gorm.DB) error {
	sqls := []string{
		// Remove shareable_link
		`ALTER TABLE events DROP COLUMN IF EXISTS shareable_link`,

		// Restore original user roles (set all to 'participant')
		`UPDATE users SET role = 'participant' WHERE role IS NULL`,
		`ALTER TABLE users ALTER COLUMN role SET DEFAULT 'participant'`,
		`ALTER TABLE users ALTER COLUMN role SET NOT NULL`,

		// Remove role column from event_participants
		`ALTER TABLE event_participants DROP COLUMN IF EXISTS role`,

		// Drop enum type
		`DROP TYPE IF EXISTS event_participant_role`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}
