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
	backend         *accounting.Backend
	ledger          *accounting.Ledger
}

func (driver) Open(name string, backend *accounting.Backend) (accounting.Connection, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	conn := new(ledgerConnection)
	conn.file = url.Path
	conn.backend = backend
	conn.ledger = backend.Ledger
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
		if len(ledger.Comments[a]) > 0 {
			comment = " ; " + ledger.Comments[a][0]
		}
		fmt.Fprintf(out, "account %s%s\n", a.FullName(), comment)
		if len(ledger.Comments[a]) > 1 {
			for _, c := range ledger.Comments[a][1:] {
				fmt.Fprintf(out, "\t: %s\n", c)
			}
		}
	}
	fmt.Fprintln(out, "\n; Currencies:")
	for _, cu := range ledger.Currencies {
		comment := ""
		if len(ledger.Comments[cu]) > 0 {
			comment = " ; " + ledger.Comments[cu][0]
		}
		var v accounting.Value
		v.Amount = 1_000_000 * accounting.U
		v.Currency = cu
		fmt.Fprintf(out, "commodity %s%s\n", v.String(), comment)
		if len(ledger.Comments[cu]) > 1 {
			for _, c := range ledger.Comments[cu][1:] {
				fmt.Fprintf(out, "\t; %s\n", c)
			}
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
			p = ledger.Prices[j]
			tp = p.Time
		}
		// fmt.Fprintf(out, "DEBUG: i=%d j=%d tt=%v tp=%v\n", i, j, tt, tp)
		if p == nil || (t != nil && !tt.After(tp)) {
			i++
			comment := ""
			if len(ledger.Comments[t]) > 0 {
				comment = " ; " + ledger.Comments[t][0]
			}
			fmt.Fprintf(out, "%s %s%s\n", t.Time.Format("2006-01-02/15:04"), t.Description, comment)
			if len(ledger.Comments[t]) > 1 {
				for _, c := range ledger.Comments[t][1:] {
					fmt.Fprintf(out, "\t; %s\n", c)
				}
			}
			for _, s := range t.Splits {
				comment = ""
				if len(ledger.Comments[s]) > 0 {
					comment = " ; " + ledger.Comments[s][0]
				}
				fmt.Fprintf(out, "  %-50s  %s", s.Account.FullName(), s.Value.FullString())
				if len(ledger.Comments[s]) > 1 {
					for _, c := range ledger.Comments[s][1:] {
						fmt.Fprintf(out, "\t; %s\n", c)
					}
				}
				if v, ok := ledger.SplitPrices[s]; ok == true {
					fmt.Fprintf(out, " @@ %s", v.FullString())
				}
				fmt.Fprintf(out, "%s\n", comment)
			}
		} else {
			j++
			comment := ""
			if len(ledger.Comments[p]) > 0 {
				comment = " ; " + ledger.Comments[p][0]
			}
			fmt.Fprintf(out, "P %s %s %s%s\n", p.Time.Format("2006-01-02/15:04"), p.Currency.Name, p.Value.FullString(), comment)
			if len(ledger.Comments[p]) > 1 {
				for _, c := range ledger.Comments[p][1:] {
					fmt.Fprintf(out, "\t; %s\n", c)
				}
			}
		}
	}
}

func (conn *ledgerConnection) Display(out io.Writer) {
	Display(out, conn.ledger)
}
