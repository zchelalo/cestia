package domain

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
	ErrInvalidPrice    = errors.New("price is outside the supported range")
)

var (
	maxQuantity = decimal.RequireFromString("999999.999")
	maxPrice    = decimal.RequireFromString("99999999.9999")
)

// ParseQuantity validates a user-supplied quantity and normalizes it to three
// decimal places. The returned value is exact and safe for persistence.
func ParseQuantity(raw string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || value.LessThanOrEqual(decimal.Zero) || value.GreaterThan(maxQuantity) {
		return decimal.Zero, ErrInvalidQuantity
	}
	if value.Exponent() < -3 {
		return decimal.Zero, ErrInvalidQuantity
	}
	return value, nil
}

// ParsePrice validates a nullable monetary unit price. An empty string means
// that the price is unknown and must remain null in persistence.
func ParsePrice(raw string) (*decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.IsNegative() || value.GreaterThan(maxPrice) {
		return nil, ErrInvalidPrice
	}
	if value.Exponent() < -4 {
		return nil, ErrInvalidPrice
	}
	return &value, nil
}

// LineTotal applies the single rounding rule used by both the application and
// the stored generated column in PostgreSQL.
func LineTotal(quantity, unitPrice decimal.Decimal) decimal.Decimal {
	return quantity.Mul(unitPrice).Round(2)
}

func MoneyString(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	formatted := value.StringFixed(2)
	return &formatted
}

func FormatMoney(value decimal.Decimal) string {
	return value.StringFixed(2)
}
