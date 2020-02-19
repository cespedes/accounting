package ledger

import (
	"net/url"

	"github.com/cespedes/accounting"
)

type Driver struct{}

func init() {
	accounting.Register("ledger", Driver{})
}

type ledger struct {
	file         string
	accounts     []accounting.Account
	transactions []accounting.Transaction
	currencies   []accounting.Currency
	prices       []accounting.Price
}

func (Driver) Open(name string) (accounting.Conn, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	ledger := new(ledger)
	ledger.file = url.Path
	return ledger, nil
}

func (l *ledger) Accounts() (accounts []*accounting.Account) {
	return nil
}

func (l *ledger) Transactions() (transactions []accounting.Transaction) {
	l.Read()
	return l.transactions
}

func (l *ledger) Close() error {
	return nil
}
