package migrations

import "gorm.io/gorm"

// migration012Up unifies registration and attachment_upload stages into participation
func migration012Up(db *gorm.DB) error {
	sqls := []string{
		// 1. Drop views that depend on the stage column
		`DROP VIEW IF EXISTS system_validation CASCADE`,

		// 2. Temporarily disable the future_start_date constraint
		`ALTER TABLE events DROP CONSTRAINT IF EXISTS future_start_date`,

		// 3. Migrate all events in old stages to a temporary stage
		// We use 'creation' temporarily to avoid enum constraint violations
		`UPDATE events 
		 SET stage = 'creation' 
		 WHERE stage IN ('registration', 'attachment_upload')`,

		// 4. Remove the default value from stage column before changing type
		`ALTER TABLE events ALTER COLUMN stage DROP DEFAULT`,

		// 5. Create a new enum with updated values
		`CREATE TYPE event_stage_new AS ENUM (
			'creation', 
			'participation', 
			'voting', 
			'results'
		)`,

		// 6. Update the column to use the new type
		// USING clause converts the old enum value to text and then to new enum
		`ALTER TABLE events 
		 ALTER COLUMN stage TYPE event_stage_new 
		 USING stage::text::event_stage_new`,

		// 7. Drop the old enum type
		`DROP TYPE event_stage`,

		// 8. Rename the new enum to the original name
		`ALTER TYPE event_stage_new RENAME TO event_stage`,

		// 9. Restore the default value for the stage column
		`ALTER TABLE events ALTER COLUMN stage SET DEFAULT 'creation'::event_stage`,

		// 10. Recreate the system_validation view with the new enum
		`CREATE VIEW system_validation AS
        SELECT 
            e.id as event_id,
            e.name as event_name,
            e.stage as current_stage,
            vc.attachments_per_evaluator as m_parameter,
            vc.quality_good_threshold as q_good,
            vc.quality_bad_threshold as q_bad,
            vc.adjustment_magnitude as n_parameter,
            COUNT(DISTINCT ep.user_id) as n_participants,
            COUNT(DISTINCT att.id) as k_attachments,
            CASE 
                WHEN vc.attachments_per_evaluator >= CEIL(2 * LOG(2, GREATEST(COUNT(DISTINCT att.id), 2))) 
                THEN 'OPTIMAL'
                WHEN vc.attachments_per_evaluator >= CEIL(LOG(2, GREATEST(COUNT(DISTINCT att.id), 2)))
                THEN 'ACCEPTABLE'
                ELSE 'INSUFFICIENT'
            END as mathematical_validity,
            CEIL(2 * LOG(2, GREATEST(COUNT(DISTINCT att.id), 2))) as optimal_m,
            ROUND(
                (COUNT(DISTINCT ep.user_id) * vc.attachments_per_evaluator)::DECIMAL / 
                GREATEST(COUNT(DISTINCT att.id), 1), 2
            ) as coverage_ratio,
            check_coverage_constraints(e.id) as coverage_sufficient
        FROM events e
        LEFT JOIN voting_configurations vc ON e.id = vc.event_id
        LEFT JOIN event_participants ep ON e.id = ep.event_id
        LEFT JOIN attachments att ON e.id = att.event_id
        GROUP BY e.id, e.name, e.stage, vc.attachments_per_evaluator, vc.quality_good_threshold, 
                 vc.quality_bad_threshold, vc.adjustment_magnitude`,

		// 11. Finally, update migrated events to the new 'participation' stage
		// We identify them by checking if they're still in 'creation' stage
		// and have participants or attachments (indicating they were in a later stage)
		`UPDATE events 
		 SET stage = 'participation' 
		 WHERE stage = 'creation' 
		 AND (
			 EXISTS (SELECT 1 FROM event_participants WHERE event_participants.event_id = events.id)
			 OR EXISTS (SELECT 1 FROM attachments WHERE attachments.event_id = events.id)
		 )`,

		// 12. Re-enable the future_start_date constraint
		// Using NOT VALID to avoid checking existing rows
		// The constraint will only be checked for new inserts and updates
		`ALTER TABLE events 
		 ADD CONSTRAINT future_start_date 
		 CHECK (start_date >= CURRENT_TIMESTAMP - INTERVAL '1 day') NOT VALID`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}

// migration012Down reverts the unification of participation stages
func migration012Down(db *gorm.DB) error {
	sqls := []string{
		// 1. Drop views that depend on the stage column
		`DROP VIEW IF EXISTS system_validation CASCADE`,

		// 2. Temporarily disable the future_start_date constraint
		`ALTER TABLE events DROP CONSTRAINT IF EXISTS future_start_date`,

		// 3. Migrate events in 'participation' stage to 'creation' temporarily
		`UPDATE events 
		 SET stage = 'creation' 
		 WHERE stage = 'participation'`,

		// 4. Remove the default value from stage column before changing type
		`ALTER TABLE events ALTER COLUMN stage DROP DEFAULT`,

		// 5. Create the old enum with original values
		`CREATE TYPE event_stage_old AS ENUM (
			'creation',
			'registration',
			'attachment_upload',
			'voting',
			'results'
		)`,

		// 6. Update the column to use the old type
		`ALTER TABLE events 
		 ALTER COLUMN stage TYPE event_stage_old 
		 USING stage::text::event_stage_old`,

		// 7. Drop the current enum type
		`DROP TYPE event_stage`,

		// 8. Rename the old enum back to the original name
		`ALTER TYPE event_stage_old RENAME TO event_stage`,

		// 9. Restore the default value for the stage column
		`ALTER TABLE events ALTER COLUMN stage SET DEFAULT 'creation'::event_stage`,

		// 10. Recreate the system_validation view with the old enum
		`CREATE VIEW system_validation AS
        SELECT 
            e.id as event_id,
            e.name as event_name,
            e.stage as current_stage,
            vc.attachments_per_evaluator as m_parameter,
            vc.quality_good_threshold as q_good,
            vc.quality_bad_threshold as q_bad,
            vc.adjustment_magnitude as n_parameter,
            COUNT(DISTINCT ep.user_id) as n_participants,
            COUNT(DISTINCT att.id) as k_attachments,
            CASE 
                WHEN vc.attachments_per_evaluator >= CEIL(2 * LOG(2, GREATEST(COUNT(DISTINCT att.id), 2))) 
                THEN 'OPTIMAL'
                WHEN vc.attachments_per_evaluator >= CEIL(LOG(2, GREATEST(COUNT(DISTINCT att.id), 2)))
                THEN 'ACCEPTABLE'
                ELSE 'INSUFFICIENT'
            END as mathematical_validity,
            CEIL(2 * LOG(2, GREATEST(COUNT(DISTINCT att.id), 2))) as optimal_m,
            ROUND(
                (COUNT(DISTINCT ep.user_id) * vc.attachments_per_evaluator)::DECIMAL / 
                GREATEST(COUNT(DISTINCT att.id), 1), 2
            ) as coverage_ratio,
            check_coverage_constraints(e.id) as coverage_sufficient
        FROM events e
        LEFT JOIN voting_configurations vc ON e.id = vc.event_id
        LEFT JOIN event_participants ep ON e.id = ep.event_id
        LEFT JOIN attachments att ON e.id = att.event_id
        GROUP BY e.id, e.name, e.stage, vc.attachments_per_evaluator, vc.quality_good_threshold, 
                 vc.quality_bad_threshold, vc.adjustment_magnitude`,

		// 11. Restore events to 'registration' stage as default
		// In a real rollback scenario, we can't perfectly restore which stage they were in,
		// so we default to 'registration' as it's the earlier stage
		`UPDATE events 
		 SET stage = 'registration' 
		 WHERE stage = 'creation' 
		 AND (
			 EXISTS (SELECT 1 FROM event_participants WHERE event_participants.event_id = events.id)
			 OR EXISTS (SELECT 1 FROM attachments WHERE attachments.event_id = events.id)
		 )`,

		// 12. Re-enable the future_start_date constraint
		// Using NOT VALID to avoid checking existing rows
		// The constraint will only be checked for new inserts and updates
		`ALTER TABLE events 
		 ADD CONSTRAINT future_start_date 
		 CHECK (start_date >= CURRENT_TIMESTAMP - INTERVAL '1 day') NOT VALID`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}

