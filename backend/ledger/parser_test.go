package ledger

import (
	"testing"
)

type testCase struct {
	input  string
	output string
	err    bool
}

var testCases [][]testCase = [][]testCase{
	{
		{"1.000", "", true},
		{"1.000.000", "1.000.000", false},
		{"1eur", "1eur", false},
		{"eur1", "1eur", false},
		{"25", "25", false},
		{"blah", "", true},
	},
	{
		{"1'000'000", "1'000'000", false},
		{"234", "234", false},
		{"1000", "1'000", false},
		{"1000'000", "", true},
		{"1.234'5  gbp", "1.234'5 gbp", false},
		{"1 SP500", "1 SP500", false},
	},
	{
		{"$1.23", "$1.23", false},
		{"1.2345 $", "$1.23", false},
	},
}

func TestGetValue(t *testing.T) {
	for _, cc := range testCases {
		l := ledger{}
		for _, c := range cc {
			v, e := l.getValue(c.input)
			if c.err && e == nil {
				t.Errorf("getValue(%q) = %q (expected failure)", c.input, v.String())
				t.Logf("  (amount = %d, currency=%#v)", v.Amount, v.Currency)
				continue
			}
			if !c.err && e != nil {
				t.Errorf("getValue(%q) failed with: \"%s\" (expected %q)", c.input, e.Error(), c.output)
				continue
			}
			if c.err {
				// t.Logf("OK: Value(%q) = Error(%q)", c.input, e.Error())
				continue
			}
			if c.output != v.String() {
				t.Errorf("Value(%q) = %q (expected %q)", c.input, v.String(), c.output)
				t.Logf("  (amount = %d, currency=%#v)", v.Amount, v.Currency)
				continue
			}
			// t.Logf("OK: Value(%q) = %q", c.input, c.output)
		}
	}
}
