package main

import (
	"fmt"
	"os"

	"github.com/cespedes/accounting"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

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
	fmt.Printf("ledger = %#v\n", ledger)

	// accounts := ledger.Accounts()
	transactions := ledger.Transactions()

	app := tview.NewApplication()
	table := tview.NewTable()
	table.SetBorder(false)
	table.SetTitle(" Accounting ")
	table.SetSeparator(tview.Borders.Vertical)
	table.SetFixed(1, 0)
	table.SetFixedColumnsWidth(true)
	table.SetSelectable(true, false)
	table.SetCell(0, 0, tview.NewTableCell("[yellow]date").
		SetSelectable(false).
		SetAlign(tview.AlignCenter).
		SetMaxWidth(10))
	table.SetCell(0, 1, tview.NewTableCell("[yellow]concept").
		SetSelectable(false).
		SetAlign(tview.AlignCenter).
		SetExpansion(1))
	table.SetCell(0, 2, tview.NewTableCell("[yellow]value").
		SetSelectable(false).
		SetAlign(tview.AlignCenter).
		SetMaxWidth(10))
	table.SetCell(0, 3, tview.NewTableCell("[yellow]balance").
		SetSelectable(false).
		SetAlign(tview.AlignCenter).
		SetMaxWidth(11))
	for i, t := range transactions {
		table.SetCell(i+1, 0, tview.NewTableCell(t.Time.Format("2006-01-02")))
		table.SetCell(i+1, 1, tview.NewTableCell(t.Description))
		table.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%d.%02d", 10*i, i%100)).SetAlign(tview.AlignRight))
		table.SetCell(i+1, 3, tview.NewTableCell("123456.78"))
	}
	app.SetRoot(table, true)
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		_, _, width, _ := table.GetRect()
		for i := 1; i < table.GetRowCount(); i++ {
			table.GetCell(i, 1).SetMaxWidth(width - 33)
		}
		return false
	})
	if err := app.Run(); err != nil {
		panic(err)
	}
}
