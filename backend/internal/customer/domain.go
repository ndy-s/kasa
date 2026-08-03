package customer

import (
	"context"
	"errors"
	"regexp"
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
	ErrInvalidEmail     = errors.New("invalid email")
	ErrCustomerNotFound = errors.New("customer not found")
	ErrEmailTaken       = errors.New("email already taken")
)

type Repository interface {
	Create(ctx context.Context, c *Customer) (*Customer, error)
	GetByID(ctx context.Context, id string) (*Customer, error)
	GetByEmail(ctx context.Context, email string) (*Customer, error)
}

func NewCustomer(name, email string) (*Customer, error) {
	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}

	return &Customer{
		Name:   name,
		Email:  email,
		Status: StatusPending,
	}, nil
}

func isValidEmail(email string) bool {
	var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	return emailRe.MatchString(email)
}
