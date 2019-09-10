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
	"database/sql"
	"time"
	_ "github.com/lib/pq"
	"github.com/cespedes/accounting"
)

type psqlDriver struct {}

const (
	RefreshTimeout = 5 * time.Second
)

func (p psqlDriver) Open(name string) (accounting.Conn, error) {
	db, err := sql.Open("postgres", name)
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
	db *sql.DB
	accounts []accounting.Account
	transactions []accounting.Transaction
	updated time.Time
}

func (c *conn) Close() error {
	return c.db.Close()
}

func (c *conn) Accounts() (result []accounting.Account) {
	t := time.Now()
	if t.Sub(c.updated) < RefreshTimeout && c.accounts != nil {
		return c.accounts
	}
	query := `
		SELECT a.id,a.name,coalesce(a.code,'') as code,coalesce((100*sum(s.value))::integer,0) as balance from account a left join split s on a.id=s.account_id group by a.id
	`
	rows, err := c.db.Query(query)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var (
			id int
			name string
			code string
			balance int
			acc accounting.Account
		)
		if err := rows.Scan(&id, &name, &code, &balance); err != nil {
			panic(err)
		}
		acc.Id = id
		acc.Name = name
		acc.Code = code
		acc.Balance = balance
		result = append(result, acc)
	}
	c.accounts = result
	c.updated = time.Now()
	return
}

func (c *conn) Transactions() (transactions []accounting.Transaction) {
	t := time.Now()
	if t.Sub(c.updated) > RefreshTimeout {
		c.Accounts()
	} else if c.transactions != nil {
		return c.transactions
	}
	idAccount := make(map[int]*accounting.Account)
	for i, a := range c.accounts {
		idAccount[a.Id] = &c.accounts[i]
	}
	query := `SELECT datetime,transaction_id,account_id,description,(100*value)::integer,(100*balance)::integer from money`
	rows, err := c.db.Query(query)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var (
			date time.Time
			tid int
			aid int
			desc string
			value int
			balance int
			tra *accounting.Transaction
			split accounting.Split
		)
		if err := rows.Scan(&date, &tid, &aid, &desc, &value, &balance); err != nil {
			panic(err)
		}
		if l := len(transactions); l == 0 || transactions[l-1].Id != tid {
			transactions = append(transactions, accounting.Transaction{
				Id:tid,
				Time:date,
				Description:desc})
		}
		tra = &transactions[len(transactions)-1]
		split.Account = idAccount[aid]
		split.Value = value
		split.Balance = balance
		tra.Splits = append(tra.Splits, split)
	}
	c.transactions = transactions
	return
}

func init() {
	accounting.Register("psql", psqlDriver{})
}
