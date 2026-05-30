package money

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cespedes/accounting"
)

// Money represents a monetary amount with precise decimal arithmetic
// Uses the accounting package to avoid floating point precision issues
type Money struct {
	value    accounting.Value
	currency string
}

// NewMoney creates a new Money instance from a float64 amount
func NewMoney(amount float64) *Money {
	return &Money{
		value:    accounting.Value{Amount: int64(amount * 100)}, // Convert to cents
		currency: "₫",                                           // Default currency
	}
}

// NewMoneyFromInt creates a new Money instance from cents (int64)
func NewMoneyFromInt(cents int64) *Money {
	return &Money{
		value:    accounting.Value{Amount: cents},
		currency: "₫",
	}
}

// NewMoneyFromString creates a new Money instance from a string representation
func NewMoneyFromString(amount string) (*Money, error) {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount format: %w", err)
	}
	return NewMoney(f), nil
}

// Float returns the monetary amount as a float64
func (m *Money) Float() float64 {
	return float64(m.value.Amount) / 100.0
}

// Int64 returns the monetary amount in cents
func (m *Money) Int64() int64 {
	return m.value.Amount
}

// String returns a formatted string representation of the money
func (m *Money) String() string {
	return fmt.Sprintf("%.2f %s", m.Float(), m.currency)
}

// Add adds another Money amount to this one
func (m *Money) Add(other *Money) *Money {
	if other == nil {
		return m
	}
	return &Money{
		value:    accounting.Value{Amount: m.value.Amount + other.value.Amount},
		currency: m.currency,
	}
}

// Subtract subtracts another Money amount from this one
func (m *Money) Subtract(other *Money) *Money {
	if other == nil {
		return m
	}
	return &Money{
		value:    accounting.Value{Amount: m.value.Amount - other.value.Amount},
		currency: m.currency,
	}
}

// Multiply multiplies the money amount by a factor
func (m *Money) Multiply(factor float64) *Money {
	cents := int64(float64(m.value.Amount) * factor)
	return &Money{
		value:    accounting.Value{Amount: cents},
		currency: m.currency,
	}
}

// IsZero returns true if the amount is zero
func (m *Money) IsZero() bool {
	return m.value.Amount == 0
}

// IsPositive returns true if the amount is positive
func (m *Money) IsPositive() bool {
	return m.value.Amount > 0
}

// IsNegative returns true if the amount is negative
func (m *Money) IsNegative() bool {
	return m.value.Amount < 0
}

// Equal checks if two Money amounts are equal
func (m *Money) Equal(other *Money) bool {
	if other == nil {
		return m.IsZero()
	}
	return m.value.Amount == other.value.Amount && m.currency == other.currency
}

// GreaterThan checks if this amount is greater than another
func (m *Money) GreaterThan(other *Money) bool {
	if other == nil {
		return m.IsPositive()
	}
	return m.value.Amount > other.value.Amount
}

// LessThan checks if this amount is less than another
func (m *Money) LessThan(other *Money) bool {
	if other == nil {
		return m.IsNegative()
	}
	return m.value.Amount < other.value.Amount
}

// Abs returns the absolute value of the money
func (m *Money) Abs() *Money {
	if m.IsNegative() {
		return &Money{
			value:    accounting.Value{Amount: -m.value.Amount},
			currency: m.currency,
		}
	}
	return m
}

// Negate returns the negated amount
func (m *Money) Negate() *Money {
	return &Money{
		value:    accounting.Value{Amount: -m.value.Amount},
		currency: m.currency,
	}
}

// MarshalJSON implements json.Marshaler interface
func (m *Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Float())
}

// UnmarshalJSON implements json.Unmarshaler interface
func (m *Money) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	m.value = accounting.Value{Amount: int64(f * 100)}
	m.currency = "VND"
	return nil
}

// MoneySum calculates the sum of multiple Money amounts
func MoneySum(amounts ...*Money) *Money {
	sum := NewMoney(0)
	for _, amount := range amounts {
		if amount != nil {
			sum = sum.Add(amount)
		}
	}
	return sum
}

// ValidateMoneyAmount validates that a money amount is valid for ledger operations
func ValidateMoneyAmount(amount *Money) error {
	if amount == nil {
		return fmt.Errorf("amount cannot be nil")
	}

	if amount.IsNegative() {
		return fmt.Errorf("amount cannot be negative: %s", amount.String())
	}

	if amount.IsZero() {
		return fmt.Errorf("amount cannot be zero")
	}

	return nil
}
