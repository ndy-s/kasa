package customer

import (
	"context"

	"github.com/ndy-s/kasa/backend/internal/platform/auth"
)

type Service struct {
	repo   Repository
	issuer *auth.TokenIssuer
}

func NewService(repo Repository, issuer *auth.TokenIssuer) *Service {
	return &Service{repo: repo, issuer: issuer}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (*Customer, error) {
	c, err := NewCustomer(name, email)
	if err != nil {
		return nil, err
	}

	hash, err := auth.Hash(password)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateWithCredential(ctx, c, hash)
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	customerID, hash, err := s.repo.GetCredentialByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	ok, err := auth.Verify(password, hash)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidCredentials
	}

	return s.issuer.Issue(customerID)
}

func (s *Service) Get(ctx context.Context, id string) (*Customer, error) {
	return s.repo.GetByID(ctx, id)
}
