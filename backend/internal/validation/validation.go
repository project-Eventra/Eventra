package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ValidationError holds field-level validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Validator provides chainable validation
type Validator struct {
	errors ValidationErrors
}

// New creates a new validator
func New() *Validator {
	return &Validator{}
}

// Required checks that a string is not empty
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "is required"})
	}
	return v
}

// MinLength checks minimum string length
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if utf8.RuneCountInString(strings.TrimSpace(value)) < min {
		v.errors = append(v.errors, ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d characters", min)})
	}
	return v
}

// MaxLength checks maximum string length
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.errors = append(v.errors, ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d characters", max)})
	}
	return v
}

// Email validates email format
func (v *Validator) Email(field, value string) *Validator {
	if value == "" {
		return v
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "must be a valid email address"})
	}
	return v
}

var phoneRegex = regexp.MustCompile(`^\+?[0-9\s\-()]{7,20}$`)

// Phone validates phone number format
func (v *Validator) Phone(field, value string) *Validator {
	if value == "" {
		return v
	}
	if !phoneRegex.MatchString(value) {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "must be a valid phone number"})
	}
	return v
}

// OneOf checks that value is one of the allowed values
func (v *Validator) OneOf(field, value string, allowed []string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errors = append(v.errors, ValidationError{Field: field, Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", "))})
	return v
}

// DateAfterNow checks that a date string is in the future
func (v *Validator) DateAfterNow(field, value string) *Validator {
	if value == "" {
		return v
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "must be a valid date-time (RFC3339 format)"})
		return v
	}
	if t.Before(time.Now()) {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "must be in the future"})
	}
	return v
}

// DateBefore checks that dateA is before dateB
func (v *Validator) DateBefore(fieldA, valueA, fieldB, valueB string) *Validator {
	if valueA == "" || valueB == "" {
		return v
	}
	tA, errA := time.Parse(time.RFC3339, valueA)
	tB, errB := time.Parse(time.RFC3339, valueB)
	if errA != nil || errB != nil {
		return v
	}
	if !tA.Before(tB) {
		v.errors = append(v.errors, ValidationError{Field: fieldA, Message: fmt.Sprintf("must be before %s", fieldB)})
	}
	return v
}

// Min checks that an integer is at least the minimum
func (v *Validator) Min(field string, value, min int) *Validator {
	if value < min {
		v.errors = append(v.errors, ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d", min)})
	}
	return v
}

// Max checks that an integer is at most the maximum
func (v *Validator) Max(field string, value, max int) *Validator {
	if value > max {
		v.errors = append(v.errors, ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d", max)})
	}
	return v
}

// MinFloat checks that a float is at least the minimum
func (v *Validator) MinFloat(field string, value, min float64) *Validator {
	if value < min {
		v.errors = append(v.errors, ValidationError{Field: field, Message: fmt.Sprintf("must be at least %.2f", min)})
	}
	return v
}

// Custom adds a custom validation rule
func (v *Validator) Custom(field, message string, isValid bool) *Validator {
	if !isValid {
		v.errors = append(v.errors, ValidationError{Field: field, Message: message})
	}
	return v
}

// Errors returns the collected validation errors
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// Password validates password strength (min 8 characters)
func (v *Validator) Password(field, value string) *Validator {
	if utf8.RuneCountInString(value) < 8 {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "must be at least 8 characters"})
	}
	return v
}
