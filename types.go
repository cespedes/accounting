package accounting

import "time"

// U is the amount by which every amount must be multiplied before storing it
const U = 100_000_000

// Currency represents a currency or commodity, and stores
// its name and how to display it with an amount.
//
// For more ideas on Currency, see github.com/leekchan/accounting
type Currency struct {
	Name        string // "EUR", "USD", etc
	PrintBefore bool   // "$1.00" vs "1.00$"
	PrintSpace  bool   // "1.00EUR" vs "1.00 EUR"
	Thousand    string // What to use (if any) every 3 digits
	Decimal     string // decimal separator ("." if empty)
	Precision   int    // Number of decimal places to show
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
	ID      int      // Used to identify this account.
	Parent  *Account // Optional
	Name    string   // Common name (ie, "Cash")
	Code    string   // Optional: for example, account number
	Comment string   // Optional
}

// Split is a deposit or withdrawal from an account.
type Split struct {
	Account *Account   // Origin or destination of funds
	Value   Value      // Amount to be transferred
	Balance Balance    // Balance of this account, after this movement
	Time    *time.Time // if nil, it inherits the transactions' time
	Comment string     // Split comment (if any)
}

// Price declares a market price, which is an exchange rate between
// two currencies on a certain date.
type Price struct {
	Currency *Currency
	Time     time.Time
	Value    Value
	Comment  string
}

// A Tag is a label which can be added to a transaction or movement.
type Tag struct {
	Name  string
	Value string
}

// Transaction stores an entry in the journal, consisting in a timestamp,
// a description and two or more money movements from different accounts.
type Transaction struct {
	ID          int       // Used to identify this transaction
	Time        time.Time // Date and time
	Description string    // Short description
	Comment     string    // Transaction comment (optional)
	Tags        []Tag     // Transaction tags (optional)
	Splits      []Split   // List of movements
}
