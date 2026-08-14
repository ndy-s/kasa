package loan

import (
	"errors"
	"math"
	"time"

	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type Installment struct {
	Number    int
	DueDate   time.Time
	Interest  money.Money
	Principal money.Money
	Balance   money.Money // remaining after this installment
	Status    string      // "due" or "paid"; unset when generating a fresh schedule
}

// Schedule builds a fixed-payment (annuity) amortization schedule: the same total
// payment every month, split between interest on the remaining balance and
// principal, with the final installment paying off the exact remaining balance
// so no cent is lost to rounding.
func Schedule(principal money.Money, annualBps int64, months int, disbursedAt time.Time) ([]Installment, error) {
	if months <= 0 {
		return nil, errors.New("months must be positive")
	}
	cur := principal.Currency()

	// derive the fixed monthly payment; float is used only for this single scalar
	r := float64(annualBps) / 10000.0 / 12.0
	p := float64(principal.Amount())
	var paymentMinor int64
	if r == 0 {
		paymentMinor = int64(math.Ceil(p / float64(months)))
	} else {
		paymentMinor = int64(math.Round(p * r / (1 - math.Pow(1+r, -float64(months)))))
	}
	payment := money.FromMinor(paymentMinor, cur)

	schedule := make([]Installment, 0, months)
	balance := principal
	for i := 1; i <= months; i++ {
		interest := balance.InterestForDays(annualBps, 30) // ~one month
		principalPart, err := payment.Sub(interest)
		if err != nil {
			return nil, err
		}
		if i == months {
			principalPart = balance // pay off the exact remaining balance
		}
		balance, err = balance.Sub(principalPart)
		if err != nil {
			return nil, err
		}
		schedule = append(schedule, Installment{
			Number:    i,
			DueDate:   disbursedAt.AddDate(0, i, 0),
			Interest:  interest,
			Principal: principalPart,
			Balance:   balance,
		})
	}
	return schedule, nil
}
