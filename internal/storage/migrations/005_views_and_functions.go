package migrations

import "gorm.io/gorm"

// migration005Up creates analytical views for system monitoring and validation
func migration005Up(db *gorm.DB) error {
	views := []string{
		`CREATE VIEW assignment_statistics AS
        SELECT 
            a.event_id,
            e.name as event_name,
            COUNT(*) as total_assignments,
            COUNT(*) FILTER (WHERE a.is_completed) as completed_assignments,
            ROUND(AVG(a.quality_score), 4) as average_quality_score,
            ROUND(STDDEV(a.quality_score), 4) as quality_score_stddev,
            ROUND(
                100.0 * COUNT(*) FILTER (WHERE a.is_completed) / COUNT(*), 2
            ) as completion_percentage
        FROM assignments a
        JOIN events e ON a.event_id = e.id
        GROUP BY a.event_id, e.name`,

		`CREATE VIEW voting_progress AS
        SELECT 
            v.event_id,
            e.name as event_name,
            COUNT(DISTINCT v.voter_id) as active_voters,
            COUNT(*) as total_votes,
            COUNT(DISTINCT v.attachment_id) as attachments_with_votes,
            ROUND(AVG(v.rank_position::DECIMAL), 2) as average_rank,
            ROUND(AVG(v.score), 2) as average_score,
            ROUND(STDDEV(v.score), 2) as score_stddev,
            ROUND(AVG(v.confidence), 3) as average_confidence,
            COUNT(*) FILTER (WHERE v.is_quality_vote = TRUE) as quality_votes,
            ROUND(
                100.0 * COUNT(*) FILTER (WHERE v.is_quality_vote = TRUE) / COUNT(*), 2
            ) as quality_vote_percentage
        FROM votes v
        JOIN events e ON v.event_id = e.id
        GROUP BY v.event_id, e.name`,

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

		`CREATE VIEW quality_assessment AS
        SELECT 
            a.event_id,
            e.name as event_name,
            a.participant_id,
            u.name as participant_name,
            u.email as participant_email,
            a.quality_score,
            CASE 
                WHEN a.quality_score >= vc.quality_good_threshold THEN 'GOOD'
                WHEN a.quality_score <= vc.quality_bad_threshold THEN 'BAD'
                ELSE 'NEUTRAL'
            END as quality_classification,
            COUNT(v.id) as votes_submitted,
            ROUND(AVG(v.confidence), 3) as average_confidence,
            ROUND(AVG(v.evaluation_time_seconds)::DECIMAL / 60, 1) as avg_evaluation_time_minutes,
            a.is_completed as assignment_completed,
            a.completed_at
        FROM assignments a
        JOIN events e ON a.event_id = e.id
        JOIN users u ON a.participant_id = u.id
        LEFT JOIN voting_configurations vc ON a.event_id = vc.event_id
        LEFT JOIN votes v ON a.id = v.assignment_id
        GROUP BY a.event_id, e.name, a.participant_id, u.name, u.email, a.quality_score, 
                 vc.quality_good_threshold, vc.quality_bad_threshold, a.is_completed, a.completed_at`,
	}

	for _, viewSQL := range views {
		if err := db.Exec(viewSQL).Error; err != nil {
			return err
		}
	}

	comments := []string{
		"COMMENT ON TABLE users IS 'System users (administrators and participants in voting events)'",
		"COMMENT ON TABLE events IS 'Voting events for telescope time allocation and other distributed evaluation processes'",
		"COMMENT ON TABLE event_participants IS 'Many-to-many relationship between events and participating users'",
		"COMMENT ON TABLE attachments IS 'Proposals/files submitted for evaluation (set F in mathematical notation)'",
		"COMMENT ON TABLE voting_configurations IS 'Mathematical parameters: m (attachments per evaluator), Q_good, Q_bad, n (adjustment magnitude)'",
		"COMMENT ON TABLE assignments IS 'Distributed assignment function A: P → 2^F mapping participants to subsets of files'",
		"COMMENT ON TABLE votes IS 'Individual rankings R_i: A(p_i) → {1, 2, ..., m} with calculated Borda scores'",
		"COMMENT ON TABLE voting_results IS 'Final Modified Borda Count (MBC) calculations and global ranking G'",

		"COMMENT ON COLUMN voting_configurations.attachments_per_evaluator IS 'Parameter m: number of attachments each evaluator reviews (recommended: m ≥ 2⌈log₂(k)⌉)'",
		"COMMENT ON COLUMN voting_configurations.quality_good_threshold IS 'Parameter Q_good: threshold for identifying high-quality evaluators'",
		"COMMENT ON COLUMN voting_configurations.quality_bad_threshold IS 'Parameter Q_bad: threshold for identifying low-quality evaluators'",
		"COMMENT ON COLUMN voting_configurations.adjustment_magnitude IS 'Parameter n: magnitude of rank adjustments in incentive system (n ≤ √k)'",
		"COMMENT ON COLUMN assignments.attachment_ids IS 'Set A(p_i): attachments assigned to participant p_i for evaluation'",
		"COMMENT ON COLUMN assignments.quality_score IS 'Q_i: calculated quality score for participant p_i based on consensus alignment'",
		"COMMENT ON COLUMN votes.rank_position IS 'Individual rank r ∈ {1, 2, ..., m} assigned by evaluator (1 = best)'",
		"COMMENT ON COLUMN votes.score IS 'Calculated Borda score: (m - r + 1) × 100/m'",
		"COMMENT ON COLUMN voting_results.global_ranking IS 'Global ranking G sorted by Modified Borda Count scores'",
	}

	for _, commentSQL := range comments {
		db.Exec(commentSQL) // Don't fail if comments can't be added
	}

	return nil
}

// migration005Down drops analytical views
func migration005Down(db *gorm.DB) error {
	views := []string{
		"quality_assessment",
		"system_validation",
		"voting_progress",
		"assignment_statistics",
	}

	for _, view := range views {
		if err := db.Exec("DROP VIEW IF EXISTS " + view + " CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}
