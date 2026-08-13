package account

import (
	"errors"
	"testing"
)

func TestTransitions(t *testing.T) {
	tests := []struct {
		name    string
		start   Status
		action  func(*Account) error
		wantErr error
		want    Status
	}{
		{"activate pending", StatusPending, (*Account).Activate, nil, StatusActive},
		{"activate frozen", StatusFrozen, (*Account).Activate, nil, StatusActive},
		{"freeze active", StatusActive, (*Account).Freeze, nil, StatusFrozen},
		{"unfreeze frozen", StatusFrozen, (*Account).Unfreeze, nil, StatusActive},
		{"freeze pending is illegal", StatusPending, (*Account).Freeze, ErrIllegalTransition, StatusPending},
		{"unfreeze active is illegal", StatusActive, (*Account).Unfreeze, ErrIllegalTransition, StatusActive},
		{"close pending", StatusPending, (*Account).Close, nil, StatusClosed},
		{"close closed is illegal", StatusClosed, (*Account).Close, ErrIllegalTransition, StatusClosed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Account{Status: tt.start}
			err := tt.action(a)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if a.Status != tt.want {
				t.Errorf("status = %v, want %v", a.Status, tt.want)
			}
		})
	}
}
