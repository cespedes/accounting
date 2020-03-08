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
	ledger          *accounting.Ledger
}

func (driver) Open(name string, ledger *accounting.Ledger) (accounting.Connection, error) {
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
