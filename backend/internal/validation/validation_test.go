package validation

import (
	"testing"
)

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"Short password (<8)", "Abc1!", false},
		{"Empty password", "", false},
		{"Exact 8 chars", "Pass1234", true},
		{"Longer password", "StrongPassword123!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Password("password", tt.password)
			hasErrors := v.Errors().HasErrors()
			if tt.valid && hasErrors {
				t.Errorf("expected password %q to be valid, but got errors: %v", tt.password, v.Errors())
			}
			if !tt.valid && !hasErrors {
				t.Errorf("expected password %q to be rejected, but passed", tt.password)
			}
		})
	}
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"Valid email", "student@evenrtra.edu", true},
		{"Invalid email (no @)", "invalidemail.com", false},
		{"Invalid email (no domain)", "student@", false},
		{"Empty email", "", true}, // Email validator only checks format if non-empty, Required handles empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Email("email", tt.email)
			hasErrors := v.Errors().HasErrors()
			if tt.valid && hasErrors {
				t.Errorf("expected email %q to be valid, but got errors: %v", tt.email, v.Errors())
			}
			if !tt.valid && !hasErrors {
				t.Errorf("expected email %q to be invalid, but passed", tt.email)
			}
		})
	}
}
