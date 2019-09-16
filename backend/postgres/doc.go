/*
Package postgres is a Postgres driver for the github.com/cespedes/accounting package.

You just have to include github.com/cespedes/accounting and this package with a blank
identifier to begin using it:

	import (
		"github.com/cespedes/accounting"

		_ "github.com/cespedes/accounting/backend/postgres"
	)

	func main() {
		connStr := "host=localhost user=pqgotest dbname=pqgotest password=secret"
		ledger, err := accounting.Open("postgres", connStr)
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
package postgres
