package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestLineTotalUsesPerLineHalfUpRounding(t *testing.T) {
	t.Parallel()

	quantity := decimal.RequireFromString("0.125")
	price := decimal.RequireFromString("19.99")
	if got := LineTotal(quantity, price).StringFixed(2); got != "2.50" {
		t.Fatalf("line total = %s, want 2.50", got)
	}

	quantity = decimal.RequireFromString("3")
	price = decimal.RequireFromString("0.10")
	if got := LineTotal(quantity, price).StringFixed(2); got != "0.30" {
		t.Fatalf("line total = %s, want 0.30", got)
	}
}

func TestParseQuantityRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "0", "-1", "abc", "1000000"} {
		if _, err := ParseQuantity(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestMoneyParsingRejectsExcessPrecision(t *testing.T) {
	t.Parallel()

	if _, err := ParseQuantity("1.2345"); err == nil {
		t.Fatal("quantity with four decimals should be rejected")
	}
	if _, err := ParsePrice("1.23456"); err == nil {
		t.Fatal("price with five decimals should be rejected")
	}
}

func TestParsePriceDistinguishesUnknownAndZero(t *testing.T) {
	t.Parallel()

	unknown, err := ParsePrice("")
	if err != nil || unknown != nil {
		t.Fatalf("empty price should be nil, got %v, %v", unknown, err)
	}

	zero, err := ParsePrice("0")
	if err != nil || zero == nil || !zero.IsZero() {
		t.Fatalf("zero price should be known zero, got %v, %v", zero, err)
	}
}
