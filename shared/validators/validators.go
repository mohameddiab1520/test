package validators

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (e ValidationErrors) Error() string {
	messages := make([]string, len(e))
	for i, err := range e {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}

	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}

	return nil
}

// ValidatePassword validates a password
func ValidatePassword(password string) error {
	if password == "" {
		return &ValidationError{Field: "password", Message: "password is required"}
	}

	if len(password) < 8 {
		return &ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}

	if len(password) > 128 {
		return &ValidationError{Field: "password", Message: "password must be less than 128 characters"}
	}

	return nil
}

// ValidateUUID validates a UUID
func ValidateUUID(id string) error {
	if id == "" {
		return &ValidationError{Field: "id", Message: "id is required"}
	}

	if !uuidRegex.MatchString(id) {
		return &ValidationError{Field: "id", Message: "invalid UUID format"}
	}

	return nil
}

// ValidateRequired validates that a field is not empty
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Message: fmt.Sprintf("%s is required", field)}
	}
	return nil
}

// ValidateLength validates string length
func ValidateLength(field, value string, min, max int) error {
	length := len(value)

	if length < min {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s must be at least %d characters", field, min),
		}
	}

	if max > 0 && length > max {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s must be less than %d characters", field, max),
		}
	}

	return nil
}

// ValidateEnum validates that a value is in a set of allowed values
func ValidateEnum(field, value string, allowedValues []string) error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return &ValidationError{
		Field:   field,
		Message: fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowedValues, ", ")),
	}
}

// ValidateRange validates that a number is within a range
func ValidateRange(field string, value, min, max int) error {
	if value < min {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s must be at least %d", field, min),
		}
	}

	if max > 0 && value > max {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s must be less than %d", field, max),
		}
	}

	return nil
}
