package migrations

import "gorm.io/gorm"

// migration006Up inserts sample data for testing and development
func migration006Up(db *gorm.DB) error {
	usersSQL := `
        INSERT INTO users (id, name, lastname, email, role) VALUES 
            ('550e8400-e29b-41d4-a716-446655440000', 'System', 'Administrator', 'admin@telescopio.com', 'admin'),
            ('550e8400-e29b-41d4-a716-446655440001', 'Alice', 'Researcher', 'alice@university.edu', 'participant'),
            ('550e8400-e29b-41d4-a716-446655440002', 'Bob', 'Scientist', 'bob@observatory.org', 'participant'),
            ('550e8400-e29b-41d4-a716-446655440003', 'Carol', 'Professor', 'carol@institute.edu', 'participant'),
            ('550e8400-e29b-41d4-a716-446655440004', 'David', 'PostDoc', 'david@research.com', 'participant'),
            ('550e8400-e29b-41d4-a716-446655440005', 'Eve', 'GradStudent', 'eve@university.edu', 'participant'),
            ('550e8400-e29b-41d4-a716-446655440006', 'Frank', 'PI', 'frank@telescope.org', 'participant'),
            ('550e8400-e29b-41d4-a716-446655440007', 'Grace', 'Astronomer', 'grace@space.gov', 'participant')
        ON CONFLICT (email) DO NOTHING
    `

	if err := db.Exec(usersSQL).Error; err != nil {
		return err
	}

	eventSQL := `
        INSERT INTO events (id, name, description, author_id, start_date, end_date, stage) VALUES 
            ('660e8400-e29b-41d4-a716-446655440000', 
             'Distributed Telescope Time Allocation 2026', 
             'Annual telescope time allocation using distributed voting system based on Merrifield & Saari (2009) mathematical framework for fair and efficient proposal evaluation.',
             '550e8400-e29b-41d4-a716-446655440000',
             '2026-01-15 00:00:00+00',
             '2026-06-30 23:59:59+00',
             'registration')
        ON CONFLICT (id) DO NOTHING
    `

	if err := db.Exec(eventSQL).Error; err != nil {
		return err
	}

	participantsSQL := `
        INSERT INTO event_participants (event_id, user_id) VALUES 
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440001'),
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440002'),
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440003'),
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440004'),
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440005'),
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440006'),
            ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440007')
        ON CONFLICT (event_id, user_id) DO NOTHING
    `

	if err := db.Exec(participantsSQL).Error; err != nil {
		return err
	}

	configSQL := `
        INSERT INTO voting_configurations (
            event_id,
            attachments_per_evaluator,
            min_evaluations_per_file,
            quality_good_threshold,
            quality_bad_threshold,
            adjustment_magnitude,
            use_expertise_matching,
            enable_co_idetection
        ) VALUES (
            '660e8400-e29b-41d4-a716-446655440000',
            8,      -- m = 8 attachments per evaluator
            4,      -- minimum 4 evaluations per file
            0.65,   -- Q_good = 0.65 threshold
            0.35,   -- Q_bad = 0.35 threshold
            3,      -- n = 3 rank adjustment magnitude
            FALSE,  -- basic random assignment
            TRUE    -- enable conflict of interest detection
        ) ON CONFLICT (event_id) DO NOTHING
    `

	if err := db.Exec(configSQL).Error; err != nil {
		return err
	}

	return nil
}

// migration006Down removes sample data
func migration006Down(db *gorm.DB) error {
	queries := []string{
		"DELETE FROM voting_configurations WHERE event_id = '660e8400-e29b-41d4-a716-446655440000'",
		"DELETE FROM event_participants WHERE event_id = '660e8400-e29b-41d4-a716-446655440000'",
		"DELETE FROM events WHERE id = '660e8400-e29b-41d4-a716-446655440000'",
		"DELETE FROM users WHERE email IN ('admin@telescopio.com', 'alice@university.edu', 'bob@observatory.org', 'carol@institute.edu', 'david@research.com', 'eve@university.edu', 'frank@telescope.org', 'grace@space.gov')",
	}

	for _, query := range queries {
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}

	return nil
}
