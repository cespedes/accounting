package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cespedes/accounting"
	"github.com/cespedes/tableview"

	_ "github.com/cespedes/accounting/backend/ledger"
	_ "github.com/cespedes/accounting/backend/postgres"
	_ "github.com/cespedes/accounting/backend/txtdb"
)

func tableAccounts(l *accounting.Ledger) {
	t := tableview.NewTableView()
	t.FillTable([]string{"account", "balance"}, [][]string{})
	t.SetExpansion(0, 1)
	for i, ac := range l.Accounts {
		// t.SetCell(i, 0, strconv.Itoa(ac.ID))
		t.SetCell(i, 0, ac.FullName())
		t.SetAlign(1, tableview.AlignRight)
		t.SetCell(i, 1, l.GetBalance(ac, time.Time{}).String())
	}
	t.SetSelectedFunc(func(row int) {
		tableTransactions(l, l.Accounts[row-1].ID)
	})
	t.Run()
}

func tableTransactions(l *accounting.Ledger, accID accounting.ID) {
	account := l.Account(accID)
	fmt.Printf("account %d: %d splits\n", accID, len(account.Splits))
	t := tableview.NewTableView()
	t.FillTable([]string{"date", "description", "value", "balance"}, [][]string{})
	t.SetExpansion(1, 1)
	for i, sp := range account.Splits {
		t.SetCell(i, 0, sp.Time.Format("02-01-2006"))
		t.SetCell(i, 1, sp.Transaction.Description)
		t.SetCell(i, 2, sp.Value.String())
		t.SetAlign(2, tableview.AlignRight)
		t.SetCell(i, 3, sp.Balance.String())
		t.SetAlign(3, tableview.AlignRight)
	}
	t.Run()
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

	tableAccounts(ledger)
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
