package accounting

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"
)

// Currency stores the representation of a currency,
// with its name and the number of decimal positions (if any)
//
// For more ideas on Currency, see github.com/leekchan/accounting
type Currency struct {
	Name    string // "EUR", "USD", etc
	Decimal int    // Number of significant decimal places
}

// Account specifies one origin or destination of funds
type Account struct {
	ID       int      // Used to identify this account.
	Parent   *Account // Optional
	Name     string   // Common name (ie, "Cash")
	Code     string   // Optional: for example, account number
	Currency Currency //
}

// Split is a deposit or withdrawal from an account
type Split struct {
	Account *Account // Origin or destination of funds
	Value   int      // Amount to be transferred
	Balance int      // Account balance after this transfer
}

// Transaction stores an entry in the journal, consisting in a timestamp,
// a description and two or more money movements from different accounts.
type Transaction struct {
	ID          int       // Used to identify this transaction
	Time        time.Time // Date and time
	Description string    // Short description
	Splits      []Split   // List of movements
}

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

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (l *Ledger) Money(value int) string {
	// TODO: support different currency formats
	return fmt.Sprintf("%d.%02d", value/100, abs(value%100))
}

// Close closes the ledger and prevents new queries from starting.
func (l *Ledger) Close() error {
	return l.driver.Close()
}

// Accounts returns the list of all the accounts
func (l *Ledger) Accounts() []*Account {
	return l.driver.Accounts()
}

// Transactions returns all the transactions
func (l *Ledger) Transactions() []Transaction {
	return l.driver.Transactions()
}

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

func (a Account) FullName() string {
	if a.Parent == nil {
		return a.Name
	}
	return a.Parent.FullName() + ":" + a.Name
}

// GetBalance gets an account balance at a given time.
// If passed the zero value, it gets the current balance.
func (l *Ledger) GetBalance(account int, when time.Time) int {
	x, ok := l.driver.(interface {
		GetBalance(int, time.Time) int
	})
	if ok {
		return x.GetBalance(account, when)
	}
	balance := 0
	for _, t := range l.TransactionsInAccount(account) {
		if (when != time.Time{}) && t.Time.After(when) {
			continue
		}

		for _, s := range t.Splits {
			if s.Account.ID == account {
				balance += s.Value
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
			//			log.Printf("s.Account.ID=%d account=%d", s.Account.ID, account)
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
