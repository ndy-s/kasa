package loan

import (
	"testing"
	"time"

	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestSchedule(t *testing.T) {
	principal := money.FromMinor(12_000_000_00, money.IDR) // Rp 12,000,000.00
	disbursed := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("builds one installment per month and zeroes out", func(t *testing.T) {
		schedule, err := Schedule(principal, 1800, 12, disbursed) // 18% annual, 12 months
		if err != nil {
			t.Fatal(err)
		}
		if len(schedule) != 12 {
			t.Fatalf("got %d installments, want 12", len(schedule))
		}
		last := schedule[len(schedule)-1]
		if !last.Balance.IsZero() {
			t.Errorf("final balance = %s, want 0", last.Balance)
		}
	})

	t.Run("principal sums exactly to the loan amount", func(t *testing.T) {
		schedule, _ := Schedule(principal, 1800, 12, disbursed)
		var total int64
		for _, inst := range schedule {
			total += inst.Principal.Amount()
		}
		if total != principal.Amount() {
			t.Errorf("principal sum = %d, want %d", total, principal.Amount())
		}
	})

	t.Run("interest declines as the balance is paid down", func(t *testing.T) {
		schedule, _ := Schedule(principal, 1800, 12, disbursed)
		if schedule[len(schedule)-1].Interest.Amount() >= schedule[0].Interest.Amount() {
			t.Errorf("interest did not decline: first = %s, last = %s",
				schedule[0].Interest, schedule[len(schedule)-1].Interest)
		}
	})

	t.Run("due dates step one month at a time from disbursement", func(t *testing.T) {
		schedule, _ := Schedule(principal, 1800, 12, disbursed)
		for i, inst := range schedule {
			want := disbursed.AddDate(0, i+1, 0)
			if !inst.DueDate.Equal(want) {
				t.Errorf("installment %d due date = %s, want %s", inst.Number, inst.DueDate, want)
			}
		}
	})

	t.Run("zero rate splits principal evenly with no interest", func(t *testing.T) {
		schedule, err := Schedule(principal, 0, 12, disbursed)
		if err != nil {
			t.Fatal(err)
		}
		for _, inst := range schedule {
			if !inst.Interest.IsZero() {
				t.Errorf("installment %d interest = %s, want 0", inst.Number, inst.Interest)
			}
		}
		if !schedule[len(schedule)-1].Balance.IsZero() {
			t.Errorf("final balance = %s, want 0", schedule[len(schedule)-1].Balance)
		}
	})

	t.Run("rejects a non-positive term", func(t *testing.T) {
		if _, err := Schedule(principal, 1800, 0, disbursed); err == nil {
			t.Error("want an error for a zero-month term")
		}
	})
}
