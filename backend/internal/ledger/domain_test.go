package ledger

import (
	"errors"
	"testing"

	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func line(dir Direction, minor int64) JournalLine {
	return JournalLine{AccountID: "acct", Direction: dir, Amount: money.FromMinor(minor, money.USD)}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		lines   []JournalLine
		wantErr error
	}{
		{"balanced", []JournalLine{line(Debit, 100), line(Credit, 100)}, nil},
		{"unbalanced", []JournalLine{line(Debit, 100), line(Credit, 90)}, ErrUnbalanced},
		{"single line", []JournalLine{line(Debit, 100)}, ErrTooFewLines},
		{"mixed currency", []JournalLine{
			line(Debit, 100),
			{AccountID: "x", Direction: Credit, Amount: money.FromMinor(100, money.SGD)},
		}, ErrCurrencyMixed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JournalEntry{Lines: tt.lines}.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
