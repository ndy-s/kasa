package customer

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
)

var _ Repository = (*PgRepository)(nil)

type PgRepository struct {
	q *postgres.Queries
}

func NewPgRepository(q *postgres.Queries) *PgRepository {
	return &PgRepository{q: q}
}

func (r *PgRepository) Create(ctx context.Context, c *Customer) (*Customer, error) {
	row, err := r.q.CreateCustomer(ctx, postgres.CreateCustomerParams{
		Name:  c.Name,
		Email: c.Email,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return toDomain(row), nil
}

func (r *PgRepository) GetByID(ctx context.Context, id string) (*Customer, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrCustomerNotFound
	}
	row, err := r.q.GetCustomerByID(ctx, pgtype.UUID{Bytes: parsed, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return toDomain(row), nil
}

func (r *PgRepository) GetByEmail(ctx context.Context, email string) (*Customer, error) {
	row, err := r.q.GetCustomerByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return toDomain(row), nil

}

func toDomain(row postgres.Customer) *Customer {
	return &Customer{
		ID:     uuid.UUID(row.ID.Bytes).String(),
		Name:   row.Name,
		Email:  row.Email,
		Status: Status(row.Status),
	}
}
