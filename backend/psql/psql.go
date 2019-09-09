/*
Package psql is a Postgres driver for the github.com/cespedes/accounting package.

You just have to include github.com/cespedes/accounting and this package with a blank
identifier to begin using it:

	import (
		"github.com/cespedes/accounting"

		_ "github.com/cespedes/accounting/backend/psql"
	)

	func main() {
		connStr := "host=localhost user=pqgotest dbname=pqgotest password=secret"
		ledger, err := accounting.Open("psql", connStr)
		if err != nil {
			panic(err)
		}

		accounts := ledger.Accounts()
		transactions := ledger.Transactions()
		…
	}

This package uses github.com/lib/pq so you can use the same syntax to connect to the database.

The database to connect must already exist, and must have these tables:

	CREATE TABLE account (
	  id        SERIAL PRIMARY KEY,
	  parent_id INTEGER REFERENCES account(id),
	  name      TEXT,
	  code      TEXT
	);

	CREATE TABLE transaction (
	  id          SERIAL PRIMARY KEY,
	  datetime    TIMESTAMP WITHOUT TIME ZONE NOT NULL,
	  description TEXT
	);

	CREATE TABLE split (
	  transaction_id INTEGER NOT NULL REFERENCES transaction(id),
	  account_id     INTEGER NOT NULL REFERENCES account(id),
	  value          NUMERIC
	);
*/
package psql

import (
	"errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/cespedes/accounting"
)

type psqlDriver struct {}

func (p psqlDriver) Open(name string) (accounting.Conn, error) {
	db, err := sqlx.Open("postgres", name)
	if err != nil {
		return nil, errors.New("psql.Open: " + err.Error())
	}
	if err = db.Ping(); err != nil {
		return nil, errors.New("psql.Open: " + err.Error())
	}
	// Now, let's check the SQL schema...
	// TODO
	conn := new(conn)
	conn.db = db
	return conn, nil
}

type conn struct{
	db *sqlx.DB
}

func (c *conn) Close() error {
	return nil
}

func (c *conn) Accounts() (result []accounting.Account) {
	query := `
		SELECT a.id,a.name,coalesce(a.code,'') as code,coalesce((100*sum(s.value))::integer,0) as balance from account a left join split s on a.id=s.account_id group by a.id
	`
	err := c.db.Select(&result, query)
	if err != nil {
		panic(err)
	}
	return
}

func (c *conn) Transactions() []accounting.Transaction {
	return nil
}

func init() {
	accounting.Register("psql", psqlDriver{})
}
