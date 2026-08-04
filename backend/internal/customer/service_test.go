package customer

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	byEmail     map[string]*Customer
	createErr   error
	createCalls int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byEmail: make(map[string]*Customer)}
}

func (f *fakeRepository) Create(ctx context.Context, c *Customer) (*Customer, error) {
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	c.ID = "fake-id"
	f.byEmail[c.Email] = c
	return c, nil
}

func (f *fakeRepository) GetByID(ctx context.Context, id string) (*Customer, error) {
	for _, c := range f.byEmail {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, ErrCustomerNotFound
}

func (f *fakeRepository) GetByEmail(ctx context.Context, email string) (*Customer, error) {
	c, ok := f.byEmail[email]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	return c, nil
}

func TestRegisterCustomer(t *testing.T) {
	tests := []struct {
		name            string
		inName          string
		inEmail         string
		repo            *fakeRepository
		wantErr         error
		wantCreateCalls int
	}{
		{
			name:            "valid registration succeeds",
			inName:          "Jane Doe",
			inEmail:         "jane@example.com",
			repo:            newFakeRepository(),
			wantErr:         nil,
			wantCreateCalls: 1,
		},
		{
			name:            "invalid email is rejected before hitting the repository",
			inName:          "Jane Doe",
			inEmail:         "not-an-email",
			repo:            newFakeRepository(),
			wantErr:         ErrInvalidEmail,
			wantCreateCalls: 0,
		},
		{
			name:            "duplicate email surfaces from the repository",
			inName:          "Jane Doe",
			inEmail:         "jane@example.com",
			repo:            &fakeRepository{byEmail: map[string]*Customer{}, createErr: ErrEmailTaken},
			wantErr:         ErrEmailTaken,
			wantCreateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			got, err := svc.RegisterCustomer(context.Background(), tt.inName, tt.inEmail)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && got == nil {
				t.Fatal("expected a customer, got nil")
			}

			if tt.repo.createCalls != tt.wantCreateCalls {
				t.Fatalf("createCalls = %d, want %d", tt.repo.createCalls, tt.wantCreateCalls)
			}
		})
	}
}
