package main

import (
	"fmt"
	"os"

	"github.com/cespedes/accounting"
	"github.com/cespedes/tableview"

	_ "github.com/cespedes/accounting/backend/postgres"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: tacc <database>")
		os.Exit(1)
	}
	ledger, err := accounting.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	// accounts := ledger.Accounts()
	transactions := ledger.Transactions()

	t := tableview.NewTableView()
	t.FillTable([]string{"date", "concept", "value", "balance"}, [][]string{})
	t.SetExpansion(1, 1)

	for i, tr := range transactions {
		t.SetCell(i, 0, tr.Time.Format("2006-01-02"))
		t.SetCell(i, 1, tr.Description)
		t.SetCell(i, 2, fmt.Sprintf("%d", len(tr.Splits)))
		t.SetCell(i, 3, "123456.78")
	}
	t.Run()
}
