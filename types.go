package accounting

import "time"

// U is the number by which every amount must be multiplied before storing it.
const U = 100_000_000

// Ledger stores all the accounts and transactions in one accounting.
type Ledger struct {
	connection      Connection
	Accounts        []*Account
	Transactions    []*Transaction           // sorted by Time.
	Currencies      []*Currency              // can be empty.
	Prices          []*Price                 // can be empty; sorted by Time.
	Comments        map[interface{}][]string // Comments in Accounts, Transactions, Currencies or Prices.
	Assertions      map[*Split]Value         // Value that should be in an account after one split.
	SplitPrices     map[*Split]Value         // Price for the value in a split, in another currency.
	DefaultCurrency *Currency                // Default currency.
	// Tags            map[interface{}][]Tag
	// TagsByName      map[string][]struct {Value string; Place interface{}}
}

// ID is used to identify one currency, account, transaction, split or price.
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
	ISIN         string // International Securities Identification Number
}

// Value specifies an amount and its currency
type Value struct {
	Amount   int64     // Amount (actual value times U)
	Currency *Currency // Currency or commodity
}

// Balance is a list of currencies and amounts.
type Balance []Value

// Account specifies one origin or destination of funds.
type Account struct {
	ID           ID         // used to identify this account.
	Parent       *Account   // Optional
	Children     []*Account // Automatically filled.
	Level        int        // Number of ancestors does this Account have. Automatically filled.
	Name         string     // Common (short) name (ie, "Cash")
	Code         string     // Optional. For example, account number
	Splits       []*Split   // List of movements in this account
	StartBalance Balance    // Balance at the start of current period (zero if no start date was specified)
}

// TransferAccount is a special account used when a transaction has two or more splits with different times.
// Ledger.Fill() automatically generates splits with this account.
var TransferAccount Account = Account{
	Name: "Assets:Transfer account",
}

// Transaction stores an entry in the journal, consisting in a timestamp,
// a description and two or more money movements from different accounts.
type Transaction struct {
	ID          ID        // used to identify this transaction.
	Time        time.Time // Date and time
	Description string    // Short description
	Splits      []*Split  // List of movements
}

// Split is a deposit or withdrawal from an account.
type Split struct {
	ID          ID           // used to identify this split.
	Account     *Account     // Origin or destination of funds.
	Transaction *Transaction // Transaction this split belongs to.
	Time        *time.Time   // In most cases, this is equal to Transaction.Time
	Value       Value        // Amount to be transferred.
	Balance     Balance      // Balance of this account, after this movement.
}

// Price declares a market price, which is an exchange rate between
// two currencies on a certain date.
type Price struct {
	ID       ID // used to identify this price.
	Time     time.Time
	Currency *Currency
	Value    Value
}

// A Tag is a label which can be added to a transaction or movement.
type Tag struct {
	Name  string
	Value string
}
