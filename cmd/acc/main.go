package main

import (
	"fmt"
	"os"

	"github.com/cespedes/accounting"
	"github.com/cespedes/accounting/backend/ledger"
	_ "github.com/cespedes/accounting/backend/postgres"
	_ "github.com/cespedes/accounting/backend/txtdb"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: tacc <database>")
		os.Exit(1)
	}
	L, err := accounting.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	ledger.Export(os.Stdout, L)
	//	for _, a := range L.Accounts {
	//		fmt.Printf("%#v\n", a)
	//	}
}
