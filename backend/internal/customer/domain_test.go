package customer

import (
	"errors"
	"testing"
)

func TestNewCustomer(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantErr   error
		wantEmail string
	}{
		{"valid", "a@b.com", nil, "a@b.com"},
		{"missing @", "abc.com", ErrInvalidEmail, ""},
		{"empty", "", ErrInvalidEmail, ""},
		{"no dot", "a@b", ErrInvalidEmail, ""},
		{"trailing dot with nothing after", "a@b.", ErrInvalidEmail, ""},
		{"contains space", "a b@c.com", ErrInvalidEmail, ""},
		{"mixed case and surrounding whitespace is normalized", "  Bob@Example.com  ", nil, "bob@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCustomer("Alice", tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got.Name != "Alice" {
				t.Errorf("Name = %q, want %q", got.Name, "Alice")
			}
			if got.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", got.Email, tt.wantEmail)
			}
			if got.Status != StatusPending {
				t.Errorf("Status = %q, want %q", got.Status, StatusPending)
			}
		})
	}
}
