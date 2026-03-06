# Reusable Code Index — telescopio-api

## Summary

| Category | Count |
|----------|-------|
| Services | 1 |
| Utils/Helpers | 2 |
| Types | 3 |

---

## Services

- **GoogleAuthService** (`internal/handlers/google_auth_service.go`) - Google OAuth token validation and user resolution logic

## Utils/Helpers

- **GetByGoogleID** (`internal/storage/postgres/user_repository.go`) - Retrieve user by Google OAuth ID from PostgreSQL
- **UsernameExists** (`internal/storage/postgres/user_repository.go`) - Check if a username (Name field) is already taken in PostgreSQL

## Types

- **GoogleProfile** (`internal/handlers/google_auth_service.go`) - Holds user profile data extracted from a validated Google id_token
- **UserResolution** (`internal/handlers/google_auth_service.go`) - Result of resolving a Google profile against the local user store
- **ErrInvalidGoogleToken / ErrGoogleAPIUnavailable** (`internal/handlers/google_auth_service.go`) - Sentinel errors for Google token validation failures
