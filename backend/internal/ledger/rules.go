package ledger

import (
	"errors"

	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

// PostingParams carries the resolved GL account IDs and the amount for a transaction.
type PostingParams struct {
	Amount        money.Money
	CashAccountID string // the bank's cash GL account
	FromAccountID string // the debited customer account (withdraw, transfer source)
	ToAccountID   string // the credited customer account (deposit, transfer destination)
}

type PostingRule func(p PostingParams) ([]JournalLine, error)

var ErrUnknownTransactionType = errors.New("unknown transaction type")

var rules = map[TransactionType]PostingRule{
	Deposit:        depositRule,
	Withdrawal:     withdrawalRule,
	Transfer:       transferRule,
	InterestCredit: interestCreditRule,
}

// LinesFor looks up the rule for a transaction type and produces its balanced lines.
func LinesFor(t TransactionType, p PostingParams) ([]JournalLine, error) {
	rule, ok := rules[t]
	if !ok {
		return nil, ErrUnknownTransactionType
	}
	return rule(p)
}

func depositRule(p PostingParams) ([]JournalLine, error) {
	return []JournalLine{
		{AccountID: p.CashAccountID, Direction: Debit, Amount: p.Amount},
		{AccountID: p.ToAccountID, Direction: Credit, Amount: p.Amount},
	}, nil
}

func withdrawalRule(p PostingParams) ([]JournalLine, error) {
	return []JournalLine{
		{AccountID: p.FromAccountID, Direction: Debit, Amount: p.Amount},
		{AccountID: p.CashAccountID, Direction: Credit, Amount: p.Amount},
	}, nil
}

func transferRule(p PostingParams) ([]JournalLine, error) {
	return []JournalLine{
		{AccountID: p.FromAccountID, Direction: Debit, Amount: p.Amount},
		{AccountID: p.ToAccountID, Direction: Credit, Amount: p.Amount},
	}, nil
}

func interestCreditRule(p PostingParams) ([]JournalLine, error) {
	return []JournalLine{
		{AccountID: p.FromAccountID, Direction: Debit, Amount: p.Amount},
		{AccountID: p.ToAccountID, Direction: Credit, Amount: p.Amount},
	}, nil
}
