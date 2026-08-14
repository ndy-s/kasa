package ledger

import (
	"errors"
	"testing"

	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestLinesFor(t *testing.T) {
	amt := func(minor int64) money.Money { return money.FromMinor(minor, money.IDR) }

	tests := []struct {
		name string
		typ  TransactionType
		p    PostingParams
		want []JournalLine
	}{
		{
			name: "deposit debits cash, credits the customer account",
			typ:  Deposit,
			p:    PostingParams{Amount: amt(1000), CashAccountID: "cash", ToAccountID: "customer"},
			want: []JournalLine{
				{AccountID: "cash", Direction: Debit, Amount: amt(1000)},
				{AccountID: "customer", Direction: Credit, Amount: amt(1000)},
			},
		},
		{
			name: "withdrawal debits the customer account, credits cash",
			typ:  Withdrawal,
			p:    PostingParams{Amount: amt(1000), CashAccountID: "cash", FromAccountID: "customer"},
			want: []JournalLine{
				{AccountID: "customer", Direction: Debit, Amount: amt(1000)},
				{AccountID: "cash", Direction: Credit, Amount: amt(1000)},
			},
		},
		{
			name: "transfer debits the source, credits the destination",
			typ:  Transfer,
			p:    PostingParams{Amount: amt(1000), FromAccountID: "from", ToAccountID: "to"},
			want: []JournalLine{
				{AccountID: "from", Direction: Debit, Amount: amt(1000)},
				{AccountID: "to", Direction: Credit, Amount: amt(1000)},
			},
		},
		{
			name: "interest credit debits the expense account, credits the deposit",
			typ:  InterestCredit,
			p:    PostingParams{Amount: amt(50), FromAccountID: "expense", ToAccountID: "deposit"},
			want: []JournalLine{
				{AccountID: "expense", Direction: Debit, Amount: amt(50)},
				{AccountID: "deposit", Direction: Credit, Amount: amt(50)},
			},
		},
		{
			name: "disbursement debits loans receivable, credits the deposit",
			typ:  Disbursement,
			p:    PostingParams{Amount: amt(500000), CashAccountID: "receivable", ToAccountID: "deposit"},
			want: []JournalLine{
				{AccountID: "receivable", Direction: Debit, Amount: amt(500000)},
				{AccountID: "deposit", Direction: Credit, Amount: amt(500000)},
			},
		},
		{
			name: "repayment splits principal and interest across three lines",
			typ:  Repayment,
			p: PostingParams{
				Amount: amt(900), FromAccountID: "deposit", ToAccountID: "receivable",
				SecondAmount: amt(100), SecondAccountID: "income",
			},
			want: []JournalLine{
				{AccountID: "deposit", Direction: Debit, Amount: amt(1000)},
				{AccountID: "receivable", Direction: Credit, Amount: amt(900)},
				{AccountID: "income", Direction: Credit, Amount: amt(100)},
			},
		},
		{
			name: "repayment with zero interest still posts a zero-amount income line",
			typ:  Repayment,
			p: PostingParams{
				Amount: amt(1000), FromAccountID: "deposit", ToAccountID: "receivable",
				SecondAmount: amt(0), SecondAccountID: "income",
			},
			want: []JournalLine{
				{AccountID: "deposit", Direction: Debit, Amount: amt(1000)},
				{AccountID: "receivable", Direction: Credit, Amount: amt(1000)},
				{AccountID: "income", Direction: Credit, Amount: amt(0)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LinesFor(tt.typ, tt.p)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
			// every rule must itself produce a balanced entry
			if err := (JournalEntry{Lines: got}).Validate(); err != nil {
				t.Errorf("unbalanced entry: %v", err)
			}
		})
	}
}

func TestLinesForUnknownType(t *testing.T) {
	if _, err := LinesFor("bogus", PostingParams{}); !errors.Is(err, ErrUnknownTransactionType) {
		t.Fatalf("got %v, want ErrUnknownTransactionType", err)
	}
}

func TestRepaymentRuleCurrencyMismatch(t *testing.T) {
	_, err := LinesFor(Repayment, PostingParams{
		Amount:          money.FromMinor(1000, money.IDR),
		FromAccountID:   "deposit",
		ToAccountID:     "receivable",
		SecondAmount:    money.FromMinor(100, money.USD), // mismatched currency
		SecondAccountID: "income",
	})
	if !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}
