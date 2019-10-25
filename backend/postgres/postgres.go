package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/cespedes/accounting"

	_ "github.com/lib/pq" // This package is just for PostgreSQL
)

type driver struct{}

func init() {
	accounting.Register("postgres", driver{})
}

const refreshTimeout = 5 * time.Second

func (driver) Open(name string) (accounting.Conn, error) {
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
	return conn, nil
}

type conn struct {
	db           *sql.DB
	accounts     []accounting.Account
	transactions []accounting.Transaction
	updated      time.Time
}

func (c *conn) Close() error {
	return c.db.Close()
}

func (c *conn) Accounts() (result []accounting.Account) {
	t := time.Now()
	if t.Sub(c.updated) < refreshTimeout && c.accounts != nil {
		return c.accounts
	}
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
		acc.ID = id
		acc.Name = name
		acc.Code = code
		// acc.Balance = balance
		result = append(result, acc)
	}
	c.accounts = result
	c.updated = time.Now()
	return
}

func (c *conn) Transactions() (transactions []accounting.Transaction) {
	t := time.Now()
	if t.Sub(c.updated) > refreshTimeout {
		c.Accounts()
	} else if c.transactions != nil {
		return c.transactions
	}
	idAccount := make(map[int]*accounting.Account)
	for i, a := range c.accounts {
		idAccount[a.ID] = &c.accounts[i]
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
			tid     int
			aid     int
			desc    string
			value   int
			balance int
		)
		if err := rows.Scan(&date, &tid, &aid, &desc, &value, &balance); err != nil {
			panic(err)
		}
		if l := len(transactions); l == 0 || transactions[l-1].ID != tid {
			transactions = append(transactions, accounting.Transaction{
				ID:          tid,
				Time:        date,
				Description: desc})
		}
		var split accounting.Split
		split.Account = idAccount[aid]
		split.Value = value
		split.Balance = balance
		tra := &transactions[len(transactions)-1]
		tra.Splits = append(tra.Splits, split)
	}
	c.transactions = transactions
	return
}

// Flush is a no-op in SQL: all the writes to the database are unbuffered
func (c *conn) Flush() error {
	return nil
}
