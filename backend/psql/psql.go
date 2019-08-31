package psql

import (
	"errors"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/cespedes/accounting"
)

type psqlDriver struct {}

func (p psqlDriver) Open(name string) (accounting.Conn, error) {
	return nil, errors.New("Not implemented")
}

type conn struct{
	db *sql.DB
}

func (c conn) Accounts() []accounting.Account {
	return nil
}

func (c conn) Transactions() []accounting.Transaction {
	return nil
}

func init() {
	accounting.Register("psql", psqlDriver{})
}
