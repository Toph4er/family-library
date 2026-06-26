// Package validation provides simple request validation helpers for handlers.
//
// Usage:
//
//	v := validation.New()
//	v.Required("title", title)
//	v.Required("isbn", isbn)
//	if v.HasErrors() {
//	    // render errors however the handler wants (HTML, JSON, etc.)
//	}
package validation

import (
	"fmt"
	"strings"
)

// Error represents a single field-level validation error.
type Error struct {
	Field   string // form field name
	Message string // human-readable message
}

func (e Error) String() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Errors is a collection of validation errors for one request.
type Errors []Error

// HasErrors reports whether any validation errors were collected.
func (errs Errors) HasErrors() bool {
	return len(errs) > 0
}

// First returns the first error message, or "" if none.
func (errs Errors) First() string {
	if len(errs) == 0 {
		return ""
	}
	return errs[0].Message
}

// Add records a validation error for the given field.
func (errs *Errors) Add(field, message string) {
	*errs = append(*errs, Error{Field: field, Message: message})
}

// Required checks that value is non-empty after trimming whitespace.
func (errs *Errors) Required(field, value string) {
	if value == "" {
		errs.Add(field, fmt.Sprintf("%s is required", field))
	}
}

// MaxLength checks that the trimmed value does not exceed max bytes.
func (errs *Errors) MaxLength(field, value string, max int) {
	if len(value) > max {
		errs.Add(field, fmt.Sprintf("%s must be at most %d characters", field, max))
	}
}

// MinValue checks that an integer is >= min.
func (errs *Errors) MinValue(field string, value, min int) {
	if value < min {
		errs.Add(field, fmt.Sprintf("%s must be at least %d", field, min))
	}
}

// MaxValue checks that an integer is <= max.
func (errs *Errors) MaxValue(field string, value, max int) {
	if value > max {
		errs.Add(field, fmt.Sprintf("%s must be at most %d", field, max))
	}
}

// InRange checks that an integer is between min and max (inclusive).
func (errs *Errors) InRange(field string, value, min, max int) {
	if value < min || value > max {
		errs.Add(field, fmt.Sprintf("%s must be between %d and %d", field, min, max))
	}
}

// Positive checks that an integer is > 0.
func (errs *Errors) Positive(field string, value int64) {
	if value <= 0 {
		errs.Add(field, fmt.Sprintf("%s must be greater than zero", field))
	}
}

// AllMessages joins all error messages with newlines for display in HTMX responses.
func (errs Errors) AllMessages() string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Message
	}
	return strings.Join(msgs, "\n")
}
