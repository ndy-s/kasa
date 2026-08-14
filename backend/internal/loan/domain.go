package loan

import "time"

type Status string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

type Loan struct {
	ID               string
	CustomerID       string
	DepositAccountID string
	PrincipalMinor   int64
	TermMonths       int
	InterestRateBps  int32
	Currency         string
	Status           Status
	DisbursedAt      time.Time
}
