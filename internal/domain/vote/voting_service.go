package vote

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
)

// VotingService implements the distributed voting system
type VotingService struct {
	voteRepo       VoteRepository
	attachmentRepo AttachmentRepository
	userRepo       UserRepository
}

func NewVotingService(voteRepo VoteRepository, attachmentRepo AttachmentRepository, userRepo UserRepository) *VotingService {
	return &VotingService{
		voteRepo:       voteRepo,
		attachmentRepo: attachmentRepo,
		userRepo:       userRepo,
	}
}

// GenerateAssignments implements the assignment algorithm A: P → 2^F
func (vs *VotingService) GenerateAssignments(eventID uuid.UUID, participants []uuid.UUID, attachments []uuid.UUID, config *VotingConfiguration) ([]*Assignment, error) {
	n := len(participants)
	k := len(attachments)
	m := config.AttachmentsPerEvaluator

	if m > k {
		return nil, errors.New("attachments per evaluator (m) cannot exceed total attachments (k)")
	}

	recommendedM := int(math.Ceil(2 * math.Log2(float64(k))))
	if m < recommendedM {
		return nil, fmt.Errorf("recommended minimum attachments per evaluator is %d for %d total attachments", recommendedM, k)
	}

	assignments := make([]*Assignment, n)
	assignmentMatrix := make([][]bool, n)
	for i := range assignmentMatrix {
		assignmentMatrix[i] = make([]bool, k)
	}

	evaluationsPerAttachment := make([]int, k)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Phase 1: Ensure minimum evaluations per file
	for attachmentIdx := range k {
		participantIndices := rng.Perm(n)
		for evalCount := 0; evalCount < config.MinEvaluationsPerFile && evalCount < n; evalCount++ {
			participantIdx := participantIndices[evalCount]

			// Check if participant can evaluate this attachment (conflict of interest)
			if !vs.hasConflictOfInterest(participants[participantIdx], attachments[attachmentIdx]) {
				assignmentMatrix[participantIdx][attachmentIdx] = true
				evaluationsPerAttachment[attachmentIdx]++
			}
		}
	}

	// Phase 2: Complete assignments to reach exactly m attachments per participant
	for participantIdx := range n {
		currentAssignments := 0
		for attachmentIdx := range k {
			if assignmentMatrix[participantIdx][attachmentIdx] {
				currentAssignments++
			}
		}

		attachmentIndices := rng.Perm(k)
		for _, attachmentIdx := range attachmentIndices {
			if currentAssignments >= m {
				break
			}

			if !assignmentMatrix[participantIdx][attachmentIdx] &&
				!vs.hasConflictOfInterest(participants[participantIdx], attachments[attachmentIdx]) {
				assignmentMatrix[participantIdx][attachmentIdx] = true
				evaluationsPerAttachment[attachmentIdx]++
				currentAssignments++
			}
		}

		var assignedAttachments []uuid.UUID
		for attachmentIdx := range k {
			if assignmentMatrix[participantIdx][attachmentIdx] {
				assignedAttachments = append(assignedAttachments, attachments[attachmentIdx])
			}
		}

		assignments[participantIdx] = NewAssignment(eventID, participants[participantIdx], assignedAttachments)
	}

	return assignments, nil
}

