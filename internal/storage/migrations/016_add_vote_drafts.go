package migrations

import "gorm.io/gorm"

// migration016Up creates the vote_drafts table for persisting partial voting selections.
// A vote draft is automatically saved as the participant selects rankings, so they don't
// lose progress if they leave the page before submitting.
func migration016Up(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE TABLE vote_drafts (
			id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
			event_id       UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			assignment_id  UUID        NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
			participant_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rankings       JSONB       NOT NULL DEFAULT '[]',
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_vote_draft_assignment_participant UNIQUE (assignment_id, participant_id)
		)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`CREATE INDEX idx_vote_drafts_assignment ON vote_drafts(assignment_id)`).Error; err != nil {
		return err
	}

	return db.Exec(`CREATE INDEX idx_vote_drafts_participant ON vote_drafts(participant_id)`).Error
}

// migration016Down removes the vote_drafts table and its indexes
func migration016Down(db *gorm.DB) error {
	if err := db.Exec(`DROP INDEX IF EXISTS idx_vote_drafts_participant`).Error; err != nil {
		return err
	}

	if err := db.Exec(`DROP INDEX IF EXISTS idx_vote_drafts_assignment`).Error; err != nil {
		return err
	}

	return db.Exec(`DROP TABLE IF EXISTS vote_drafts`).Error
}
