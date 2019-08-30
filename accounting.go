/*
Package accounting implements a double-entry accounting system.

It can use text (ledger format) or PostgreSQL back-ends
*/
package accounting

import (
	"errors"
	"sync"
	"time"
	"github.com/cespedes/accounting/backend"
)

// Currency stores the representation of a currency,
// with its name and the number of decimal positions (if any)
type Currency struct {
	Name    string // "EUR", "USD", etc
	Decimal int    // Number of significant decimal places
}

/*
For more ideas on Currency, see github.com/leekchan/accounting
*/

// Account specifies one origin or destination of funds
type Account struct {
	Id       int      // Used to identify this account
	Parent   *Account // Optional
	Name     string   // Common name (ie, "Cash")
	Code     string   // Optional: for example, account number
	Balance  int      // Final balance of account
	Currency Currency // 
}

// Split is a deposit or withdrawal from an account
type Split struct {
	Account *Account // Origin or destination of funds
	Value   int      // Amount to be transferred
	Balance int      // Account balance after this transfer
}

// Transaction stores on entry in the journal, consisting in one description
// and two or more money movements from different accounts
type Transaction struct {
	Id          int       // Used to identify this transaction
	Time        time.Time // Date and time
	Description string    // Short description
	Splits      []Split   // List of movements
}

// Ledger stores all the accounts and transactions in one accounting
type Ledger struct {
	Accounts     []Transaction
	Transactions []Transaction
}

// Open opens a ledger specified by its backend name and a backend-specific
// data source name, usually consisting on a file name or a database name
func Open(backend, dataSource string) (*Ledger, error) {
	return nil, errors.New("Not implemented")
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]backend.Driver)
)

// Register makes an accounting backend available by the provided name.
// If Register is called twice with the same name or if driver is nil, it panics.
func Register(name string, driver backend.Driver) {
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

// Close closes the ledger and prevents new queries from starting.
func (l *Ledger) Close() error {
	return errors.New("Not implemented")
}

// Transaction returns all the transactions concerning that account
func (l *Ledger) AccountTransactions(a *Account) []Transaction {
	return nil
}

// NewAccount adds a new Account in a ledger
func (l *Ledger) NewAccount(a Account) error {
	return errors.New("Not implemented")
}

// NewTransaction adds a new Transaction in a ledger
func (l *Ledger) NewTransaction(t Transaction) error {
	return errors.New("Not implemented")
}
