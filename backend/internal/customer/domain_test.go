package customer

import (
	"errors"
	"testing"
)

func TestNewCustomer(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"valid", "a@b.com", nil},
		{"missing @", "abc.com", ErrInvalidEmail},
		{"empty", "", ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCustomer("Alice", tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})

	}

}
