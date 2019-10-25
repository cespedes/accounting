package accounting

import "time"

// Driver is the interface that must be implemented by the
// accounting backend.
type Driver interface {
	Open(name string) (Conn, error)
}

// Conn is a connection to an accounting backend.
type Conn interface {
	// Close flushes, if necessary, and closes the connection
	Close() error

	// Accounts returns the list of all the accounts in the ledger
	Accounts() []Account

	// Transaction returns all the transactions
	Transactions() []Transaction
}

// ConnExtra contains some extra methods that Conn could support.
// If it supports any ot these methods, the package will use them.
type ConnExtra interface {
	// GetBalance gets an account balance at a given time.
	// If passed the zero value, it gets the current balance.
	GetBalance(account int, t time.Time) int

	// TransactionsInAccount gets the list of all the transactions
	// involving that account.
	TransactionsInAccount(account int) []Transaction

	// TransactionsInInterval returns all the transactions between two times.
	TransactionsInInterval(start, end time.Time) []Transaction

	// NewAccount adds a new Account in a ledger.
	// ID field is ignored, and regenerated.
	NewAccount(a Account) (*Account, error)

	// EditAccount edits an Account in a ledger.
	// ID field must remain unchanged.
	EditAccount(a Account) (*Account, error)

	// NewTransaction adds a new Transaction in a ledger
	NewTransaction(t Transaction) (*Transaction, error)

	// EditTransaction edits a Transaction in a ledger
	EditTransaction(t Transaction) (*Transaction, error)

	// Flush writes all the pending changes to the backend.
	// If not implemented, we suppose it is not necessary
	// and return nil.
	Flush() error
}
