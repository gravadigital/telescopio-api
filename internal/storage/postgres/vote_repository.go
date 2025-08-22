package postgres

import (
	"errors"

	"github.com/gravadigital/telescopio-api/internal/domain/vote"
)

// InMemoryVoteRepository is a temporary in-memory implementation
// TODO: Replace with actual PostgreSQL implementation
type InMemoryVoteRepository struct {
	votes map[string]*vote.Vote
}

func NewInMemoryVoteRepository() *InMemoryVoteRepository {
	return &InMemoryVoteRepository{
		votes: make(map[string]*vote.Vote),
	}
}

func (r *InMemoryVoteRepository) Create(vote *vote.Vote) error {
	r.votes[vote.ID] = vote
	return nil
}

func (r *InMemoryVoteRepository) GetByID(id string) (*vote.Vote, error) {
	vote, exists := r.votes[id]
	if !exists {
		return nil, errors.New("vote not found")
	}
	return vote, nil
}

func (r *InMemoryVoteRepository) GetByEventID(eventID string) ([]*vote.Vote, error) {
	var votes []*vote.Vote
	for _, vote := range r.votes {
		if vote.EventID == eventID {
			votes = append(votes, vote)
		}
	}
	return votes, nil
}

func (r *InMemoryVoteRepository) GetByVoterID(voterID string) ([]*vote.Vote, error) {
	var votes []*vote.Vote
	for _, vote := range r.votes {
		if vote.VoterID == voterID {
			votes = append(votes, vote)
		}
	}
	return votes, nil
}

func (r *InMemoryVoteRepository) GetByAttachmentID(attachmentID string) ([]*vote.Vote, error) {
	var votes []*vote.Vote
	for _, vote := range r.votes {
		if vote.AttachmentID == attachmentID {
			votes = append(votes, vote)
		}
	}
	return votes, nil
}

func (r *InMemoryVoteRepository) HasVoted(eventID, voterID string) (bool, error) {
	for _, vote := range r.votes {
		if vote.EventID == eventID && vote.VoterID == voterID {
			return true, nil
		}
	}
	return false, nil
}

func (r *InMemoryVoteRepository) GetEventResults(eventID string) (map[string]int, error) {
	results := make(map[string]int)

	for _, vote := range r.votes {
		if vote.EventID == eventID {
			results[vote.AttachmentID]++
		}
	}

	return results, nil
}
