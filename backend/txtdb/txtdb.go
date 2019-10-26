package txtdb

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cespedes/accounting"
)

type txtDriver struct{}

const refreshTimeout = 5 * time.Second

type conn struct {
	dir          string
	accounts     []*accounting.Account
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

func (c *conn) Accounts() (accounts []*accounting.Account) {
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
		parent, err := strconv.Atoi(fields[5])
		if err == nil {
			ac.Parent = c.accountMap[parent]
		}
		c.accountMap[ac.ID] = &ac
		accounts = append(accounts, &ac)
	}

	c.accounts = accounts
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
	balance := 0
	var tr *accounting.Transaction
	for i := 1; sc.Scan(); i++ {
		if tr == nil {
			tr = new(accounting.Transaction)
		}
		// var sp accounting.Split
		line := sc.Text()
		fields := strings.Split(line, ":")
		if len(fields) != 7 { // badly-formatted line: skip
			continue
		}
		if len(fields[5]) == 0 {
			if balance != 0 {
				log.Printf("transactions line %d: no value inside transaction (balance=%d)", i, balance)
				balance = 0
				tr = nil
				continue
			}
			continue
		}
		if tr.ID == 0 { // Fill tr only if it is not already filled
			// First field (used to be "id") is ignored
			tr.ID = nextID
			tr.Time, err = time.Parse("2006-01-02 15.04", strings.TrimSpace(fields[1]))
			if err != nil {
				tr.Time, err = time.Parse("2006-01-02", strings.TrimSpace(fields[1]))
			}
			if err != nil {
				log.Printf("transactions line %d: datetime error (%s)\n", i, strings.TrimSpace(fields[1]))
				continue
			}
			tr.Description = fields[2]
		}
		accountID, err := strconv.Atoi(fields[4])
		if err != nil {
			log.Printf("transactions line %d: invalid account (%s)", i, fields[4])
			continue
		}
		var sp accounting.Split
		sp.Account = c.accountMap[accountID]
		if sp.Account == nil {
			log.Printf("transactions line %d: invalid account (%s)", i, fields[4])
			continue
		}
		var sign int
		if fields[5][0] == '+' {
			sign = 1
		} else if fields[5][0] == '-' {
			sign = -1
		} else {
			log.Printf("transaction line %d: invalid value (%s)", i, fields[5])
			continue
		}
		f, err := strconv.ParseFloat(fields[5][1:], 64)
		if err != nil {
			log.Printf("transaction line %d: invalid value (%s)", i, fields[5])
			continue
		}
		sp.Value = sign * int(math.Round(100*f))
		sp.Balance = accountID // TODO FIXME XXX
		balance += sp.Value
		tr.Splits = append(tr.Splits, sp)
		if balance == 0 {
			transactions = append(transactions, *tr)
			tr = nil
			nextID++
		}
	}
	c.transactions = transactions
	return
}

func init() {
	accounting.Register("txtdb", txtDriver{})
}
