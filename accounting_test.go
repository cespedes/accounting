package accounting

import (
	"testing"
)

func TestCurrencyString(t *testing.T) {
	var v Value

	if got := v.String(); got != "0" {
		t.Errorf("Money(0) = %q", got)
	}

	v.Amount = 1 * U
	if got := v.String(); got != "1" {
		t.Errorf("Money(1) = %q", got)
	}

	v.Currency = new(Currency)

	v.Amount = 1 * U
	v.Currency.Precision = 1
	if got := v.String(); got != "1.0" {
		t.Errorf("Money(1.0) = %q", got)
	}

	v.Amount = 1 * U
	v.Currency.Precision = 3
	if got := v.String(); got != "1.000" {
		t.Errorf("Money(1.000) = %q", got)
	}

	v.Amount = 1 * U
	v.Currency.Precision = 3
	v.Currency.Decimal = "'"
	if got := v.String(); got != "1'000" {
		t.Errorf("Money(1'000) = %q", got)
	}

	v.Amount = 1.2345 * U
	v.Currency.Precision = 0
	v.Currency.Decimal = "'"
	if got := v.String(); got != "1" {
		t.Errorf("Money(1) = %q", got)
	}

	v.Amount = -1.2345 * U
	v.Currency.Precision = 0
	v.Currency.Decimal = "'"
	if got := v.String(); got != "-1" {
		t.Errorf("Money(-1) = %q", got)
	}

	v.Amount = 1.999 * U
	v.Currency.Precision = 0
	v.Currency.Decimal = "'"
	if got := v.String(); got != "1" {
		t.Errorf("Money(1) = %q", got)
	}

	v.Amount = -1.999 * U
	v.Currency.Precision = 0
	v.Currency.Decimal = "'"
	if got := v.String(); got != "-1" {
		t.Errorf("Money(-1) = %q", got)
	}

	v.Amount = 1.2345 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ","
	if got := v.String(); got != "1,23" {
		t.Errorf("Money(1,23) = %q", got)
	}

	v.Amount = 0.2345 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	if got := v.String(); got != "0.23" {
		t.Errorf("Money(0.23) = %q", got)
	}

	v.Amount = -0.2345 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	if got := v.String(); got != "-0.23" {
		t.Errorf("Money(-0.23) = %q", got)
	}

	v.Amount = 9876.2345 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	if got := v.String(); got != "9876.23" {
		t.Errorf("Money(9876.23) = %q", got)
	}

	v.Amount = 9876.23456 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ""
	v.Currency.Thousand = ","
	if got := v.String(); got != "9,876.23" {
		t.Errorf("Money(9,876.23) = %q", got)
	}

	v.Amount = 12000.99999 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = ","
	if got := v.String(); got != "12,000" {
		t.Errorf("Money(12,000) = %q", got)
	}

	v.Amount = 10 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = ","
	if got := v.String(); got != "10" {
		t.Errorf("Money(10) = %q", got)
	}

	v.Amount = 100 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = ","
	if got := v.String(); got != "100" {
		t.Errorf("Money(100) = %q", got)
	}

	v.Amount = 1000 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = ","
	if got := v.String(); got != "1,000" {
		t.Errorf("Money(1,000) = %q", got)
	}

	v.Amount = 10_000 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = ""
	if got := v.String(); got != "10000" {
		t.Errorf("Money(10000) = %q", got)
	}

	v.Amount = 100_000 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = "."
	if got := v.String(); got != "100.000" {
		t.Errorf("Money(100.000) = %q", got)
	}

	v.Amount = 1_000_000 * U
	v.Currency.Precision = 0
	v.Currency.Thousand = " "
	if got := v.String(); got != "1 000 000" {
		t.Errorf("Money(1 000 000) = %q", got)
	}

	v.Amount = 23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ","
	v.Currency.Name = "€"
	if got := v.String(); got != "23,45€" {
		t.Errorf("Money(23,45€) = %q", got)
	}

	v.Amount = -23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ","
	v.Currency.Name = "€"
	if got := v.String(); got != "-23,45€" {
		t.Errorf("Money(-23,45€) = %q", got)
	}

	v.Amount = 23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = ","
	v.Currency.Name = "EUR"
	v.Currency.PrintSpace = true
	if got := v.String(); got != "23,45 EUR" {
		t.Errorf("Money(23,45 EUR) = %q", got)
	}

	v.Amount = 23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = "."
	v.Currency.Name = "USD"
	v.Currency.PrintBefore = true
	v.Currency.PrintSpace = true
	if got := v.String(); got != "USD 23.45" {
		t.Errorf("Money(USD 23.45) = %q", got)
	}

	v.Amount = -23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = "."
	v.Currency.Name = "USD"
	v.Currency.PrintBefore = true
	v.Currency.PrintSpace = true
	if got := v.String(); got != "USD -23.45" {
		t.Errorf("Money(USD -23.45) = %q", got)
	}

	v.Amount = 23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = "."
	v.Currency.Name = "$"
	v.Currency.PrintBefore = true
	v.Currency.PrintSpace = false
	if got := v.String(); got != "$23.45" {
		t.Errorf("Money($23.45) = %q", got)
	}

	v.Amount = -23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = "."
	v.Currency.Name = "$"
	v.Currency.PrintBefore = true
	v.Currency.PrintSpace = false
	if got := v.String(); got != "$-23.45" {
		t.Errorf("Money($-23.45) = %q", got)
	}

	v.Amount = -23.45 * U
	v.Currency.Precision = 2
	v.Currency.Decimal = "."
	v.Currency.Name = ""
	v.Currency.PrintBefore = true
	v.Currency.PrintSpace = false
	if got := v.String(); got != "-23.45" {
		t.Errorf("Money(-23.45) = %q", got)
	}
}
