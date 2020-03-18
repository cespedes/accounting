package accounting

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"sort"
	"sync"
	"time"
)

var (
	driversMu      sync.RWMutex
	drivers        = make(map[string]Driver)
	defaultSchemes = []string{"ledger", "txtdb", "postgres"}
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
	if backend == "" {
		for _, b := range defaultSchemes {
			if drivers[b] != nil {
				backend = b
				break
			}
		}
	}
	if drivers[backend] == nil {
		return nil, errors.New("accounting.Open: Backend " + backend + " is not registered.")
	}
	b := new(Backend)
	b.ready = true
	b.Ledger = new(Ledger)
	b.Ledger.connection, err = drivers[backend].Open(dataSource, b)
	if err != nil {
		return nil, err
	}

	return b.Ledger, nil
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

func (value Value) getString(full bool) string {
	var result string
	var c Currency

	if value.Currency != nil {
		c = *value.Currency
	}
	if c.PrintBefore {
		result += c.Name
		if !c.WithoutSpace {
			result += " "
		}
	}
	if value.Amount < 0 {
		result += "-"
		value.Amount = -value.Amount
	}
	i := value.Amount / 100_000_000
	d := value.Amount % 100_000_000
	if c.Decimal == "" { // shouldn't happen
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
		result += integer[start:end]
	}
	if c.Precision < 0 || c.Precision > 8 {
		panic(fmt.Sprintf("Money: invalid precision %d", c.Precision))
	}
	if c.Precision > 0 || (full && d > 0) {
		result += c.Decimal
		precision := c.Precision
		digits := fmt.Sprintf("%08d", d)
		if full {
			for i := 7; i >= precision; i-- {
				if digits[i] != '0' {
					precision = i + 1
					break
				}
			}
		}
		result += digits[:precision]
	}
	if !c.PrintBefore {
		if !c.WithoutSpace && c.Name != "" {
			result += " "
		}
		result += c.Name
	}

	return result
}

// String returns a string with the correct
// representation of that value, including its currency.
// The amount is represented with just the default digits in the currency definition.
func (value Value) String() string {
	return value.getString(false)
}

// FullString returns a string with the correct
// representation of that value, including its currency.
// The amount is represented with all the relevant digits.
func (value Value) FullString() string {
	return value.getString(true)
}

// String returns "0" for empty balances, or a list of its values separated by commas.
func (b Balance) String() string {
	if len(b) == 0 {
		return "0"
	}
	var s string
	for _, v := range b {
		if s != "" {
			s += ", "
		}
		s += v.String()
	}
	return s
}

// Close closes the ledger and prevents new queries from starting.
func (l *Ledger) Close() error {
	return l.connection.Close()
}

// Refresh loads again (if needed) all the accounting data.
func (l *Ledger) Refresh() {
	l.connection.Refresh()
}

// Account returns details for one account, given its ID.
func (l *Ledger) Account(id ID) *Account {
	x, ok := l.connection.(interface {
		Account(id ID) *Account
	})
	if ok {
		return x.Account(id)
	}
	for _, a := range l.Accounts {
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
func (l *Ledger) GetBalance(account *Account, when time.Time) Balance {
	if len(account.Splits) == 0 {
		return nil
	}
	if (when == time.Time{}) {
		return account.Splits[len(account.Splits)-1].Balance
	}
	for i := 1; i < len(account.Splits); i++ {
		if account.Splits[i].Time.After(when) {
			return account.Splits[i-1].Balance
		}
	}
	return account.Splits[len(account.Splits)-1].Balance
}

// TransactionsInAccount gets the list of all the transactions
// involving that account.
func (l *Ledger) TransactionsInAccount(account ID) []*Transaction {
	x, ok := l.connection.(interface {
		TransactionsInAccount(ID) []*Transaction
	})
	if ok {
		return x.TransactionsInAccount(account)
	}
	trans := make([]*Transaction, 0)
	for _, t := range l.Transactions {
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
func (l *Ledger) TransactionsInInterval(start, end time.Time) []*Transaction {
	x, ok := l.connection.(interface {
		TransactionsInInterval(time.Time, time.Time) []*Transaction
	})
	if ok {
		return x.TransactionsInInterval(start, end)
	}
	trans := make([]*Transaction, 0)
	for _, t := range l.Transactions {
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
	x, ok := l.connection.(interface {
		NewAccount(Account) (*Account, error)
	})
	if ok {
		return x.NewAccount(a)
	}
	return nil, errors.New("Ledger.NewAccount: not implemented")
}

// EditAccount edits an Account in a ledger
func (l *Ledger) EditAccount(a Account) (*Account, error) {
	x, ok := l.connection.(interface {
		EditAccount(Account) (*Account, error)
	})
	if ok {
		return x.EditAccount(a)
	}
	return nil, errors.New("Ledger.EditAccount: not implemented")
}

// NewTransaction adds a new Transaction in a ledger
func (l *Ledger) NewTransaction(t Transaction) (*Transaction, error) {
	x, ok := l.connection.(interface {
		NewTransaction(Transaction) (*Transaction, error)
	})
	if ok {
		return x.NewTransaction(t)
	}
	return nil, errors.New("Ledger.NewTransaction: not implemented")
}

// EditTransaction edits a Transaction in a ledger
func (l *Ledger) EditTransaction(t Transaction) (*Transaction, error) {
	x, ok := l.connection.(interface {
		EditTransaction(Transaction) (*Transaction, error)
	})
	if ok {
		return x.EditTransaction(t)
	}
	return nil, errors.New("Ledger.EditTransaction: not implemented")
}

// Flush writes all the pending changes to the backend.
func (l *Ledger) Flush() error {
	x, ok := l.connection.(interface {
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
func (l *Ledger) BalanceTransaction(transaction *Transaction) error {
	return l.balanceTransaction(transaction)
}

func (l *Ledger) balanceTransaction(transaction *Transaction) error {
	var unbalancedSplit *Split
	var balance Balance
	for i, s := range transaction.Splits {
		if s.Value.Currency == nil {
			if unbalancedSplit != nil {
				return fmt.Errorf("%s: more than one posting without amount", transaction.ID)
			}
			unbalancedSplit = transaction.Splits[i]
			continue
		}
		if v, ok := l.SplitPrices[s]; ok == true {
			balance.Add(v)
		} else {
			balance.Add(s.Value)
		}
	}
	for i := 0; i < len(balance); i++ {
		for i < len(balance) && balance[i].Amount == 0 {
			balance[i] = balance[len(balance)-1]
			balance = balance[:len(balance)-1]
		}
	}
	if len(balance) == 0 {
		// everything is balanced
		return nil
	}
	if unbalancedSplit != nil && len(balance) == 1 {
		unbalancedSplit.Value = balance[0]
		unbalancedSplit.Value.Amount = -unbalancedSplit.Value.Amount
		return nil
	}
	if unbalancedSplit != nil {
		return fmt.Errorf("%s: could not balance account %q: two or more currencies in transaction", transaction.ID, unbalancedSplit.Account.FullName())
	}
	if len(balance) == 1 {
		return fmt.Errorf("%s: could not balance transaction: total amount is %s", transaction.ID, balance[0])
	}
	if len(balance) == 2 {
		// we add 2 automatic prices, converting one currency to another and vice-versa
		price := new(Price)
		var i *big.Int
		price.Time = transaction.Time
		price.Currency = balance[0].Currency
		i = big.NewInt(-U)
		i.Mul(i, big.NewInt(balance[1].Amount))
		i.Quo(i, big.NewInt(balance[0].Amount))
		price.Value.Amount = i.Int64()
		price.Value.Currency = balance[1].Currency
		l.Prices = append(l.Prices, price)
		l.Comments[price] = append(l.Comments[price], "automatic")
		price = new(Price)
		price.Currency = balance[1].Currency
		i = big.NewInt(-U)
		i.Mul(i, big.NewInt(balance[0].Amount))
		i.Quo(i, big.NewInt(balance[1].Amount))
		price.Value.Amount = i.Int64()
		price.Value.Currency = balance[0].Currency
		l.Prices = append(l.Prices, price)
		l.Comments[price] = append(l.Comments[price], "automatic")
		return nil
	}
	if len(balance) > 2 {
		return fmt.Errorf("%s: not able to balance transactions with 3 or more currencies", transaction.ID)
	}
	panic("balanceTransaction(): unreachable code")
}

func (l *Ledger) GetCurrency(s string) *Currency {
	for i := range l.Currencies {
		if s == l.Currencies[i].Name {
			return l.Currencies[i]
		}
	}
	var currency Currency
	currency.Name = s
	l.Currencies = append(l.Currencies, &currency)
	return &currency
}

func (l *Ledger) Display(out io.Writer) {
	l.connection.Display(out)
}

// Add adds a value to a balance.
func (b *Balance) Add(v Value) {
	for i := range *b {
		if (*b)[i].Currency == v.Currency {
			(*b)[i].Amount += v.Amount
			return
		}
	}
	*b = append(*b, v)
}

// Dup duplicates a Balance.
func (b Balance) Dup() Balance {
	res := Balance{}
	for _, v := range b {
		res.Add(v)
	}
	return res
}
