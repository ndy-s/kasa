package txn

import (
	"time"

	"github.com/jackc/pgx/v5"
)

// Context carries the ambient values a money-moving use case needs.
type Context struct {
	BusinessDate time.Time // from the Clock (Day 14); time.Now for now
	Actor        string    // the customer or admin ID, for audit
	Tx           pgx.Tx    // the transaction this use case owns
}
