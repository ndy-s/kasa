package statement

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type Line struct {
	BookingDate time.Time
	Type        string
	Direction   string
	Amount      money.Money
}

type Statement struct {
	Opening money.Money
	Lines   []Line
	Closing money.Money
}

type Service struct {
	q *postgres.Queries
}

func NewService(q *postgres.Queries) *Service { return &Service{q: q} }

func (s *Service) Generate(ctx context.Context, accountID string, from, to time.Time) (*Statement, error) {
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return nil, err
	}
	acc, err := s.q.GetAccountByID(ctx, pgtype.UUID{Bytes: aid, Valid: true})
	if err != nil {
		return nil, err
	}
	cur, err := money.ForCode(acc.Currency)
	if err != nil {
		return nil, err
	}

	rawOpening, err := s.q.BalanceAsOf(ctx, postgres.BalanceAsOfParams{
		LedgerAccountID: acc.LedgerAccountID,
		Before:          pgtype.Date{Time: from, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	rows, err := s.q.LinesInPeriod(ctx, postgres.LinesInPeriodParams{
		LedgerAccountID: acc.LedgerAccountID,
		FromDate:        pgtype.Date{Time: from, Valid: true},
		ToDate:          pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	closingRaw := rawOpening
	lines := make([]Line, 0, len(rows))
	for _, r := range rows {
		if r.Direction == "credit" {
			closingRaw += r.AmountMinor
		} else {
			closingRaw -= r.AmountMinor
		}
		lines = append(lines, Line{
			BookingDate: r.BookingDate.Time,
			Type:        r.TransactionType,
			Direction:   r.Direction,
			Amount:      money.FromMinor(r.AmountMinor, cur),
		})
	}

	return &Statement{
		Opening: money.FromMinor(rawOpening, cur),
		Lines:   lines,
		Closing: money.FromMinor(closingRaw, cur),
	}, nil
}
