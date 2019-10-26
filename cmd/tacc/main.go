package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cespedes/accounting"
	"github.com/cespedes/tableview"

	_ "github.com/cespedes/accounting/backend/postgres"
	_ "github.com/cespedes/accounting/backend/txtdb"
)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: tacc <database>")
		os.Exit(1)
	}
	ledger, err := accounting.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	accounts := ledger.Accounts()
	transactions := ledger.Transactions()
	fmt.Printf("%d accounts, %d transactions\n", len(accounts), len(transactions))
	fmt.Println("* Accounts")
	for _, a := range accounts {
		fmt.Println("\t", a.ID, a.Name)
	}
	for _, t := range transactions {
		fmt.Println("\t", t.ID, t.Time, t.Description, len(t.Splits))
		for _, s := range t.Splits {
			fmt.Println("\t\t", s.Account.FullName(), s.Value, s.Balance)
		}
	}

	t := tableview.NewTableView()
	t.FillTable([]string{"id", "account", "balance"}, [][]string{})
	for i, ac := range accounts {
		t.SetCell(i, 0, strconv.Itoa(ac.ID))
		t.SetAlign(0, tableview.AlignRight)
		t.SetCell(i, 1, ac.FullName())
		t.SetExpansion(1, 1)
		t.SetAlign(2, tableview.AlignRight)
		balance := ledger.GetBalance(ac.ID, time.Time{})
		t.SetCell(i, 2, fmt.Sprintf("%d.%02d", balance/100, abs(balance%100)))
	}
	t.SetSelectedFunc(func(row int) {
		fmt.Println(row)
		time.Sleep(3 * time.Second)
	})
	t.Run()
	/*
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
	*/
}
