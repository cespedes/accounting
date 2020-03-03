package ledger

import (
	"net/url"

	"github.com/cespedes/accounting"
)

type driver struct{}

func init() {
	accounting.Register("ledger", driver{})
}

type ledgerConnection struct {
	file            string
	defaultCurrency *accounting.Currency
}

func (driver) Open(name string, ledger *accounting.Ledger) (accounting.Connection, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	conn := new(ledgerConnection)
	conn.file = url.Path
	conn.readJournal(url.Path, ledger)
	return conn, nil
}

func (conn *ledgerConnection) Close() error {
	return nil
}

func (conn *ledgerConnection) Refresh(ledger *accounting.Ledger) {
	// TODO FIXME XXX: notifier
}
