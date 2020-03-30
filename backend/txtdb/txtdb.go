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

type driver struct{}

const refreshTimeout = 5 * time.Second

type conn struct {
	dir        string
	backend    *accounting.Backend
	ledger     *accounting.Ledger
	accountMap map[int]*accounting.Account
	currency   accounting.Currency // just one currency for now
}

type ID int

func (id ID) String() string {
	return fmt.Sprintf("id:%d", id)
}

// Opens a connection to a txtdb database
func (p driver) Open(name string, backend *accounting.Backend) (accounting.Connection, error) {
	url, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	conn := new(conn)
	conn.dir = url.Path
	conn.accountMap = make(map[int]*accounting.Account)
	conn.currency.Precision = 2
	conn.backend = backend
	conn.ledger = backend.Ledger
	conn.ledger.Comments = make(map[interface{}][]string)
	conn.ledger.SplitPrices = make(map[*accounting.Split]accounting.Value)
	conn.ledger.Assertions = make(map[*accounting.Split]accounting.Value)

	err = conn.read()
	return conn, err
}

func (c *conn) Close() error {
	return nil
}

func (c *conn) Refresh() {
	// TODO FIXME XXX: notifier
}

func (c *conn) Flush() error {
	return errors.New("unimplemented")
}

func (c *conn) read() error {
	f, err := os.Open(filepath.Join(c.dir, "accounts"))
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ac accounting.Account
		line := sc.Text()
		// TODO: handle comments
		fields := strings.Split(line, ":")
		if len(fields) != 6 { // badly-formatted line: skip
			// TODO: show error
			continue
		}
		var id int
		id, err = strconv.Atoi(fields[0])
		if err != nil { // TODO: handle errors
			continue
		}
		ac.ID = ID(id)
		ac.Name = fields[3]
		ac.Code = fields[4]
		parent, err := strconv.Atoi(fields[5])
		if err == nil {
			ac.Parent = c.accountMap[parent]
		}
		c.accountMap[id] = &ac
		c.ledger.Accounts = append(c.ledger.Accounts, &ac)
	}
	c.ledger.Accounts = accounting.SortAccounts(c.ledger.Accounts)

	f, err = os.Open(filepath.Join(c.dir, "transactions"))
	if err != nil {
		return nil
	}
	sc = bufio.NewScanner(f)
	nextID := 1
	var balance int64
	var tr *accounting.Transaction
	var oldTime, thisTime time.Time
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
		thisTime, err = time.Parse("2006-01-02 15.04", strings.TrimSpace(fields[1]))
		if err != nil {
			thisTime, err = time.Parse("2006-01-02", strings.TrimSpace(fields[1]))
		}
		if err != nil {
			log.Printf("transactions line %d: datetime error (%s)\n", i, strings.TrimSpace(fields[1]))
			continue
		}
		if oldTime.After(thisTime) {
			log.Printf("transactions line %d: datetime not sorted\n", i)
		}
		if tr.ID == nil { // Fill tr only if it is not already filled
			// First field (used to be "id") is ignored
			tr.ID = ID(nextID)
			tr.Time = thisTime
			tr.Description = fields[2]
		} else {
			if oldTime != thisTime {
				log.Printf("NOTICE: transactions line %d: same transaction, different datetime\n", i)
			}
		}
		oldTime = thisTime
		accountID, err := strconv.Atoi(fields[4])
		if err != nil {
			log.Printf("transactions line %d: invalid account (%s)", i, fields[4])
			continue
		}
		sp := new(accounting.Split)
		sp.Account = c.accountMap[accountID]
		if sp.Account == nil {
			log.Printf("transactions line %d: invalid account (%s)", i, fields[4])
			continue
		}
		if thisTime != tr.Time {
			sp.Time = new(time.Time)
			*sp.Time = thisTime
		}
		if len(fields[5]) == 0 {
			if balance != 0 {
				log.Printf("transactions line %d: no value inside transaction (balance=%d)", i, balance)
				balance = 0
			}
			if len(fields[6]) == 0 {
				tr = nil
				continue
			}
			var sign int64 = 1
			offset := 0
			if fields[6][0] == '+' {
				sign = 1
				offset = 1
			} else if fields[6][0] == '-' {
				sign = -1
				offset = 1
			}
			f, err := strconv.ParseFloat(fields[6][offset:], 64)
			if err != nil {
				log.Printf("transactions line %d: invalid balance (%s)", i, fields[6])
				continue
			}
			var v accounting.Value
			v.Currency = &c.currency
			v.Amount = sign * int64(math.Round(100*f)) * 1000_000
			c.ledger.Assertions[sp] = v
		}
		if len(fields[5]) > 0 {
			var sign int64
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
			sp.Value.Currency = &c.currency
			sp.Value.Amount = sign * int64(math.Round(100*f)) * 1000_000
			balance += sp.Value.Amount
		}
		tr.Splits = append(tr.Splits, sp)
		sp.Account.Splits = append(sp.Account.Splits, sp)
		if balance == 0 {
			c.ledger.Transactions = append(c.ledger.Transactions, tr)
			tr = nil
			nextID++
		}
	}
	if balance != 0 {
		log.Printf("transactions: balance is %d, not zero", balance)
	}
	return nil
}

func init() {
	accounting.Register("txtdb", driver{})
}
