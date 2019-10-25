package txtdb

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cespedes/accounting"
)

type txtDriver struct{}

const (
	refreshTimeout = 5 * time.Second
)

type conn struct {
	dir          string
	accounts     []accounting.Account
	accountMap   map[int]*accounting.Account
	transactions []accounting.Transaction
	updated      time.Time
}

// Opens a connection to a txtdb database
func (p txtDriver) Open(name string) (accounting.Conn, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	conn := new(conn)
	conn.dir = url.Path
	conn.accountMap = make(map[int]*accounting.Account)
	tra := filepath.Join(conn.dir, "transactions")
	acc := filepath.Join(conn.dir, "accounts")
	if fi, err := os.Stat(tra); err != nil {
		return nil, err
	} else {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%s: %w", acc, err)
		}
	}
	if fi, err := os.Stat(acc); err != nil {
		return nil, err
	} else {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%s: %w", acc, err)
		}
	}
	return conn, nil
}

func (c *conn) Close() error {
	return nil
}

func (c *conn) Flush() error {
	return errors.New("unimplemented")
}

func (c *conn) Accounts() (accounts []accounting.Account) {
	t := time.Now()
	if t.Sub(c.updated) < refreshTimeout && c.accounts != nil {
		return c.accounts
	}
	f, err := os.Open(filepath.Join(c.dir, "accounts"))
	if err != nil {
		return nil
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ac accounting.Account
		line := sc.Text()
		fields := strings.Split(line, ":")
		if len(fields) != 6 { // badly-formatted line: skip
			continue
		}
		ac.ID, err = strconv.Atoi(fields[0])
		if err != nil { // TODO: handle errors
			continue
		}
		ac.Name = fields[3]
		ac.Code = fields[4]
		accounts = append(accounts, ac)
	}

	c.accounts = accounts
	for _, a := range accounts {
		c.accountMap[a.ID] = &a
	}
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
	f, err := os.Open(filepath.Join(c.dir, "transactions"))
	if err != nil {
		return nil
	}
	sc := bufio.NewScanner(f)
	nextID := 1
	// balance := 0
	var tr *accounting.Transaction
	for i := 0; sc.Scan(); i++ {
		if tr == nil {
			tr = new(accounting.Transaction)
		}
		// var sp accounting.Split
		line := sc.Text()
		fields := strings.Split(line, ":")
		if len(fields) != 7 { // badly-formatted line: skip
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			id = nextID
		}
		if tr.ID == 0 {
			tr.ID = id
		}
		if id != tr.ID {
			panic("id != tr.ID")
		}
		tr.Time, err = time.Parse("2006-01-02 15.04", fields[1])
		if err != nil {
			tr.Time, err = time.Parse("2006-01-02", fields[1])
		}
		if err != nil { // TODO: handle errors
			continue
		}
		tr.Description = fields[2]
		transactions = append(transactions, *tr)
		tr = nil
	}

	c.transactions = transactions
	return
}

func init() {
	accounting.Register("txtdb", txtDriver{})
}
