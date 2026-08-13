package money

import (
	"errors"
	"strings"
	"testing"
	"testing/quick"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"123.45", 12345, false},
		{"0.05", 5, false},
		{"-1.00", -100, false},
		{"10", 1000, false},
		{"1.234", 0, true}, // too many decimals for exponent 2
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			m, err := Parse(tt.in, USD)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.Amount() != tt.want {
				t.Errorf("got %d, want %d", m.Amount(), tt.want)
			}
		})
	}
}

func TestAddCurrencyMismatch(t *testing.T) {
	usd := FromMinor(100, USD)
	sgd := FromMinor(100, SGD)
	if _, err := usd.Add(sgd); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestString(t *testing.T) {
	if got := FromMinor(12345, USD).String(); got != "123.45 USD" {
		t.Errorf("got %q", got)
	}
	if got := FromMinor(-5, USD).String(); got != "-0.05 USD" {
		t.Errorf("got %q", got)
	}
}

func TestAllocateSumsBack(t *testing.T) {
	f := func(amount int64, n uint8) bool {
		amount %= 1_000_000_000
		parts := n%10 + 1 // 1..10 parts
		got, err := FromMinor(amount, USD).Allocate(int(parts))
		if err != nil {
			return false
		}
		var sum int64
		for _, p := range got {
			sum += p.Amount()
		}
		return sum == amount
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestParseStringRoundTrip(t *testing.T) {
	f := func(amount int64) bool {
		amount %= 1_000_000_000_000_000 // keep within a safe range
		m := FromMinor(amount, USD)
		parsed, err := Parse(strings.Fields(m.String())[0], USD)
		return err == nil && parsed.Amount() == amount
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestApplyRate(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		bps    int64
		want   int64
	}{
		{"1,000,000 minor units at 1.25% divides exactly", 1_000_000, 125, 12500},
		{"exact half, odd quotient rounds up (banker's rounding)", 1, 15000, 2},
		{"exact half, even quotient stays put (banker's rounding)", 1, 25000, 2},
		{"negative amount", -1_000_000, 125, -12500},
		{"zero rate", 100000, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromMinor(tt.amount, USD).ApplyRate(tt.bps).Amount()
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
