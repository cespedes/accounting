package main

import (
	"fmt"
	"os"

	"github.com/cespedes/accounting"
	_ "github.com/cespedes/accounting/backend/ledger"
	_ "github.com/cespedes/accounting/backend/postgres"
	_ "github.com/cespedes/accounting/backend/txtdb"

	"github.com/davecgh/go-spew/spew"
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

	// ledger.Display(os.Stdout, L)
	spew.Dump(L)
	//	for _, a := range L.Accounts {
	//		fmt.Printf("%#v\n", a)
	//	}
}
