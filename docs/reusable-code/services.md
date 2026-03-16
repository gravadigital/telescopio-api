# Reusable Code — Services

## GoogleAuthService

**Location:** `internal/handlers/google_auth_service.go`

**Description:** Business logic for Google OAuth authentication. Validates Google id_tokens against the Google tokeninfo API and resolves/links users in the local database.

**Functions:**

```go
// verifyGoogleToken calls Google tokeninfo API to validate an id_token.
// Returns ErrInvalidGoogleToken for bad/expired tokens, ErrGoogleAPIUnavailable for network errors.
func verifyGoogleToken(token, clientID string) (*GoogleProfile, error)

// resolveUser looks up a user by google_id then by email.
// Auto-links google_id if user found by email without one.
// Returns UserResolution with Status "existing_user" or "new_user".
func resolveUser(profile *GoogleProfile, userRepo postgres.UserRepository) (*UserResolution, error)
```

**Sentinel Errors:**

```go
var ErrInvalidGoogleToken    = errors.New("invalid or expired Google token")
var ErrGoogleAPIUnavailable  = errors.New("Google API unavailable")
```

**Types:**

```go
type GoogleProfile struct {
    GoogleID string
    Email    string
    Name     string
}

type UserResolution struct {
    Status string // "new_user" or "existing_user"
    User   *participant.User
}
```

**Usage example:**

```go
profile, err := verifyGoogleToken(idToken, cfg.Google.ClientID)
if errors.Is(err, ErrInvalidGoogleToken) {
    // return 401
}
if errors.Is(err, ErrGoogleAPIUnavailable) {
    // return 500
}

resolution, err := resolveUser(profile, userRepo)
if resolution.Status == "existing_user" {
    // generate JWT and return user
}
// else: prompt for username and call /register
```

**Dependencies:** `net/http`, `encoding/json`, `internal/domain/participant`, `internal/storage/postgres`
