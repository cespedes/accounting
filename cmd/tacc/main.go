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

func tableAccounts(l *accounting.Ledger) {
	accounts := l.Accounts()

	t := tableview.NewTableView()
	t.FillTable([]string{"id", "account", "balance"}, [][]string{})
	t.SetExpansion(1, 1)
	for i, ac := range accounts {
		t.SetCell(i, 0, strconv.Itoa(ac.ID))
		t.SetAlign(0, tableview.AlignRight)
		t.SetCell(i, 1, ac.FullName())
		t.SetAlign(2, tableview.AlignRight)
		var balance accounting.Value
		for _, a := range (l.GetBalance(ac.ID, time.Time{})) {
			balance.Amount += a
		}
		t.SetCell(i, 2, l.Money(balance))
	}
	t.SetSelectedFunc(func(row int) {
		tableTransactions(l, accounts[row-1].ID)
	})
	t.Run()
}

func tableTransactions(l *accounting.Ledger, acc int) {
	transactions := l.TransactionsInAccount(acc)
	fmt.Printf("account %d: %d transactions\n", acc, len(transactions))
	t := tableview.NewTableView()
	t.FillTable([]string{"date", "description", "value", "balance"}, [][]string{})
	t.SetExpansion(1, 1)
	for i, tr := range transactions {
		var sp accounting.Split
		for _, sp = range tr.Splits {
			if sp.Account.ID == acc {
				break
			}
		}
		t.SetCell(i, 0, tr.Time.Format("02-01-2006"))
		t.SetCell(i, 1, tr.Description)
		t.SetCell(i, 2, l.Money(sp.Value))
		t.SetAlign(2, tableview.AlignRight)
		t.SetCell(i, 3, l.Money(sp.Balance))
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
