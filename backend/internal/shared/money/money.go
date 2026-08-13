package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Currency struct {
	Code     string
	Exponent int // number of minor-unit digits: 2 means cents
}

var (
	USD = Currency{Code: "USD", Exponent: 2}
	SGD = Currency{Code: "SGD", Exponent: 2}
)

type Money struct {
	amount   int64 // minor units, e.g. 12345 == 123.45
	currency Currency
}

var (
	ErrCurrencyMismatch = errors.New("currency mismatch")
	ErrInvalidAmount    = errors.New("invalid amount")
)

// FromMinor builds Money from an exact minor-unit count.
func FromMinor(amount int64, c Currency) Money {
	return Money{amount: amount, currency: c}
}

// Parse reads a decimal string such as "123.45" into Money, using the currency's exponent.
func Parse(s string, c Currency) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Money{}, ErrInvalidAmount
	}

	neg := false
	switch s[0] {
	case '-':
		neg, s = true, s[1:]
	case '+':
		s = s[1:]
	}

	intPart, fracPart := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if len(fracPart) > c.Exponent {
		return Money{}, fmt.Errorf("%w: too many decimal places for %s", ErrInvalidAmount, c.Code)
	}
	for len(fracPart) < c.Exponent {
		fracPart += "0"
	}

	digits := intPart + fracPart
	if digits == "" {
		return Money{}, ErrInvalidAmount
	}
	amount, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}
	if neg {
		amount = -amount
	}
	return Money{amount: amount, currency: c}, nil
}

func (m Money) Amount() int64      { return m.amount }
func (m Money) Currency() Currency { return m.currency }
func (m Money) IsZero() bool       { return m.amount == 0 }
func (m Money) IsNegative() bool   { return m.amount < 0 }

func (m Money) Add(o Money) (Money, error) {
	if m.currency != o.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount + o.amount, currency: m.currency}, nil
}

func (m Money) Sub(o Money) (Money, error) {
	if m.currency != o.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount - o.amount, currency: m.currency}, nil
}

func (m Money) Negate() Money {
	return Money{amount: -m.amount, currency: m.currency}
}

func (m Money) Compare(o Money) (int, error) {
	if m.currency != o.currency {
		return 0, ErrCurrencyMismatch
	}
	switch {
	case m.amount < o.amount:
		return -1, nil
	case m.amount > o.amount:
		return 1, nil
	default:
		return 0, nil
	}
}

func (m Money) String() string {
	if m.currency.Exponent == 0 {
		return fmt.Sprintf("%d %s", m.amount, m.currency.Code)
	}
	neg, a := "", m.amount
	if a < 0 {
		neg, a = "-", -a
	}
	factor := int64(1)
	for i := 0; i < m.currency.Exponent; i++ {
		factor *= 10
	}
	return fmt.Sprintf("%s%d.%0*d %s", neg, a/factor, m.currency.Exponent, a%factor, m.currency.Code)
}

// Allocate splits the amount into n parts that sum exactly to the original.
func (m Money) Allocate(n int) ([]Money, error) {
	if n <= 0 {
		return nil, ErrInvalidAmount
	}
	base := m.amount / int64(n)
	rem := m.amount % int64(n)
	parts := make([]Money, n)
	for i := range parts {
		amt := base
		switch {
		case rem > 0 && int64(i) < rem:
			amt++
		case rem < 0 && int64(i) < -rem:
			amt--
		}
		parts[i] = Money{amount: amt, currency: m.currency}
	}
	return parts, nil
}

// ApplyRate multiplies by a rate in basis points (150 == 1.50%), rounding half to even.
func (m Money) ApplyRate(bps int64) Money {
	num := m.amount * bps
	q := num / 10000
	r := num % 10000
	if r < 0 {
		r = -r
	}
	const half = 5000
	if r > half || (r == half && q%2 != 0) {
		if num < 0 {
			q--
		} else {
			q++
		}
	}
	return Money{amount: q, currency: m.currency}
}
