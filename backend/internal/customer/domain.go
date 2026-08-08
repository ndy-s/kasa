package customer

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)

type Customer struct {
	ID     string
	Name   string
	Email  string
	Status Status
}

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Repository interface {
	Create(ctx context.Context, c *Customer) (*Customer, error)
	CreateWithCredential(ctx context.Context, c *Customer, passwordHash string) (*Customer, error)
	GetByID(ctx context.Context, id string) (*Customer, error)
	GetByEmail(ctx context.Context, email string) (*Customer, error)
	GetCredentialByEmail(ctx context.Context, email string) (customerID, passwordHash string, err error)
}

func NewCustomer(name, email string) (*Customer, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if !isValidEmail(normalized) {
		return nil, ErrInvalidEmail
	}

	return &Customer{
		Name:   name,
		Email:  normalized,
		Status: StatusPending,
	}, nil
}

func isValidEmail(email string) bool {
	var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	return emailRe.MatchString(email)
}
