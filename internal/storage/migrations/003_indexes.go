package migrations

import "gorm.io/gorm"

// migration003Up creates performance indexes
func migration003Up(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)",

		"CREATE INDEX IF NOT EXISTS idx_events_author ON events(author_id)",
		"CREATE INDEX IF NOT EXISTS idx_events_stage ON events(stage)",
		"CREATE INDEX IF NOT EXISTS idx_events_dates ON events(start_date, end_date)",
		"CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at DESC)",

		"CREATE INDEX IF NOT EXISTS idx_event_participants_event ON event_participants(event_id)",
		"CREATE INDEX IF NOT EXISTS idx_event_participants_user ON event_participants(user_id)",

		"CREATE INDEX IF NOT EXISTS idx_attachments_event ON attachments(event_id)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_participant ON attachments(participant_id)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_vote_count ON attachments(vote_count DESC)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_uploaded_at ON attachments(uploaded_at DESC)",

		"CREATE INDEX IF NOT EXISTS idx_voting_configurations_event ON voting_configurations(event_id)",

		"CREATE INDEX IF NOT EXISTS idx_assignments_event ON assignments(event_id)",
		"CREATE INDEX IF NOT EXISTS idx_assignments_participant ON assignments(participant_id)",
		"CREATE INDEX IF NOT EXISTS idx_assignments_completed ON assignments(is_completed)",
		"CREATE INDEX IF NOT EXISTS idx_assignments_quality ON assignments(quality_score DESC)",
		"CREATE INDEX IF NOT EXISTS idx_assignments_round ON assignments(assignment_round)",

		"CREATE INDEX IF NOT EXISTS idx_votes_event ON votes(event_id)",
		"CREATE INDEX IF NOT EXISTS idx_votes_assignment ON votes(assignment_id)",
		"CREATE INDEX IF NOT EXISTS idx_votes_voter ON votes(voter_id)",
		"CREATE INDEX IF NOT EXISTS idx_votes_attachment ON votes(attachment_id)",
		"CREATE INDEX IF NOT EXISTS idx_votes_rank ON votes(rank_position)",
		"CREATE INDEX IF NOT EXISTS idx_votes_score ON votes(score DESC)",
		"CREATE INDEX IF NOT EXISTS idx_votes_quality ON votes(is_quality_vote)",
		"CREATE INDEX IF NOT EXISTS idx_votes_event_voter ON votes(event_id, voter_id)",

		"CREATE INDEX IF NOT EXISTS idx_voting_results_event ON voting_results(event_id)",
		"CREATE INDEX IF NOT EXISTS idx_voting_results_calculated_at ON voting_results(calculated_at DESC)",
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			return err
		}
	}

	return nil
}

// migration003Down drops performance indexes
func migration003Down(db *gorm.DB) error {
	indexes := []string{
		"idx_users_email",
		"idx_users_role",
		"idx_events_author",
		"idx_events_stage",
		"idx_events_dates",
		"idx_events_created_at",
		"idx_event_participants_event",
		"idx_event_participants_user",
		"idx_attachments_event",
		"idx_attachments_participant",
		"idx_attachments_vote_count",
		"idx_attachments_uploaded_at",
		"idx_voting_configurations_event",
		"idx_assignments_event",
		"idx_assignments_participant",
		"idx_assignments_completed",
		"idx_assignments_quality",
		"idx_assignments_round",
		"idx_votes_event",
		"idx_votes_assignment",
		"idx_votes_voter",
		"idx_votes_attachment",
		"idx_votes_rank",
		"idx_votes_score",
		"idx_votes_quality",
		"idx_votes_event_voter",
		"idx_voting_results_event",
		"idx_voting_results_calculated_at",
	}

	for _, index := range indexes {
		if err := db.Exec("DROP INDEX IF EXISTS " + index).Error; err != nil {
			return err
		}
	}

	return nil
}
