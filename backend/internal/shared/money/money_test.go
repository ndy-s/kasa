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

func TestAdd(t *testing.T) {
	got, err := FromMinor(300, USD).Add(FromMinor(200, USD))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Amount() != 500 {
		t.Errorf("got %d, want 500", got.Amount())
	}
}

func TestAddCurrencyMismatch(t *testing.T) {
	usd := FromMinor(100, USD)
	sgd := FromMinor(100, SGD)
	if _, err := usd.Add(sgd); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestSub(t *testing.T) {
	got, err := FromMinor(500, USD).Sub(FromMinor(200, USD))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Amount() != 300 {
		t.Errorf("got %d, want 300", got.Amount())
	}

	if _, err := FromMinor(500, USD).Sub(FromMinor(200, SGD)); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestNegate(t *testing.T) {
	if got := FromMinor(500, USD).Negate().Amount(); got != -500 {
		t.Errorf("got %d, want -500", got)
	}
	if got := FromMinor(-500, USD).Negate().Amount(); got != 500 {
		t.Errorf("got %d, want 500", got)
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b int64
		want int
	}{
		{"less than", 100, 200, -1},
		{"greater than", 200, 100, 1},
		{"equal", 100, 100, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromMinor(tt.a, USD).Compare(FromMinor(tt.b, USD))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}

	if _, err := FromMinor(100, USD).Compare(FromMinor(100, SGD)); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestSignPredicates(t *testing.T) {
	tests := []struct {
		amount                          int64
		wantZero, wantNegative, wantPos bool
	}{
		{0, true, false, false},
		{-1, false, true, false},
		{1, false, false, true},
	}
	for _, tt := range tests {
		m := FromMinor(tt.amount, USD)
		if got := m.IsZero(); got != tt.wantZero {
			t.Errorf("IsZero(%d) = %v, want %v", tt.amount, got, tt.wantZero)
		}
		if got := m.IsNegative(); got != tt.wantNegative {
			t.Errorf("IsNegative(%d) = %v, want %v", tt.amount, got, tt.wantNegative)
		}
		if got := m.IsPositive(); got != tt.wantPos {
			t.Errorf("IsPositive(%d) = %v, want %v", tt.amount, got, tt.wantPos)
		}
	}
}

func TestForCode(t *testing.T) {
	tests := []struct {
		code string
		want Currency
	}{
		{"IDR", IDR},
		{"USD", USD},
		{"SGD", SGD},
	}
	for _, tt := range tests {
		got, err := ForCode(tt.code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != tt.want {
			t.Errorf("ForCode(%q) = %+v, want %+v", tt.code, got, tt.want)
		}
	}

	if _, err := ForCode("XYZ"); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("got %v, want ErrInvalidAmount for an unknown currency", err)
	}
}

func TestCurrency(t *testing.T) {
	if got := FromMinor(100, SGD).Currency(); got != SGD {
		t.Errorf("got %+v, want %+v", got, SGD)
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

func TestInterestForDays(t *testing.T) {
	tests := []struct {
		name      string
		amount    int64
		annualBps int64
		days      int
		want      int64
	}{
		{"1000.00 at 1.50% for 1 day", 100000, 150, 1, 4},
		{"1000.00 at 1.50% for a full year equals exactly the annual rate", 100000, 150, 365, 1500},
		{"exact half, odd quotient rounds up (banker's rounding)", 5475000, 1, 1, 2},
		{"exact half, even quotient stays put (banker's rounding)", 9125000, 1, 1, 2},
		{"negative balance", -100000, 150, 1, -4},
		{"zero amount", 0, 150, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromMinor(tt.amount, USD).InterestForDays(tt.annualBps, tt.days)
			if got.Amount() != tt.want {
				t.Errorf("got %d, want %d", got.Amount(), tt.want)
			}
		})
	}
}
