package main

import (
	"fmt"

	"github.com/cespedes/accounting"
	_ "github.com/cespedes/accounting/backend/psql"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func main() {
	connStr := "host=localhost user=katiuskas dbname=katiuskas password=veiThoh6ju"
	ledger, err := accounting.Open("psql", connStr)
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
	table.SetSelectable(true, false)
	table.SetCell(0, 0, tview.NewTableCell("[yellow]date").SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("[yellow]concept").SetSelectable(false).SetExpansion(1))
	table.SetCell(0, 2, tview.NewTableCell("[yellow]value").SetSelectable(false))
	table.SetCell(0, 3, tview.NewTableCell("[yellow]balance").SetSelectable(false))
	for i, t := range transactions {
		table.SetCell(i+1, 0, tview.NewTableCell(t.Time.Format("2006-01-02")))
		table.SetCell(i+1, 1, tview.NewTableCell(t.Description))
	}
	app.SetRoot(table, true)
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		cell := table.GetCell(1, 1)
		_, _, width, _ := table.GetRect()
		cell.SetText(fmt.Sprintf("width = %d", width))
		return false
	})
	if err := app.Run(); err != nil {
		panic(err)
	}
}
