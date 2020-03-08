package accounting

import (
	"io"
	"time"
)

// Driver is the interface that must be implemented by the
// accounting backend.
type Driver interface {
	Open(url string, ledger *Ledger, backend *BackendLedger) (Connection, error)
}

// Connection is a connection to an accounting backend.
type Connection interface {
	// Close flushes, if necessary, and closes the connection.
	Close() error

	// Refresh loads again (if needed) all the accounting data.
	Refresh()

	// Show the ledger data.
	Display(out io.Writer)
}

// BackendLedger contains some methods to be called only by the backends.
type BackendLedger struct {
	ledger *Ledger
}

// NewTransaction adds a new transaction to the ledger, updating
// the ledger's Accounts and Transactions fields.
// It also runs some sanity checks.
func (b *BackendLedger) NewTransaction(t *Transaction) {
	// TODO: only chronologically sorted transactions
	//       and splits are supported right now.
	b.ledger.balanceTransaction(t)
	b.ledger.Transactions = append(b.ledger.Transactions, t)
	for _, s := range t.Splits {
		s.Balance = make(Balance)
		if len(s.Account.Splits) > 0 {
			s.Balance = s.Account.Splits[len(s.Account.Splits)-1].Balance
		}
		s.Balance[s.Value.Currency] += s.Value.Amount
		s.Account.Splits = append(s.Account.Splits, s)
	}
}

// ConnExtra contains some extra methods that Conn could support.
// If it supports any ot these methods, the package will use them.
// If they are not available, it will fall back to another approach,
// or fail if it is not possible.
type ConnExtra interface {
	// Account returns the account with the specified id, if present
	Account(id int) *Account

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