// CalculateModifiedBordaCount implements the MBC formula
func (vs *VotingService) CalculateModifiedBordaCount(eventID uuid.UUID, config *VotingConfiguration) (*VotingResults, error) {
	// Validate configuration parameters
	if config == nil {
		return nil, errors.New("voting configuration is required")
	}

	if config.AttachmentsPerEvaluator < 2 {
		return nil, fmt.Errorf("invalid configuration: attachments_per_evaluator must be at least 2, got %d", config.AttachmentsPerEvaluator)
	}

	votes, err := vs.voteRepo.GetByEventID(eventID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get votes: %w", err)
	}

	if len(votes) == 0 {
		return nil, errors.New("no votes found for this event")
	}

	attachments, err := vs.attachmentRepo.GetByEventID(eventID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}

	if len(attachments) == 0 {
		return nil, errors.New("no attachments found for this event")
	}

	m := config.AttachmentsPerEvaluator

	// Safety check to prevent division by zero
	if m < 2 {
		return nil, fmt.Errorf("attachments per evaluator must be at least 2, got %d", m)
	}

	votesByAttachment := make(map[uuid.UUID][]*Vote)
	for _, vote := range votes {
		votesByAttachment[vote.AttachmentID] = append(votesByAttachment[vote.AttachmentID], vote)
	}

	var results []AttachmentResult
	for _, attachment := range attachments {
		attachmentVotes := votesByAttachment[attachment.GetID()]

		// Calculate MBC(f_j) = (1/(m(m-1))) * Σ(m - R_i(f_j))
		var mbcSum float64
		voteCount := len(attachmentVotes)

		for _, vote := range attachmentVotes {
			// Convert rank to Borda points: 1st place = m-1 points, last place = 0 points
			bordaPoints := float64(m - vote.RankPosition)
			mbcSum += bordaPoints
		}

		// Normalize by m(m-1) to ensure 0 ≤ MBC ≤ 1
		mbcScore := mbcSum / (float64(m) * float64(m-1))

		// Calculate average rank for additional insight
		var averageRank float64
		if voteCount > 0 {
			var rankSum int
			for _, vote := range attachmentVotes {
				rankSum += vote.RankPosition
			}
			averageRank = float64(rankSum) / float64(voteCount)
		}

		results = append(results, AttachmentResult{
			AttachmentID:    attachment.GetID(),
			Filename:        attachment.GetOriginalName(),
			ParticipantID:   attachment.GetParticipantID(),
			ParticipantName: "", // Will be populated later
			MBCScore:        mbcScore,
			VoteCount:       voteCount,
			AverageRank:     averageRank,
		})
	}

	// Sort by MBC score (highest first) to create global ranking G
	// In case of ties, use vote count as tiebreaker for deterministic ordering
	sort.Slice(results, func(i, j int) bool {
		if results[i].MBCScore == results[j].MBCScore {
			// More votes wins in case of tie
			if results[i].VoteCount == results[j].VoteCount {
				// If still tied, use attachment ID for deterministic order
				return results[i].AttachmentID.String() < results[j].AttachmentID.String()
			}
			return results[i].VoteCount > results[j].VoteCount
		}
		return results[i].MBCScore > results[j].MBCScore
	})

	for i := range results {
		results[i].GlobalRank = i + 1
	}

	// Calculate participant quality scores Q_i
	participantQualities, err := vs.calculateParticipantQualities(eventID, results, m)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate participant qualities: %w", err)
	}

	// Count total votes
	totalVotes := 0
	for _, result := range results {
		totalVotes += result.VoteCount
	}

	// Apply incentive system to create adjusted ranking G'
	adjustedResults := vs.applyIncentiveSystem(results, participantQualities, config)

	// Create voting results object
	votingResults := &VotingResults{
		ID:                      uuid.New(),
		EventID:                 eventID,
		GlobalRanking:           results,
		ParticipantQualities:    participantQualities,
		AdjustedRanking:         adjustedResults,
		TotalParticipants:       len(participantQualities),
		TotalVotes:              totalVotes,
		AttachmentsPerEvaluator: m,
		CalculatedAt:            time.Now(),
	}

	return votingResults, nil
}

