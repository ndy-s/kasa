package customer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndy-s/kasa/backend/internal/platform/auth"
)

type fakeRepository struct {
	byEmail     map[string]*Customer
	hashByEmail map[string]string
	createErr   error
	createCalls int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byEmail:     make(map[string]*Customer),
		hashByEmail: make(map[string]string),
	}
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

func (f *fakeRepository) CreateWithCredential(ctx context.Context, c *Customer, passwordHash string) (*Customer, error) {
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	c.ID = "fake-id"
	f.byEmail[c.Email] = c
	f.hashByEmail[c.Email] = passwordHash
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

func (f *fakeRepository) GetCredentialByEmail(ctx context.Context, email string) (string, string, error) {
	c, ok := f.byEmail[email]
	if !ok {
		return "", "", ErrInvalidCredentials
	}
	return c.ID, f.hashByEmail[email], nil
}

func newTestService(repo Repository) *Service {
	return NewService(repo, auth.NewTokenIssuer("test-secret", time.Hour))
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name            string
		inName          string
		inEmail         string
		inPassword      string
		repo            *fakeRepository
		wantErr         error
		wantCreateCalls int
	}{
		{
			name:            "valid registration succeeds",
			inName:          "Jane Doe",
			inEmail:         "jane@example.com",
			inPassword:      "s3cret-password",
			repo:            newFakeRepository(),
			wantErr:         nil,
			wantCreateCalls: 1,
		},
		{
			name:            "invalid email is rejected before hitting the repository",
			inName:          "Jane Doe",
			inEmail:         "not-an-email",
			inPassword:      "s3cret-password",
			repo:            newFakeRepository(),
			wantErr:         ErrInvalidEmail,
			wantCreateCalls: 0,
		},
		{
			name:            "duplicate email surfaces from the repository",
			inName:          "Jane Doe",
			inEmail:         "jane@example.com",
			inPassword:      "s3cret-password",
			repo:            &fakeRepository{byEmail: map[string]*Customer{}, hashByEmail: map[string]string{}, createErr: ErrEmailTaken},
			wantErr:         ErrEmailTaken,
			wantCreateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.repo)
			got, err := svc.Register(context.Background(), tt.inName, tt.inEmail, tt.inPassword)

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

func TestLogin(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(repo)

	ctx := context.Background()
	if _, err := svc.Register(ctx, "Jane Doe", "jane@example.com", "s3cret-password"); err != nil {
		t.Fatalf("register: %v", err)
	}

	t.Run("correct password issues a token", func(t *testing.T) {
		token, err := svc.Login(ctx, "jane@example.com", "s3cret-password")
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		if token == "" {
			t.Fatal("expected a non-empty token")
		}
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		_, err := svc.Login(ctx, "jane@example.com", "wrong-password")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidCredentials)
		}
	})

	t.Run("unknown email is rejected", func(t *testing.T) {
		_, err := svc.Login(ctx, "nobody@example.com", "whatever")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidCredentials)
		}
	})
}
