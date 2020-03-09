package ledger

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/cespedes/accounting"
)

type driver struct{}

func init() {
	accounting.Register("ledger", driver{})
}

type ledgerConnection struct {
	file            string
	defaultCurrency *accounting.Currency
	ledger          *accounting.Ledger
}

func (driver) Open(name string, ledger *accounting.Ledger, _ *accounting.BackendLedger) (accounting.Connection, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	conn := new(ledgerConnection)
	conn.file = url.Path
	conn.ledger = ledger
	conn.readJournal()
	return conn, nil
}

func (conn *ledgerConnection) Close() error {
	return nil
}

func (conn *ledgerConnection) Refresh() {
	// TODO FIXME XXX: notifier
}

func Display(out io.Writer, ledger *accounting.Ledger) {
	fmt.Fprintln(out, "\n; Accounts:")
	for _, a := range ledger.Accounts {
		comment := ""
		if len(a.Comments) > 0 {
			comment = " ; " + a.Comments[0]
			a.Comments = a.Comments[1:]
		}
		fmt.Fprintf(out, "account %s%s\n", a.FullName(), comment)
		for _, c := range a.Comments {
			fmt.Fprintf(out, "\t; %s\n", c)
		}
	}
	fmt.Fprintln(out, "\n; Currencies:")
	for _, cu := range ledger.Currencies {
		comment := ""
		if len(cu.Comments) > 0 {
			comment = " ; " + cu.Comments[0]
			cu.Comments = cu.Comments[1:]
		}
		var v accounting.Value
		v.Amount = 1_000_000 * accounting.U
		v.Currency = cu
		fmt.Fprintf(out, "commodity %s%s\n", v.String(), comment)
		for _, c := range cu.Comments {
			fmt.Fprintf(out, "\t; %s\n", c)
		}
	}
	fmt.Fprintln(out, "\n; Transactions and prices:")
	var i, j int
	for i < len(ledger.Transactions) || j < len(ledger.Prices) {
		var t *accounting.Transaction
		var p *accounting.Price
		var tt, tp time.Time
		if i < len(ledger.Transactions) {
			t = ledger.Transactions[i]
			tt = t.Time
		}
		if j < len(ledger.Prices) {
			p = &ledger.Prices[j]
			tp = p.Time
		}
		// fmt.Fprintf(out, "DEBUG: i=%d j=%d tt=%v tp=%v\n", i, j, tt, tp)
		if p == nil || (t != nil && !tt.After(tp)) {
			i++
			comment := ""
			if len(t.Comments) > 0 {
				comment = " ; " + t.Comments[0]
				t.Comments = t.Comments[1:]
			}
			fmt.Fprintf(out, "%s %s%s\n", t.Time.Format("2006-01-02/15:04"), t.Description, comment)
			for _, c := range t.Comments {
				fmt.Fprintf(out, "\t; %s\n", c)
			}
			for _, s := range t.Splits {
				comment = ""
				if len(s.Comments) > 0 {
					comment = " ; " + s.Comments[0]
					s.Comments = s.Comments[1:]
				}
				fmt.Fprintf(out, "  %-50s %s", s.Account.FullName(), s.Value.FullString())
				for _, c := range s.Comments {
					fmt.Fprintf(out, "\t; %s\n", c)
				}
				if s.EqValue != nil {
					fmt.Fprintf(out, " @@ %s", s.EqValue.FullString())
				}
				fmt.Fprintf(out, "%s\n", comment)
			}
		} else {
			j++
			comment := ""
			if len(p.Comments) > 0 {
				comment = " ; " + p.Comments[0]
				p.Comments = p.Comments[1:]
			}
			fmt.Fprintf(out, "P %s %s %s%s\n", p.Time.Format("2006-01-02/15:04"), p.Currency.Name, p.Value.FullString(), comment)
			for _, c := range p.Comments {
				fmt.Fprintf(out, "\t; %s\n", c)
			}
		}
	}
}

func (conn *ledgerConnection) Display(out io.Writer) {
	Display(out, conn.ledger)
}
