package accounting

import (
	"testing"
)

func TestMoney(t *testing.T) {
	var v Value

	if got := Money(v); got != "0" {
		t.Errorf("Money(0) = %s", got)
	}

	v.Amount = 100_000_000
	if got := Money(v); got != "1" {
		t.Errorf("Money(1) = %s", got)
	}

	v.Currency = new(Currency)

	v.Amount = 100_000_000
	v.Currency.Precision = 1
	if got := Money(v); got != "1.0" {
		t.Errorf("Money(1.0) = %s", got)
	}

	v.Amount = 100_000_000
	v.Currency.Precision = 3
	if got := Money(v); got != "1.000" {
		t.Errorf("Money(1.000) = %s", got)
	}

	v.Amount = 100_000_000
	v.Currency.Precision = 3
	v.Currency.Decimal = "'"
	if got := Money(v); got != "1'000" {
		t.Errorf("Money(1'000) = %s", got)
	}

	v.Amount = 123_456_789
	v.Currency.Precision = 0
	v.Currency.Decimal = "'"
	if got := Money(v); got != "1" {
		t.Errorf("Money(1) = %s", got)
	}

	v.Amount = 123_456_789
	v.Currency.Precision = 2
	v.Currency.Decimal = ","
	if got := Money(v); got != "1,23" {
		t.Errorf("Money(1,23) = %s", got)
	}

	v.Amount = 23_456_789
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	if got := Money(v); got != "0.23" {
		t.Errorf("Money(0.23) = %s", got)
	}

	v.Amount = 9876_23_456_789
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	if got := Money(v); got != "9876.23" {
		t.Errorf("Money(9876.23) = %s", got)
	}

	v.Amount = 9876_23_456_789
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	v.Currency.Thousand = ","
	if got := Money(v); got != "9,876.23" {
		t.Errorf("Money(9,876.23) = %s", got)
	}

	v.Amount = 12_000_99_999_999
	v.Currency.Precision = 0
	v.Currency.Thousand = ","
	if got := Money(v); got != "12,000" {
		t.Errorf("Money(12,000) = %s", got)
	}
}
