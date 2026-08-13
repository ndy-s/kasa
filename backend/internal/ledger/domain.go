package ledger

import (
	"errors"

	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Income    AccountType = "income"
	Expense   AccountType = "expense"
)

type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

type TransactionType string

const (
	Deposit    TransactionType = "deposit"
	Withdrawal TransactionType = "withdrawal"
	Transfer   TransactionType = "transfer"
)

type LedgerAccount struct {
	ID       string
	Code     string
	Name     string
	Type     AccountType
	Currency string
}

type JournalLine struct {
	AccountID string
	Direction Direction
	Amount    money.Money
}

type JournalEntry struct {
	ID          string
	Type        TransactionType
	Description string
	Lines       []JournalLine
}

var (
	ErrTooFewLines   = errors.New("journal entry needs at least two lines")
	ErrCurrencyMixed = errors.New("journal entry mixes currencies")
	ErrUnbalanced    = errors.New("journal entry does not balance")
)

// Validate enforces the double-entry law: >= 2 lines, one currency, sum(debits) == sum(credits).
func (e JournalEntry) Validate() error {
	if len(e.Lines) < 2 {
		return ErrTooFewLines
	}

	currency := e.Lines[0].Amount.Currency()
	var debits, credits int64
	for _, l := range e.Lines {
		if l.Amount.Currency() != currency {
			return ErrCurrencyMixed
		}
		switch l.Direction {
		case Debit:
			debits += l.Amount.Amount()
		case Credit:
			credits += l.Amount.Amount()
		}
	}

	if debits != credits {
		return ErrUnbalanced
	}
	return nil
}
