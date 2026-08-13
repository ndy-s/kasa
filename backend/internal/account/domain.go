package account

import "errors"

type Status string

const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
	StatusFrozen  Status = "frozen"
	StatusClosed  Status = "closed"
)

type Account struct {
	ID              string
	CustomerID      string
	ProductID       string
	Currency        string
	Status          Status
	LedgerAccountID string
}

var ErrIllegalTransition = errors.New("illegal account status transition")

func (a *Account) Activate() error {
	if a.Status != StatusPending && a.Status != StatusFrozen {
		return ErrIllegalTransition
	}
	a.Status = StatusActive
	return nil
}

func (a *Account) Freeze() error {
	if a.Status != StatusActive {
		return ErrIllegalTransition
	}
	a.Status = StatusFrozen
	return nil
}

func (a *Account) Unfreeze() error {
	if a.Status != StatusFrozen {
		return ErrIllegalTransition
	}
	a.Status = StatusActive
	return nil
}

func (a *Account) Close() error {
	if a.Status == StatusClosed {
		return ErrIllegalTransition
	}
	a.Status = StatusClosed
	return nil
}
