package accounting

type Backend interface{
	Open(name string) (Conn, error)
}

type Conn interface {
	Accounts()     []Account
	Transactions() []Transaction
}
