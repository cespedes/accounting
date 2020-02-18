package accounting

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Ledger stores all the accounts and transactions in one accounting
type Ledger struct {
	driver Conn
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

// Open opens a ledger specified by a URL-like string, where the scheme is the
// backend name and the rest of the URL is backend-specific (usually consisting
// on a file name or a database name).
func Open(dataSource string) (*Ledger, error) {
	url, err := url.Parse(dataSource)
	if err != nil {
		return nil, fmt.Errorf("accounting.Open: %v", err)
	}
	backend := url.Scheme
	driversMu.RLock()
	defer driversMu.RUnlock()
	if drivers[backend] == nil {
		return nil, errors.New("accounting.Open: Backend " + backend + " is not registered.")
	}
	conn, err := drivers[backend].Open(dataSource)
	if err != nil {
		return nil, err
	}
	l := new(Ledger)
	l.driver = conn

	return l, nil
}

// Register makes an accounting backend available by the provided name.
// If Register is called twice with the same name or if driver is nil, it panics.
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("accounting: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("accounting: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// String returns a string with the correct
// representation of that value, including its currency.
func (value Value) String() string {
	var result string
	var c Currency

	if value.Currency != nil {
		c = *value.Currency
	}
	if c.PrintBefore {
		result += c.Name
		if c.PrintSpace {
			result += " "
		}
	}
	if value.Amount < 0 {
		result += "-"
		value.Amount = -value.Amount
	}
	i := value.Amount / 100_000_000
	d := value.Amount % 100_000_000
	if c.Decimal == "" {
		c.Decimal = "."
	}
	integer := fmt.Sprintf("%d", i)
	for n, l := 0, len(integer); n < 1+(l-1)/3; n++ {
		if n > 0 {
			result += c.Thousand
		}
		end := 3*n + (l-1)%3 + 1
		start := end - 3
		if start < 0 {
			start = 0
		}
		// result += fmt.Sprintf("[%d,%d]", start, end)
		result += integer[start:end]
	}
	if c.Precision < 0 || c.Precision > 8 {
		panic(fmt.Sprintf("Money: invalid precision %d", c.Precision))
	}
	if c.Precision > 0 {
		result += c.Decimal
		result += fmt.Sprintf("%08d", d)[:c.Precision]
	}
	if !c.PrintBefore {
		if c.PrintSpace {
			result += " "
		}
		result += c.Name
	}

	return result
}

// String returns "0" for empty balances, or a list of its values
func (b Balance) String() string {
	if len(b) == 0 {
		return "0"
	}
	var s string
	for c, a := range b {
		var v = Value{Amount: a, Currency: c}
		s = s + v.String()
	}
	return s
}

// Close closes the ledger and prevents new queries from starting.
func (l *Ledger) Close() error {
	return l.driver.Close()
}

// Accounts returns the list of all the accounts.
func (l *Ledger) Accounts() []*Account {
	return l.driver.Accounts()
}

// Transactions returns all the transactions.
func (l *Ledger) Transactions() []Transaction {
	return l.driver.Transactions()
}

// Account returns details for one account, given its id.
func (l *Ledger) Account(id int) *Account {
	x, ok := l.driver.(interface {
		Account(int) *Account
	})
	if ok {
		return x.Account(id)
	}
	for _, a := range l.Accounts() {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// FullName returns the fully qualified name of the account:
// the name of all its ancestors, separated by ":", and ending
// with this account's name.
func (a Account) FullName() string {
	if a.Parent == nil {
		return a.Name
	}
	return a.Parent.FullName() + ":" + a.Name
}

// GetBalance gets an account balance at a given time.
// If passed the zero value, it gets the current balance.
func (l *Ledger) GetBalance(account int, when time.Time) Balance {
	x, ok := l.driver.(interface {
		GetBalance(int, time.Time) Balance
	})
	if ok {
		return x.GetBalance(account, when)
	}
	balance := make(Balance)
	for _, t := range l.TransactionsInAccount(account) {
		if (when != time.Time{}) && t.Time.After(when) {
			continue
		}

		for _, s := range t.Splits {
			if s.Account.ID == account {
				balance[s.Value.Currency] += s.Value.Amount
			}
		}
	}
	return balance
}

// TransactionsInAccount gets the list of all the transactions
// involving that account.
func (l *Ledger) TransactionsInAccount(account int) []Transaction {
	x, ok := l.driver.(interface {
		TransactionsInAccount(int) []Transaction
	})
	if ok {
		return x.TransactionsInAccount(account)
	}
	trans := make([]Transaction, 0)
	for _, t := range l.Transactions() {
		for _, s := range t.Splits {
			// log.Printf("s.Account.ID=%d account=%d", s.Account.ID, account)
			if s.Account.ID == account {
				trans = append(trans, t)
				break
			}
		}
	}
	// log.Printf("Ledger.TransactionsInAccount(%d): %d trans", account, len(trans))
	return trans
}

// TransactionsInInterval returns all the transactions between two times.
func (l *Ledger) TransactionsInInterval(start, end time.Time) []Transaction {
	x, ok := l.driver.(interface {
		TransactionsInInterval(time.Time, time.Time) []Transaction
	})
	if ok {
		return x.TransactionsInInterval(start, end)
	}
	trans := make([]Transaction, 0)
	for _, t := range l.Transactions() {
		if start.After(t.Time) {
			continue
		}
		if end.Before(t.Time) {
			continue
		}
		trans = append(trans, t)
	}
	return trans
}

// NewAccount adds a new Account in a ledger
func (l *Ledger) NewAccount(a Account) (*Account, error) {
	x, ok := l.driver.(interface {
		NewAccount(Account) (*Account, error)
	})
	if ok {
		return x.NewAccount(a)
	}
	return nil, errors.New("Ledger.NewAccount: not implemented")
}

// EditAccount edits an Account in a ledger
func (l *Ledger) EditAccount(a Account) (*Account, error) {
	x, ok := l.driver.(interface {
		EditAccount(Account) (*Account, error)
	})
	if ok {
		return x.EditAccount(a)
	}
	return nil, errors.New("Ledger.EditAccount: not implemented")
}

// NewTransaction adds a new Transaction in a ledger
func (l *Ledger) NewTransaction(t Transaction) (*Transaction, error) {
	x, ok := l.driver.(interface {
		NewTransaction(Transaction) (*Transaction, error)
	})
	if ok {
		return x.NewTransaction(t)
	}
	return nil, errors.New("Ledger.NewTransaction: not implemented")
}

// EditTransaction edits a Transaction in a ledger
func (l *Ledger) EditTransaction(t Transaction) (*Transaction, error) {
	x, ok := l.driver.(interface {
		EditTransaction(Transaction) (*Transaction, error)
	})
	if ok {
		return x.EditTransaction(t)
	}
	return nil, errors.New("Ledger.EditTransaction: not implemented")
}

// Flush writes all the pending changes to the backend.
func (l *Ledger) Flush() error {
	x, ok := l.driver.(interface {
		Flush() error
	})
	if ok {
		return x.Flush()
	}
	// If not implemented by the backend, we suppose it is not needed
	// and return nil.
	return nil
}

// SortAccounts returns a properly sorted copy of a slice of accounts.
// Input parameter "accounts" may be modified by this function.
func SortAccounts(accounts []*Account) []*Account {
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].FullName() < accounts[j].FullName()
	})
	return accounts
}
