package validation

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ValidateRequired valida que un campo no esté vacío
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(fieldName + " is required")
	}
	return nil
}

// ValidateMinLength valida la longitud mínima de un string
func ValidateMinLength(value string, minLength int, fieldName string) error {
	if utf8.RuneCountInString(value) < minLength {
		return errors.New(fieldName + " must be at least " + string(rune(minLength)) + " characters long")
	}
	return nil
}

// ValidateMaxLength valida la longitud máxima de un string
func ValidateMaxLength(value string, maxLength int, fieldName string) error {
	if utf8.RuneCountInString(value) > maxLength {
		return errors.New(fieldName + " must be at most " + string(rune(maxLength)) + " characters long")
	}
	return nil
}

// ValidateUUID valida que un string sea un UUID válido
func ValidateUUID(value, fieldName string) error {
	if _, err := uuid.Parse(value); err != nil {
		return errors.New(fieldName + " must be a valid UUID")
	}
	return nil
}

// ValidateEmail valida formato básico de email
func ValidateEmail(email string) error {
	if !strings.Contains(email, "@") {
		return errors.New("email must have a valid format")
	}
	return nil
}

// ValidateDateRange valida que una fecha esté en el rango correcto
func ValidateDateRange(startDate, endDate time.Time) error {
	if endDate.Before(startDate) {
		return errors.New("end date must be after start date")
	}

	if startDate.Before(time.Now().Add(-24 * time.Hour)) {
		return errors.New("start date cannot be in the past")
	}

	return nil
}

// EventValidation contiene validaciones específicas para eventos
type EventValidation struct{}

// ValidateEventName valida el nombre de un evento
func (v EventValidation) ValidateEventName(name string) error {
	if err := ValidateRequired(name, "name"); err != nil {
		return err
	}
	if err := ValidateMinLength(name, 3, "name"); err != nil {
		return err
	}
	if err := ValidateMaxLength(name, 100, "name"); err != nil {
		return err
	}
	return nil
}

// ValidateEventDescription valida la descripción de un evento
func (v EventValidation) ValidateEventDescription(description string) error {
	if err := ValidateRequired(description, "description"); err != nil {
		return err
	}
	if err := ValidateMaxLength(description, 1000, "description"); err != nil {
		return err
	}
	return nil
}

// UserValidation contiene validaciones específicas para usuarios
type UserValidation struct{}

// ValidateUserName valida el nombre de un usuario
func (v UserValidation) ValidateUserName(name string) error {
	if err := ValidateRequired(name, "name"); err != nil {
		return err
	}
	if err := ValidateMinLength(name, 2, "name"); err != nil {
		return err
	}
	if err := ValidateMaxLength(name, 50, "name"); err != nil {
		return err
	}
	return nil
}

// ValidateUserEmail valida el email de un usuario
func (v UserValidation) ValidateUserEmail(email string) error {
	if err := ValidateRequired(email, "email"); err != nil {
		return err
	}
	if err := ValidateEmail(email); err != nil {
		return err
	}
	return nil
}
