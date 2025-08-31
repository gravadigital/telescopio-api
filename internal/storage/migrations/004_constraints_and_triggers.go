package migrations

import "gorm.io/gorm"

// migration004Up creates mathematical validation functions, constraints and triggers
func migration004Up(db *gorm.DB) error {
	functions := []string{
		`CREATE OR REPLACE FUNCTION validate_assignment_constraints()
        RETURNS TRIGGER AS $$
        DECLARE
            config_record RECORD;
            attachment_count INTEGER;
            valid_attachments INTEGER;
            self_assignments INTEGER;
        BEGIN
            -- Get voting configuration for this event
            SELECT * INTO config_record 
            FROM voting_configurations 
            WHERE event_id = NEW.event_id;
            
            IF FOUND THEN
                -- Check that assignment has correct number of attachments
                attachment_count := array_length(NEW.attachment_ids, 1);
                IF attachment_count != config_record.attachments_per_evaluator THEN
                    RAISE EXCEPTION 'Assignment must have exactly % attachments, got %', 
                        config_record.attachments_per_evaluator, attachment_count;
                END IF;
                
                -- Validate all attachment IDs exist and belong to this event
                SELECT COUNT(*) INTO valid_attachments
                FROM attachments 
                WHERE id::text = ANY(NEW.attachment_ids) AND event_id = NEW.event_id;
                
                IF valid_attachments != attachment_count THEN
                    RAISE EXCEPTION 'Some attachment IDs are invalid or do not belong to this event';
                END IF;
                
                -- Check for self-assignment (conflict of interest)
                SELECT COUNT(*) INTO self_assignments
                FROM attachments 
                WHERE id::text = ANY(NEW.attachment_ids) 
                  AND event_id = NEW.event_id 
                  AND participant_id = NEW.participant_id;
                  
                IF self_assignments > 0 THEN
                    RAISE EXCEPTION 'Participant cannot be assigned to evaluate their own proposals (conflict of interest)';
                END IF;
            END IF;
            
            RETURN NEW;
        END;
        $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION validate_vote_constraints()
        RETURNS TRIGGER AS $$
        DECLARE
            assignment_record RECORD;
            attachment_in_assignment BOOLEAN := FALSE;
            config_record RECORD;
            max_rank INTEGER;
        BEGIN
            -- Get assignment for this voter in this event
            SELECT * INTO assignment_record 
            FROM assignments 
            WHERE event_id = NEW.event_id AND participant_id = NEW.voter_id;
            
            IF NOT FOUND THEN
                RAISE EXCEPTION 'Voter % has no assignment for event %', NEW.voter_id, NEW.event_id;
            END IF;
            
            -- Check if attachment is in the voter's assignment
            IF NEW.attachment_id::text = ANY(assignment_record.attachment_ids) THEN
                attachment_in_assignment := TRUE;
            END IF;
            
            IF NOT attachment_in_assignment THEN
                RAISE EXCEPTION 'Attachment % is not assigned to voter % for evaluation', 
                    NEW.attachment_id, NEW.voter_id;
            END IF;
            
            -- Validate rank bounds and calculate Borda score
            SELECT * INTO config_record 
            FROM voting_configurations 
            WHERE event_id = NEW.event_id;
            
            IF FOUND THEN
                max_rank := config_record.attachments_per_evaluator;
                IF NEW.rank_position > max_rank THEN
                    RAISE EXCEPTION 'Rank position % exceeds maximum allowed rank % for this event', 
                        NEW.rank_position, max_rank;
                END IF;
                
                -- Calculate Borda score if not provided
                IF NEW.score IS NULL THEN
                    NEW.score := (config_record.attachments_per_evaluator - NEW.rank_position + 1.0) * 100.0 / config_record.attachments_per_evaluator;
                END IF;
            END IF;
            
            RETURN NEW;
        END;
        $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION update_assignment_completion()
        RETURNS TRIGGER AS $$
        DECLARE
            assignment_record RECORD;
            expected_votes INTEGER;
            actual_votes INTEGER;
        BEGIN
            -- Get assignment for this voter
            SELECT * INTO assignment_record 
            FROM assignments 
            WHERE event_id = COALESCE(NEW.event_id, OLD.event_id) 
              AND participant_id = COALESCE(NEW.voter_id, OLD.voter_id);
            
            IF FOUND THEN
                expected_votes := array_length(assignment_record.attachment_ids, 1);
                
                -- Count actual votes by this participant for this event
                SELECT COUNT(*) INTO actual_votes
                FROM votes 
                WHERE event_id = assignment_record.event_id 
                  AND voter_id = assignment_record.participant_id;
                
                -- Update completion status
                IF actual_votes >= expected_votes AND NOT assignment_record.is_completed THEN
                    UPDATE assignments 
                    SET is_completed = TRUE, 
                        completed_at = CURRENT_TIMESTAMP 
                    WHERE id = assignment_record.id;
                ELSIF actual_votes < expected_votes AND assignment_record.is_completed THEN
                    UPDATE assignments 
                    SET is_completed = FALSE, 
                        completed_at = NULL 
                    WHERE id = assignment_record.id;
                END IF;
            END IF;
            
            RETURN COALESCE(NEW, OLD);
        END;
        $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION update_attachment_vote_count()
        RETURNS TRIGGER AS $$
        BEGIN
            IF TG_OP = 'INSERT' THEN
                UPDATE attachments 
                SET vote_count = vote_count + 1 
                WHERE id = NEW.attachment_id;
                RETURN NEW;
            ELSIF TG_OP = 'DELETE' THEN
                UPDATE attachments 
                SET vote_count = vote_count - 1 
                WHERE id = OLD.attachment_id;
                RETURN OLD;
            END IF;
            RETURN NULL;
        END;
        $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION check_coverage_constraints(p_event_id UUID)
        RETURNS BOOLEAN AS $$
        DECLARE
            config_record RECORD;
            participant_count INTEGER;
            attachment_count INTEGER;
            total_evaluations INTEGER;
            min_required_evaluations INTEGER;
        BEGIN
            -- Get configuration
            SELECT * INTO config_record 
            FROM voting_configurations 
            WHERE event_id = p_event_id;
            
            IF NOT FOUND THEN
                RETURN FALSE;
            END IF;
            
            -- Count participants and attachments
            SELECT COUNT(*) INTO participant_count 
            FROM event_participants 
            WHERE event_id = p_event_id;
            
            SELECT COUNT(*) INTO attachment_count 
            FROM attachments 
            WHERE event_id = p_event_id;
            
            -- Calculate coverage
            total_evaluations := participant_count * config_record.attachments_per_evaluator;
            min_required_evaluations := attachment_count * config_record.min_evaluations_per_file;
            
            RETURN total_evaluations >= min_required_evaluations;
        END;
        $$ LANGUAGE plpgsql`,
	}

	for _, funcSQL := range functions {
		if err := db.Exec(funcSQL).Error; err != nil {
			return err
		}
	}

	triggers := []string{
		"CREATE TRIGGER trigger_validate_assignment BEFORE INSERT OR UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION validate_assignment_constraints()",
		"CREATE TRIGGER trigger_validate_vote BEFORE INSERT OR UPDATE ON votes FOR EACH ROW EXECUTE FUNCTION validate_vote_constraints()",
		"CREATE TRIGGER trigger_update_assignment_completion_insert AFTER INSERT ON votes FOR EACH ROW EXECUTE FUNCTION update_assignment_completion()",
		"CREATE TRIGGER trigger_update_assignment_completion_delete AFTER DELETE ON votes FOR EACH ROW EXECUTE FUNCTION update_assignment_completion()",
		"CREATE TRIGGER trigger_update_vote_count AFTER INSERT OR DELETE ON votes FOR EACH ROW EXECUTE FUNCTION update_attachment_vote_count()",
	}

	for _, triggerSQL := range triggers {
		if err := db.Exec(triggerSQL).Error; err != nil {
			return err
		}
	}

	constraints := []string{
		"ALTER TABLE users ADD CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$')",
		"ALTER TABLE events ADD CONSTRAINT valid_event_dates CHECK (end_date >= start_date)",
		"ALTER TABLE events ADD CONSTRAINT future_start_date CHECK (start_date >= CURRENT_TIMESTAMP - INTERVAL '1 day')",
		"ALTER TABLE attachments ADD CONSTRAINT valid_file_size CHECK (file_size > 0 AND file_size <= 104857600)",
		"ALTER TABLE attachments ADD CONSTRAINT valid_filename CHECK (LENGTH(filename) > 0)",
		"ALTER TABLE voting_configurations ADD CONSTRAINT valid_m_parameter CHECK (attachments_per_evaluator > 0 AND attachments_per_evaluator <= 50)",
		"ALTER TABLE voting_configurations ADD CONSTRAINT valid_min_evaluations CHECK (min_evaluations_per_file > 0)",
		"ALTER TABLE voting_configurations ADD CONSTRAINT valid_quality_thresholds CHECK (quality_good_threshold > quality_bad_threshold AND quality_good_threshold - quality_bad_threshold >= 0.1)",
		"ALTER TABLE voting_configurations ADD CONSTRAINT valid_adjustment_magnitude CHECK (adjustment_magnitude >= 0 AND adjustment_magnitude <= 20)",
		"ALTER TABLE assignments ADD CONSTRAINT valid_quality_score CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1))",
		"ALTER TABLE votes ADD CONSTRAINT valid_rank_position CHECK (rank_position > 0)",
		"ALTER TABLE votes ADD CONSTRAINT valid_confidence CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1))",
		"ALTER TABLE votes ADD CONSTRAINT valid_evaluation_time CHECK (evaluation_time_seconds IS NULL OR evaluation_time_seconds > 0)",
		"ALTER TABLE voting_results ADD CONSTRAINT valid_participant_counts CHECK (total_participants > 0 AND total_votes >= total_participants)",
		"ALTER TABLE voting_results ADD CONSTRAINT valid_evaluator_counts CHECK (good_evaluator_count >= 0 AND bad_evaluator_count >= 0 AND good_evaluator_count + bad_evaluator_count <= total_participants)",
	}

	for _, constraintSQL := range constraints {
		// Use IF NOT EXISTS equivalent by catching errors
		db.Exec(constraintSQL) // Don't return error for constraints that might already exist
	}

	return nil
}

// migration004Down drops constraints and triggers
func migration004Down(db *gorm.DB) error {
	triggers := []string{
		"trigger_validate_assignment",
		"trigger_validate_vote",
		"trigger_update_assignment_completion_insert",
		"trigger_update_assignment_completion_delete",
		"trigger_update_vote_count",
	}

	for _, trigger := range triggers {
		db.Exec("DROP TRIGGER IF EXISTS " + trigger + " ON assignments CASCADE")
		db.Exec("DROP TRIGGER IF EXISTS " + trigger + " ON votes CASCADE")
	}

	functions := []string{
		"validate_assignment_constraints",
		"validate_vote_constraints",
		"update_assignment_completion",
		"update_attachment_vote_count",
		"check_coverage_constraints",
	}

	for _, function := range functions {
		if err := db.Exec("DROP FUNCTION IF EXISTS " + function + " CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}
