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

	// SecondAmount and SecondAccountID describe a rule's third line, when it has one (loan repayment).
	SecondAmount    money.Money
	SecondAccountID string
}

type PostingRule func(p PostingParams) ([]JournalLine, error)

var ErrUnknownTransactionType = errors.New("unknown transaction type")

var rules = map[TransactionType]PostingRule{
	Deposit:        depositRule,
	Withdrawal:     withdrawalRule,
	Transfer:       transferRule,
	InterestCredit: interestCreditRule,
	Disbursement:   disbursementRule,
	Repayment:      repaymentRule,
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

// disbursementRule: Dr Loans Receivable (passed as CashAccountID, the other bank-owned GL account) / Cr
// the customer's deposit.
func disbursementRule(p PostingParams) ([]JournalLine, error) {
	return []JournalLine{
		{AccountID: p.CashAccountID, Direction: Debit, Amount: p.Amount},
		{AccountID: p.ToAccountID, Direction: Credit, Amount: p.Amount},
	}, nil
}

// repaymentRule: Dr the customer's deposit for principal+interest / Cr Loans Receivable (principal) / Cr
// Interest Income (interest). Amount and ToAccountID carry the principal leg; SecondAmount and
// SecondAccountID carry the interest leg.
func repaymentRule(p PostingParams) ([]JournalLine, error) {
	total, err := p.Amount.Add(p.SecondAmount)
	if err != nil {
		return nil, err
	}
	return []JournalLine{
		{AccountID: p.FromAccountID, Direction: Debit, Amount: total},
		{AccountID: p.ToAccountID, Direction: Credit, Amount: p.Amount},
		{AccountID: p.SecondAccountID, Direction: Credit, Amount: p.SecondAmount},
	}, nil
}
