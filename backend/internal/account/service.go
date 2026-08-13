package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
)

var ErrAccountNotFound = errors.New("account not found")
var ErrBalanceNotZero = errors.New("account balance must be zero to close")

type Service struct {
	pool *pgxpool.Pool
	q    *postgres.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: postgres.New(pool)}
}

// OpenAccount creates a GL sub-account and the customer account referencing it, in one transaction.
func (s *Service) OpenAccount(ctx context.Context, customerID, productCode string) (*Account, error) {
	product, err := s.q.GetProductByCode(ctx, productCode)
	if err != nil {
		return nil, fmt.Errorf("resolve product: %w", err)
	}
	custID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit
	qtx := s.q.WithTx(tx)

	glID, err := qtx.CreateLedgerAccount(ctx, postgres.CreateLedgerAccountParams{
		Code:     "DEP-" + uuid.NewString(),
		Name:     "Customer Deposit sub-account",
		Type:     "liability",
		Currency: product.Currency,
	})
	if err != nil {
		return nil, err
	}

	row, err := qtx.CreateAccount(ctx, postgres.CreateAccountParams{
		CustomerID:      pgtype.UUID{Bytes: custID, Valid: true},
		ProductID:       product.ID,
		Currency:        product.Currency,
		Status:          string(StatusActive),
		LedgerAccountID: glID,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return toDomain(row), nil
}

func (s *Service) Get(ctx context.Context, id string) (*Account, error) {
	aid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	row, err := s.q.GetAccountByID(ctx, pgtype.UUID{Bytes: aid, Valid: true})
	if err != nil {
		return nil, ErrAccountNotFound
	}
	return toDomain(row), nil
}

func (s *Service) Close(ctx context.Context, id string) error {
	acc, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	glID, _ := uuid.Parse(acc.LedgerAccountID)
	balance, err := s.q.LedgerBalance(ctx, pgtype.UUID{Bytes: glID, Valid: true})
	if err != nil {
		return err
	}
	if balance != 0 {
		return ErrBalanceNotZero
	}

	if err := acc.Close(); err != nil {
		return err
	}
	return s.setStatus(ctx, acc)
}

func (s *Service) Freeze(ctx context.Context, id string) error {
	acc, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := acc.Freeze(); err != nil {
		return err
	}
	return s.setStatus(ctx, acc)
}

func (s *Service) Unfreeze(ctx context.Context, id string) error {
	acc, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := acc.Unfreeze(); err != nil {
		return err
	}
	return s.setStatus(ctx, acc)
}

func (s *Service) setStatus(ctx context.Context, acc *Account) error {
	aid, _ := uuid.Parse(acc.ID)
	return s.q.UpdateAccountStatus(ctx, postgres.UpdateAccountStatusParams{
		ID: pgtype.UUID{Bytes: aid, Valid: true}, Status: string(acc.Status),
	})
}

func toDomain(row postgres.Account) *Account {
	return &Account{
		ID:              uuid.UUID(row.ID.Bytes).String(),
		CustomerID:      uuid.UUID(row.CustomerID.Bytes).String(),
		ProductID:       uuid.UUID(row.ProductID.Bytes).String(),
		Currency:        row.Currency,
		Status:          Status(row.Status),
		LedgerAccountID: uuid.UUID(row.LedgerAccountID.Bytes).String(),
	}
}