// calculateParticipantQualities implements Q_i = 1 - (2/(m(m-1))) * Σ|R_i(f_j) - RelativeRank_G(f_j, A(p_i))|
func (vs *VotingService) calculateParticipantQualities(eventID uuid.UUID, globalResults []AttachmentResult, m int) (map[string]float64, error) {
	assignments, err := vs.voteRepo.GetAssignmentsByEventID(eventID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	globalRankMap := make(map[uuid.UUID]int)
	for _, result := range globalResults {
		globalRankMap[result.AttachmentID] = result.GlobalRank
	}

	qualities := make(map[string]float64)

	for _, assignment := range assignments {
		// Include all participants, even if not completed
		// Incomplete assignments get quality score of 0
		participantID := assignment.ParticipantID.String()

		if !assignment.IsCompleted {
			qualities[participantID] = 0.0
			continue
		}

		participantVotes, err := vs.voteRepo.GetByVoterID(participantID)
		if err != nil {
			// If we can't get votes, assume quality of 0
			qualities[participantID] = 0.0
			continue
		}

		var eventVotes []*Vote
		for _, vote := range participantVotes {
			if vote.EventID == eventID {
				eventVotes = append(eventVotes, vote)
			}
		}

		// Skip if no votes found
		if len(eventVotes) == 0 {
			qualities[participantID] = 0.0
			continue
		}

		relativeRanks := vs.calculateRelativeRanks(assignment.GetAttachmentUUIDs(), globalRankMap)

		// Calculate quality score
		var deviationSum float64
		validVotes := 0
		for _, vote := range eventVotes {
			relativeRank, exists := relativeRanks[vote.AttachmentID]
			if exists {
				deviation := math.Abs(float64(vote.RankPosition - relativeRank))
				deviationSum += deviation
				validVotes++
			}
		}

		// If no valid votes, quality is 0
		if validVotes == 0 {
			qualities[participantID] = 0.0
			continue
		}

		// Q_i = 1 - (2/(m(m-1))) * Σ|R_i(f_j) - RelativeRank_G(f_j, A(p_i))|
		quality := 1.0 - (2.0/(float64(m)*float64(m-1)))*deviationSum

		// Clamp quality to [0, 1]
		if quality < 0 {
			quality = 0
		}
		if quality > 1 {
			quality = 1
		}

		qualities[participantID] = quality
	}

	return qualities, nil
}

// calculateRelativeRanks computes the relative ranking within a subset A(p_i)
func (vs *VotingService) calculateRelativeRanks(assignedAttachments []uuid.UUID, globalRankMap map[uuid.UUID]int) map[uuid.UUID]int {
	type attachmentRank struct {
		ID         uuid.UUID
		GlobalRank int
	}

	var attachmentRanks []attachmentRank
	for _, attachmentID := range assignedAttachments {
		if globalRank, exists := globalRankMap[attachmentID]; exists {
			attachmentRanks = append(attachmentRanks, attachmentRank{
				ID:         attachmentID,
				GlobalRank: globalRank,
			})
		}
	}

	sort.Slice(attachmentRanks, func(i, j int) bool {
		return attachmentRanks[i].GlobalRank < attachmentRanks[j].GlobalRank
	})

	relativeRanks := make(map[uuid.UUID]int)
	for i, ar := range attachmentRanks {
		relativeRanks[ar.ID] = i + 1
	}

	return relativeRanks
}

// applyIncentiveSystem implements the adjustment Δ(owner(f_j))
func (vs *VotingService) applyIncentiveSystem(results []AttachmentResult, qualities map[string]float64, config *VotingConfiguration) []AttachmentResult {
	adjustedResults := make([]AttachmentResult, len(results))
	copy(adjustedResults, results)

	for i := range adjustedResults {
		participantID := adjustedResults[i].ParticipantID.String()
		quality, exists := qualities[participantID]

		if !exists {
			adjustedResults[i].AdjustedRank = adjustedResults[i].GlobalRank
			continue
		}

		adjustment := 0
		if quality >= config.QualityGoodThreshold {
			// Good evaluator: bonus (negative adjustment improves rank)
			adjustment = -config.AdjustmentMagnitude
		} else if quality <= config.QualityBadThreshold {
			// Bad evaluator: penalty (positive adjustment worsens rank)
			adjustment = config.AdjustmentMagnitude
		}

		// Apply adjustment (with bounds checking)
		adjustedRank := min(max(adjustedResults[i].GlobalRank+adjustment, 1), len(results))

		adjustedResults[i].AdjustedRank = adjustedRank
	}

	// Sort by adjusted rank
	sort.Slice(adjustedResults, func(i, j int) bool {
		if adjustedResults[i].AdjustedRank == adjustedResults[j].AdjustedRank {
			// Tiebreaker: use MBC score
			return adjustedResults[i].MBCScore > adjustedResults[j].MBCScore
		}
		return adjustedResults[i].AdjustedRank < adjustedResults[j].AdjustedRank
	})

	// Reassign consecutive ranks after sorting
	for i := range adjustedResults {
		adjustedResults[i].AdjustedRank = i + 1
	}

	return adjustedResults
}

// hasConflictOfInterest checks if a participant has a conflict with an attachment
func (vs *VotingService) hasConflictOfInterest(participantID, attachmentID uuid.UUID) bool {
	attachment, err := vs.attachmentRepo.GetByID(attachmentID.String())
	if err != nil {
		return true // Err on the side of caution
	}

	return attachment.GetParticipantID() == participantID
}

// ValidateVotingConfiguration ensures the mathematical parameters are valid
func (vs *VotingService) ValidateVotingConfiguration(config *VotingConfiguration, totalAttachments, totalParticipants int) error {
	m := config.AttachmentsPerEvaluator
	k := totalAttachments
	n := totalParticipants

	if m <= 0 {
		return errors.New("attachments per evaluator must be positive")
	}
	if m > k {
		return errors.New("attachments per evaluator cannot exceed total attachments")
	}
	if config.QualityGoodThreshold <= config.QualityBadThreshold {
		return errors.New("good quality threshold must be higher than bad quality threshold")
	}
	if config.QualityGoodThreshold > 1.0 || config.QualityBadThreshold < 0.0 {
		return errors.New("quality thresholds must be in [0, 1] range")
	}

	recommendedM := int(math.Ceil(2 * math.Log2(float64(k))))
	if m < recommendedM {
		return fmt.Errorf("recommended minimum m is %d for optimal convergence with %d attachments", recommendedM, k)
	}

	totalEvaluations := n * m
	minRequiredEvaluations := k * config.MinEvaluationsPerFile
	if totalEvaluations < minRequiredEvaluations {
		return fmt.Errorf("insufficient total evaluations: need %d, have %d", minRequiredEvaluations, totalEvaluations)
	}

	return nil
}
