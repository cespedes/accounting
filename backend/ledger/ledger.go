package ledger

import (
	"net/url"

	"github.com/cespedes/accounting"
)

type driver struct{}

func init() {
	accounting.Register("ledger", driver{})
}

type ledger struct {
	file            string
	ready           bool
	accounts        []*accounting.Account
	transactions    []*accounting.Transaction
	currencies      []*accounting.Currency
	prices          []accounting.Price
	defaultCurrency *accounting.Currency
}

func (driver) Open(name string) (accounting.Conn, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	ledger := new(ledger)
	ledger.file = url.Path
	return ledger, nil
}

func (l *ledger) Accounts() (accounts []*accounting.Account) {
	l.Read()
	return l.accounts
}

func (l *ledger) Transactions() (transactions []*accounting.Transaction) {
	l.Read()
	return l.transactions
}

func (l *ledger) Prices() []accounting.Price {
	return l.prices
}

func (l *ledger) Close() error {
	return nil
}
