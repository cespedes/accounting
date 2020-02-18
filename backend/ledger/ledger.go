package ledger

import (
	"net/url"

	"github.com/cespedes/accounting"
)

type LedgerDriver struct{}

func init() {
	accounting.Register("ledger", LedgerDriver{})
}

type ledger struct {
	file         string
	accounts     []*accounting.Account
	transactions []*accounting.Transaction
	currencies   []accounting.Currency
	prices       []accounting.Price
}

func (LedgerDriver) Open(name string) (accounting.Conn, error) {
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
	return nil
}

func (l *ledger) Transactions() (transactions []accounting.Transaction) {
	return nil
}

func (l *ledger) Close() error {
	return nil
}
