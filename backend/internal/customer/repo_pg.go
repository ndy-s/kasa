package customer

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
)

var _ Repository = (*PgRepository)(nil)

type PgRepository struct {
	pool *pgxpool.Pool
	q    *postgres.Queries
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool, q: postgres.New(pool)}
}

func (r *PgRepository) Create(ctx context.Context, c *Customer) (*Customer, error) {
	row, err := r.q.CreateCustomer(ctx, postgres.CreateCustomerParams{
		Name:  c.Name,
		Email: c.Email,
	})
	if err != nil {
		return nil, mapCreateError(err)
	}
	return toDomain(row), nil
}

func (r *PgRepository) CreateWithCredential(ctx context.Context, c *Customer, passwordHash string) (*Customer, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	qtx := r.q.WithTx(tx)

	row, err := qtx.CreateCustomer(ctx, postgres.CreateCustomerParams{Name: c.Name, Email: c.Email})
	if err != nil {
		return nil, mapCreateError(err)
	}

	if err := qtx.CreateCredential(ctx, postgres.CreateCredentialParams{
		CustomerID:   row.ID,
		PasswordHash: passwordHash,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
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

func (r *PgRepository) GetCredentialByEmail(ctx context.Context, email string) (string, string, error) {
	row, err := r.q.GetCredentialByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", err
	}
	return uuid.UUID(row.ID.Bytes).String(), row.PasswordHash, nil
}

func toDomain(row postgres.Customer) *Customer {
	return &Customer{
		ID:     uuid.UUID(row.ID.Bytes).String(),
		Name:   row.Name,
		Email:  row.Email,
		Status: Status(row.Status),
	}
}

func mapCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrEmailTaken
	}
	return err
}
