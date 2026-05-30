package accounting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoney_NewMoney(t *testing.T) {
	m := NewMoney(1000.50)
	assert.Equal(t, 1000.50, m.Float())
	assert.Equal(t, int64(100050), m.Int64()) // Assuming it stores in cents
}

func TestMoney_NewMoneyFromInt(t *testing.T) {
	m := NewMoneyFromInt(100050) // 1000.50 in cents
	assert.Equal(t, 1000.50, m.Float())
	assert.Equal(t, int64(100050), m.Int64())
}

func TestMoney_Float(t *testing.T) {
	m := NewMoney(1000.50)
	assert.Equal(t, 1000.50, m.Float())
}

func TestMoney_Int64(t *testing.T) {
	m := NewMoneyFromInt(100050) // 1000.50 in cents
	assert.Equal(t, int64(100050), m.Int64())
}

func TestMoney_String(t *testing.T) {
	m := NewMoney(1000.50)
	expected := "1000.50 ₫"
	assert.Equal(t, expected, m.String())
}

func TestMoney_Add(t *testing.T) {
	m1 := NewMoney(100.00)
	m2 := NewMoney(50.00)
	result := m1.Add(m2)
	assert.Equal(t, 150.00, result.Float())
}

func TestMoney_Subtract(t *testing.T) {
	m1 := NewMoney(100.00)
	m2 := NewMoney(30.00)
	result := m1.Subtract(m2)
	assert.Equal(t, 70.00, result.Float())
}

func TestMoney_Multiply(t *testing.T) {
	m := NewMoney(100.00)
	result := m.Multiply(1.5)
	assert.Equal(t, 150.00, result.Float())
}

func TestMoney_IsZero(t *testing.T) {
	m := NewMoney(0.00)
	assert.True(t, m.IsZero())

	m = NewMoney(1.00)
	assert.False(t, m.IsZero())
}

func TestMoney_IsPositive(t *testing.T) {
	m := NewMoney(1.00)
	assert.True(t, m.IsPositive())

	m = NewMoney(-1.00)
	assert.False(t, m.IsPositive())
}

func TestMoney_IsNegative(t *testing.T) {
	m := NewMoney(-1.00)
	assert.True(t, m.IsNegative())

	m = NewMoney(1.00)
	assert.False(t, m.IsNegative())
}

func TestMoney_Equal(t *testing.T) {
	m1 := NewMoney(100.00)
	m2 := NewMoney(100.00)
	assert.True(t, m1.Equal(m2))

	m3 := NewMoney(200.00)
	assert.False(t, m1.Equal(m3))
}

func TestMoney_GreaterThan(t *testing.T) {
	m1 := NewMoney(200.00)
	m2 := NewMoney(100.00)
	assert.True(t, m1.GreaterThan(m2))
	assert.False(t, m2.GreaterThan(m1))
}

func TestMoney_LessThan(t *testing.T) {
	m1 := NewMoney(100.00)
	m2 := NewMoney(200.00)
	assert.True(t, m1.LessThan(m2))
	assert.False(t, m2.LessThan(m1))
}

func TestMoney_Abs(t *testing.T) {
	m := NewMoney(-100.00)
	abs := m.Abs()
	assert.Equal(t, 100.00, abs.Float())
	assert.Equal(t, NewMoney(100.00).Float(), abs.Float())
}

func TestMoney_Negate(t *testing.T) {
	m := NewMoney(100.00)
	neg := m.Negate()
	assert.Equal(t, -100.00, neg.Float())
}

func TestMoney_ValidateMoneyAmount(t *testing.T) {
	// Test valid amount
	err := ValidateMoneyAmount(NewMoney(100.00))
	assert.NoError(t, err)

	// Test negative amount
	err = ValidateMoneyAmount(NewMoney(-100.00))
	assert.Error(t, err)

	// Test zero amount
	err = ValidateMoneyAmount(NewMoney(0.00))
	assert.Error(t, err)
}

func TestMoney_MoneySum(t *testing.T) {
	m1 := NewMoney(100.00)
	m2 := NewMoney(50.00)
	m3 := NewMoney(25.00)

	total := MoneySum(m1, m2, m3)
	assert.Equal(t, 175.00, total.Float())

	// Test with an empty slice
	total = MoneySum()
	assert.Equal(t, 0.00, total.Float())
}
