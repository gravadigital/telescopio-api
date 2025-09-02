package participant

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents user roles in the system
type Role string

const (
	RoleAdmin       Role = "admin"
	RoleParticipant Role = "participant"
)

// String returns the string representation of the role
func (r Role) String() string {
	return string(r)
}

// IsValid checks if the role is valid
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleParticipant
}

// User represents a system user (admin or participant)
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string    `json:"name" gorm:"not null"`
	LastName  string    `json:"lastname"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Role      Role      `json:"role" gorm:"type:varchar(20);not null;default:'participant'"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName overrides the table name used by GORM
func (User) TableName() string {
	return "users"
}

// BeforeCreate sets a UUID before creating the record
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// NewUser creates a new user with default values
func NewUser(name, lastName, email string, role Role) *User {
	return &User{
		ID:        uuid.New(),
		Name:      name,
		LastName:  lastName,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// NewParticipant creates a new participant user
func NewParticipant(name, lastName, email string) *User {
	return NewUser(name, lastName, email, RoleParticipant)
}

// NewAdmin creates a new admin user
func NewAdmin(name, lastName, email string) *User {
	return NewUser(name, lastName, email, RoleAdmin)
}

// GetFullName returns the full name of the user
func (u *User) GetFullName() string {
	if u.LastName == "" {
		return u.Name
	}
	return fmt.Sprintf("%s %s", u.Name, u.LastName)
}

// UpdateRole safely updates the user role with validation
func (u *User) UpdateRole(newRole Role) error {
	if !newRole.IsValid() {
		return fmt.Errorf("invalid role: %s", newRole)
	}
	u.Role = newRole
	u.UpdatedAt = time.Now()
	return nil
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsParticipant checks if the user has participant role
func (u *User) IsParticipant() bool {
	return u.Role == RoleParticipant
}

// Validate checks if the user data is valid
func (u *User) Validate() error {
	if u.Name == "" {
		return fmt.Errorf("name is required")
	}
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	// TODO: Add email format validation using regex
	// Example: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !u.Role.IsValid() {
		return fmt.Errorf("role must be 'admin' or 'participant', got: %s", u.Role)
	}
	return nil
}

// Implement common.UserInterface to avoid circular imports
func (u *User) GetID() uuid.UUID {
	return u.ID
}

func (u *User) GetName() string {
	return u.Name
}

// TODO: Permission management methods by role
// These methods should be implemented to handle role-based permissions

// CanCreateEvent checks if user can create events
// TODO: Implement permission logic based on role
func (u *User) CanCreateEvent() bool {
	// TODO: Define which roles can create events
	return u.IsAdmin() // Placeholder logic
}

// CanManageParticipants checks if user can manage event participants
// TODO: Implement permission logic for participant management
func (u *User) CanManageParticipants() bool {
	// TODO: Define which roles can manage participants
	return u.IsAdmin() // Placeholder logic
}

// CanVote checks if user can participate in voting
// TODO: Implement voting permission logic
func (u *User) CanVote() bool {
	// TODO: Define which roles can vote (probably both admin and participant)
	return u.IsParticipant() || u.IsAdmin() // Placeholder logic
}

// CanViewResults checks if user can view voting results
// TODO: Implement result viewing permission logic
func (u *User) CanViewResults() bool {
	// TODO: Define which roles can view results
	return true // Placeholder: everyone can view results
}

// CanModifyVotingConfiguration checks if user can modify voting settings
// TODO: Implement voting configuration permission logic
func (u *User) CanModifyVotingConfiguration() bool {
	// TODO: Define which roles can modify voting configuration
	return u.IsAdmin() // Placeholder logic
}

// HasPermission checks if user has a specific permission
// TODO: Implement a comprehensive permission system
func (u *User) HasPermission(permission string) bool {
	// TODO: Implement permission checking logic
	// This could integrate with a more sophisticated RBAC system
	switch permission {
	case "create_event":
		return u.CanCreateEvent()
	case "manage_participants":
		return u.CanManageParticipants()
	case "vote":
		return u.CanVote()
	case "view_results":
		return u.CanViewResults()
	case "modify_voting_config":
		return u.CanModifyVotingConfiguration()
	default:
		return false // TODO: Define default permission behavior
	}
}
