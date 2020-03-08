package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cespedes/accounting"

	_ "github.com/lib/pq" // This package is just for PostgreSQL
)

type driver struct{}

type ID int

type conn struct {
	db      *sql.DB
	updated time.Time
	ledger  *accounting.Ledger
}

func (id ID) String() string {
	return fmt.Sprintf("%d", id)
}

func init() {
	accounting.Register("postgres", driver{})
}

const refreshTimeout = 5 * time.Second

func (driver) Open(name string, ledger *accounting.Ledger, _ *accounting.BackendLedger) (accounting.Connection, error) {
	db, err := sql.Open("postgres", name)
	if err != nil {
		return nil, errors.New("psql.Open: " + err.Error())
	}
	if err = db.Ping(); err != nil {
		return nil, errors.New("psql.Open: " + err.Error())
	}
	// TODO I should check the SQL schema...
	conn := new(conn)
	conn.db = db
	getAccounts(conn, ledger)
	getTransactions(conn, ledger)
	return conn, nil
}

func (c *conn) Refresh() {
	// TODO: do something
}

func (c *conn) Close() error {
	return c.db.Close()
}

func getAccounts(c *conn, ledger *accounting.Ledger) {
	query := `
		SELECT a.id, a.name, COALESCE(a.code, '') AS code,
			COALESCE((100*sum(s.value))::integer, 0) AS balance
		FROM account a
		LEFT JOIN split s ON a.id=s.account_id GROUP BY a.id
	`
	rows, err := c.db.Query(query)
	if err != nil {
		panic(err)
	}
	ledger.Accounts = nil
	for rows.Next() {
		var (
			id      int
			name    string
			code    string
			balance int
			acc     accounting.Account
		)
		if err := rows.Scan(&id, &name, &code, &balance); err != nil {
			panic(err)
		}
		acc.ID = ID(id)
		acc.Name = name
		acc.Code = code
		// acc.Balance = balance
		ledger.Accounts = append(ledger.Accounts, &acc)
	}
}

func getTransactions(c *conn, ledger *accounting.Ledger) {
	idAccount := make(map[accounting.ID]*accounting.Account)
	for i, a := range ledger.Accounts {
		idAccount[a.ID] = ledger.Accounts[i]
	}
	query := `
		SELECT datetime,transaction_id,account_id,description,(100*value)::integer,(100*balance)::integer FROM money
	`
	rows, err := c.db.Query(query)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var (
			date    time.Time
			tid     ID
			aid     ID
			desc    string
			value   int64
			balance int
		)
		if err := rows.Scan(&date, &tid, &aid, &desc, &value, &balance); err != nil {
			panic(err)
		}
		if l := len(ledger.Transactions); l == 0 || ledger.Transactions[l-1].ID != tid {
			ledger.Transactions = append(ledger.Transactions, &accounting.Transaction{
				ID:          tid,
				Time:        date,
				Description: desc})
		}
		split := new(accounting.Split)
		split.Account = idAccount[aid]
		split.Value.Currency = nil
		split.Value.Amount = value
		tra := ledger.Transactions[len(ledger.Transactions)-1]
		tra.Splits = append(tra.Splits, split)
	}
	return
}

// Flush is a no-op in SQL: all the writes to the database are unbuffered
func (c *conn) Flush() error {
	return nil
}

func (c *conn) Display(out io.Writer) {
	// TODO FIXME XXX
	fmt.Fprintln(out, "/* Unimplemented */")
}
