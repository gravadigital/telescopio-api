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
