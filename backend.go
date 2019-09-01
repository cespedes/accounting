package accounting

type Backend interface{
	Open(name string) (Conn, error)
}

type Conn interface {
	Close()        error
	Accounts()     []Account
	Transactions() []Transaction
}
