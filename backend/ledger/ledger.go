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

// Export shows the "Ledger" representation of an accounting ledger.
func Export(out io.Writer, ledger *accounting.Ledger) {
	fmt.Fprintln(out, "\n; Accounts:")
	for _, a := range ledger.Accounts {
		fmt.Fprintf(out, "account %s", a.FullName())
		if len(ledger.Comments[a]) > 0 {
			fmt.Fprintf(out, " ; %s", ledger.Comments[a][0])
		}
		fmt.Fprint(out, "\n")
		if len(ledger.Comments[a]) > 1 {
			for _, c := range ledger.Comments[a][1:] {
				fmt.Fprintf(out, "\t: %s\n", c)
			}
		}
	}
	fmt.Fprintln(out, "\n; Currencies:")
	for _, cu := range ledger.Currencies {
		var v accounting.Value
		v.Amount = 1_000_000 * accounting.U
		v.Currency = cu
		fmt.Fprintf(out, "commodity %s", v.String())
		if len(ledger.Comments[cu]) > 0 {
			fmt.Fprintf(out, " ; %s", ledger.Comments[cu][0])
		}
		fmt.Fprint(out, "\n")
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
			fmt.Fprintf(out, "%s %s", t.Time.Format("2006-01-02/15:04"), t.Description)
			if len(ledger.Comments[t]) > 0 {
				fmt.Fprintf(out, " ; %s", ledger.Comments[t][0])
			}
			fmt.Fprint(out, "\n")
			if len(ledger.Comments[t]) > 1 {
				for _, c := range ledger.Comments[t][1:] {
					fmt.Fprintf(out, "\t; %s\n", c)
				}
			}
			for _, s := range t.Splits {
				fmt.Fprintf(out, "  %-50s  %s", s.Account.FullName(), s.Value.FullString())
				if v, ok := ledger.SplitPrices[s]; ok == true {
					fmt.Fprintf(out, " @@ %s", v.FullString())
				}
				if v, ok := ledger.Assertions[s]; ok == true {
					fmt.Fprintf(out, " = %s", v.FullString())
				}
				var comments []string
				if *s.Time != t.Time {
					comments = append(comments, "date:"+s.Time.Format("2006-01-02/15:04"))
				}
				if len(ledger.Comments[s]) > 0 {
					comments = append(comments, ledger.Comments[s]...)
				}
				if len(comments) > 0 {
					fmt.Fprintf(out, " ; %s", comments[0])
				}
				fmt.Fprint(out, "\n")
				if len(comments) > 1 {
					for _, c := range comments[1:] {
						fmt.Fprintf(out, "\t; %s\n", c)
					}
				}
			}
		} else {
			j++
			fmt.Fprintf(out, "P %s %s %s", p.Time.Format("2006-01-02/15:04"), p.Currency.Name, p.Value.FullString())
			if len(ledger.Comments[p]) > 0 {
				fmt.Fprintf(out, " ; %s", ledger.Comments[p][0])
			}
			fmt.Fprint(out, "\n")
			if len(ledger.Comments[p]) > 1 {
				for _, c := range ledger.Comments[p][1:] {
					fmt.Fprintf(out, "\t; %s\n", c)
				}
			}
		}
	}
}
