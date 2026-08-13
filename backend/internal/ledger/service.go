package ledger

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
)

type PostingRequest struct {
	Type        TransactionType
	Description string
	BookingDate time.Time
	Lines       []JournalLine
}

type Service struct{}

func NewService() *Service { return &Service{} }

// Post validates the entry and writes it plus its lines atomically inside tx.
func (s *Service) Post(ctx context.Context, tx pgx.Tx, req PostingRequest) (string, error) {
	entry := JournalEntry{Type: req.Type, Description: req.Description, Lines: req.Lines}
	if err := entry.Validate(); err != nil {
		return "", err
	}

	q := postgres.New(tx)

	date := pgtype.Date{Time: req.BookingDate, Valid: true}
	entryID, err := q.CreateJournalEntry(ctx, postgres.CreateJournalEntryParams{
		TransactionType: string(req.Type),
		Description:     req.Description,
		BookingDate:     date,
		ValueDate:       date,
	})
	if err != nil {
		return "", err
	}

	for _, l := range req.Lines {
		accountID, err := uuid.Parse(l.AccountID)
		if err != nil {
			return "", err
		}
		err = q.CreateJournalLine(ctx, postgres.CreateJournalLineParams{
			JournalEntryID:  entryID,
			LedgerAccountID: pgtype.UUID{Bytes: accountID, Valid: true},
			Direction:       string(l.Direction),
			AmountMinor:     l.Amount.Amount(),
			Currency:        l.Amount.Currency().Code,
		})
		if err != nil {
			return "", err
		}
	}

	return uuid.UUID(entryID.Bytes).String(), nil
}
