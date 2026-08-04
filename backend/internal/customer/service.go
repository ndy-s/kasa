package customer

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterCustomer(ctx context.Context, name, email string) (*Customer, error) {
	c, err := NewCustomer(name, email)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, c)
}
