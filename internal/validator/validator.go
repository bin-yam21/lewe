package validator

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Errors collects validation failures keyed by field name.
type Errors map[string]string

// Error implements the error interface for Errors.
func (e Errors) Error() string {
	parts := make([]string, 0, len(e))
	for field, msg := range e {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(parts, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e Errors) HasErrors() bool {
	return len(e) > 0
}

// ValidateEmail checks that the email is non-empty and matches a basic pattern.
func ValidateEmail(errs Errors, field, value string) {
	if strings.TrimSpace(value) == "" {
		errs[field] = "is required"
		return
	}
	if !emailRegex.MatchString(value) {
		errs[field] = "is not a valid email address"
	}
}

// ValidateRequired checks that a string field is non-empty.
func ValidateRequired(errs Errors, field, value string) {
	if strings.TrimSpace(value) == "" {
		errs[field] = "is required"
	}
}

// ValidateMinLength checks that a string meets a minimum length.
func ValidateMinLength(errs Errors, field, value string, min int) {
	if len(value) < min {
		errs[field] = fmt.Sprintf("must be at least %d characters", min)
	}
}

// ValidateMaxLength checks that a string does not exceed a maximum length.
func ValidateMaxLength(errs Errors, field, value string, max int) {
	if len(value) > max {
		errs[field] = fmt.Sprintf("must be at most %d characters", max)
	}
}
