package accounting

import "time"

// U is the number by which every amount must be multiplied before storing it.
const U = 100_000_000

// Ledger stores all the accounts and transactions in one accounting.
type Ledger struct {
	connection   Connection
	Accounts     []*Account
	Transactions []*Transaction
	Currencies   []*Currency
	Prices       []Price
}

// ID is used to identify one currency, account, transaction or price.
type ID interface {
	String() string
}

// Currency represents a currency or commodity, and stores
// its name and how to display it with an amount.
//
// For more ideas on Currency, see github.com/leekchan/accounting
type Currency struct {
	ID           ID     // used to identify this currency
	Name         string // "EUR", "USD", etc
	PrintBefore  bool   // "$1.00" vs "1.00$"
	WithoutSpace bool   // "1.00EUR" vs "1.00 EUR"
	Thousand     string // What to use (if any) every 3 digits
	Decimal      string // decimal separator ("." if empty)
	Precision    int    // Number of decimal places to show
	Comments     []string
}

// Value specifies an amount and its currency
type Value struct {
	Amount   int64     // Amount (actual value times 10^8)
	Currency *Currency // Currency or commodity
}

// Balance is a list of currencies and amounts.
type Balance map[*Currency]int64

// Account specifies one origin or destination of funds.
type Account struct {
	ID       ID       // used to identify this account.
	Parent   *Account // Optional
	Name     string   // Common (short) name (ie, "Cash")
	Code     string   // Optional. For example, account number
	Comments []string // Optional
	Splits   []*Split // List of movements in this account
}

// Split is a deposit or withdrawal from an account.
type Split struct {
	ID          ID           // used to identify this split.
	Account     *Account     // Origin or destination of funds.
	Transaction *Transaction // Transaction this split belongs to.
	Value       Value        // Amount to be transferred.
	EqValue     *Value       // Price of this value, in another currency.
	Balance     Balance      // Balance of this account, after this movement.
	Assertion   Balance      // What part of the balance should be.
	Time        *time.Time   // In most cases, this is equal to Transaction.Time
	Comments    []string     // Split comments (if any)
}

// Price declares a market price, which is an exchange rate between
// two currencies on a certain date.
type Price struct {
	ID       ID // used to identify this price.
	Time     time.Time
	Currency *Currency
	Value    Value
	Comments []string
}

// A Tag is a label which can be added to a transaction or movement.
type Tag struct {
	Name  string
	Value string
}

// Transaction stores an entry in the journal, consisting in a timestamp,
// a description and two or more money movements from different accounts.
type Transaction struct {
	ID          ID        // used to identify this transaction.
	Time        time.Time // Date and time
	Description string    // Short description
	Comments    []string  // Transaction comment (optional)
	Tags        []Tag     // Transaction tags (optional)
	Splits      []*Split  // List of movements
}
