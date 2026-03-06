package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

// ErrInvalidGoogleToken is returned when the Google token is invalid or expired.
var ErrInvalidGoogleToken = errors.New("invalid or expired Google token")

// ErrGoogleAPIUnavailable is returned when the Google API cannot be reached.
var ErrGoogleAPIUnavailable = errors.New("Google API unavailable")

// GoogleProfile holds the user profile extracted from a validated Google token.
type GoogleProfile struct {
	GoogleID string
	Email    string
	Name     string
}

// UserResolution holds the result of resolving a Google profile against the local user store.
type UserResolution struct {
	Status string // "new_user" or "existing_user"
	User   *participant.User
}

// googleTokenInfoResponse maps the relevant fields from the Google tokeninfo endpoint.
type googleTokenInfoResponse struct {
	Sub              string `json:"sub"`
	Email            string `json:"email"`
	Name             string `json:"name"`
	Aud              string `json:"aud"`
	EmailVerified    string `json:"email_verified"`
	ErrorDescription string `json:"error_description"`
}

// verifyGoogleToken calls the Google tokeninfo endpoint to validate an id_token.
// It verifies that the token's aud matches clientID.
// Returns ErrInvalidGoogleToken for bad/expired tokens, ErrGoogleAPIUnavailable for network errors.
func verifyGoogleToken(token, clientID string) (*GoogleProfile, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGoogleAPIUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", ErrGoogleAPIUnavailable, err)
	}

	var info googleTokenInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response: %v", ErrGoogleAPIUnavailable, err)
	}

	if info.ErrorDescription != "" || resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidGoogleToken
	}

	if clientID != "" && info.Aud != clientID {
		return nil, ErrInvalidGoogleToken
	}

	if info.Sub == "" || info.Email == "" {
		return nil, ErrInvalidGoogleToken
	}

	return &GoogleProfile{
		GoogleID: info.Sub,
		Email:    info.Email,
		Name:     info.Name,
	}, nil
}

// resolveUser looks up a user by google_id then by email.
// If found by email without google_id, it links the google_id automatically.
// Returns UserResolution with Status "existing_user" or "new_user".
func resolveUser(profile *GoogleProfile, userRepo postgres.UserRepository) (*UserResolution, error) {
	// 1. Search by google_id
	user, err := userRepo.GetByGoogleID(profile.GoogleID)
	if err == nil {
		return &UserResolution{Status: "existing_user", User: user}, nil
	}
	if err.Error() != "user not found" {
		return nil, fmt.Errorf("failed to look up user by google_id: %w", err)
	}

	// 2. Search by email
	user, err = userRepo.GetByEmail(profile.Email)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "email cannot be empty" {
			return &UserResolution{Status: "new_user"}, nil
		}
		return nil, fmt.Errorf("failed to look up user by email: %w", err)
	}

	// Found by email without google_id — link automatically
	if user.GoogleID == nil {
		googleID := profile.GoogleID
		user.GoogleID = &googleID
		if err := userRepo.Update(user); err != nil {
			return nil, fmt.Errorf("failed to link google_id to existing user: %w", err)
		}
	}

	return &UserResolution{Status: "existing_user", User: user}, nil
}
