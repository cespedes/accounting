package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cespedes/accounting"
	"github.com/cespedes/accounting/backend/ledger"
	"github.com/cespedes/tableview"
)

type flags struct {
	batch     bool
	market    bool
	negate    bool
	pivot     sliceString
	beginDate time.Time
	endDate   time.Time
}

var commands = map[string]func(ledger *accounting.Ledger, flags flags, args []string) error{
	"accounts":        runAccounts,
	"a":               runAccounts,
	"balance":         runBalance,
	"bal":             runBalance,
	"b":               runBalance,
	"stats":           runStats,
	"print":           runPrint,
	"incomestatement": runIncomeStatement,
	"is":              runIncomeStatement,
	"delta":           runDelta,
	"price":           runPrice,
}

func runAccounts(L *accounting.Ledger, flags flags, args []string) error {
	var treeFlag bool
	f := flag.NewFlagSet("accounts", flag.ExitOnError)
	f.BoolVar(&treeFlag, "tree", false, "show short account names, as a tree")
	f.Parse(args)

	for _, a := range L.Accounts {
		if treeFlag {
			fmt.Printf("%*.0s%s\n", 2*a.Level, " ", a.FullName())
		} else {
			fmt.Println(a.FullName())
		}
	}
	return nil
}

type account struct {
	Name    string
	Level   int
	Account *accounting.Account
}

func insertAccount(where *[]account, name string, level int, a *accounting.Account) {
	for _, b := range *where {
		if b.Account == a {
			return
		}
	}
	*where = append(*where, account{
		Name:    name,
		Level:   level,
		Account: a,
	})
	for _, b := range a.Children {
		insertAccount(where, b.Name, level+1, b)
	}
}

func runBalance(L *accounting.Ledger, flags flags, args []string) error {
	var maxLength int
	var total accounting.Balance
	var accounts []account
	if len(args) == 0 {
		for _, a := range L.Accounts {
			accounts = append(accounts, account{Name: a.Name, Level: a.Level, Account: a})
		}
	} else {
		for _, a := range L.Accounts {
			for _, b := range args {
				if strings.Contains(strings.ToLower(a.FullName()), strings.ToLower(b)) {
					insertAccount(&accounts, a.FullName(), 0, a)
					break
				}
			}
		}
	}
	for _, a := range accounts {
		thisBal := a.Account.StartBalance
		if len(a.Account.Splits) > 0 {
			if flags.market {
				for _, v := range a.Account.Splits[len(a.Account.Splits)-1].Balance {
					thisBal.Add(L.Convert(v, flags.endDate, L.DefaultCurrency))
				}
				a.Account.Splits[len(a.Account.Splits)-1].Balance = thisBal
			}
			thisBal = a.Account.Splits[len(a.Account.Splits)-1].Balance
		}
		for _, v := range thisBal {
			length := len(v.String())
			if length > maxLength {
				maxLength = length
			}
			total.Add(v)
		}
	}
	for _, v := range total {
		length := len(v.String())
		if length > maxLength {
			maxLength = length
		}
	}
	for _, a := range accounts {
		if len(a.Account.Splits) > 0 {
			for i, v := range a.Account.Splits[len(a.Account.Splits)-1].Balance {
				fmt.Printf("%*.*s", maxLength, maxLength, v.String())
				if i == len(a.Account.Splits[len(a.Account.Splits)-1].Balance)-1 {
					fmt.Printf(" %*.0s%s\n", 2*a.Level, " ", a.Name)
				} else {
					fmt.Println()
				}
			}
		} else {
			fmt.Printf("%*.0s%s\n", maxLength+1+2*a.Level, " ", a.Name)
		}
	}
	fmt.Println(strings.Repeat("-", maxLength))
	for _, v := range total {
		fmt.Printf("%*.*s\n", maxLength, maxLength, v.String())
	}
	return nil
}

func runStats(L *accounting.Ledger, flags flags, args []string) error {
	if len(L.Transactions) == 0 {
		fmt.Println("No transactions in ledger")
	} else {
		first := L.Transactions[0].Time
		last := L.Transactions[len(L.Transactions)-1].Time
		firstYear, firstMonth, firstDay := first.Date()
		lastYear, lastMonth, lastDay := last.Date()
		end := time.Date(lastYear, lastMonth, lastDay, 0, 0, 0, 0, time.UTC)
		start := time.Date(firstYear, firstMonth, firstDay, 0, 0, 0, 0, time.UTC)
		days := int(end.Sub(start).Hours()/24.0) + 1

		fmt.Printf("Transaction span : %s to %s (%d days)\n", first.Format("2006-01-02"),
			last.Format("2006-01-02"), days)
		fmt.Printf("Transactions     : %d (%.1f per day)\n", len(L.Transactions), float64(len(L.Transactions))/float64(days))
		fmt.Printf("Accounts         : %d\n", len(L.Accounts))
		fmt.Printf("Commodities      : %d (", len(L.Currencies))
		for i, c := range L.Currencies {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(c.Name)
		}
		fmt.Println(")")
		fmt.Printf("Market prices    : %d\n", len(L.Prices))
	}
	return nil
}

func runPrint(L *accounting.Ledger, flags flags, args []string) error {
	ledger.Export(os.Stdout, L)
	return nil
}

func runIncomeStatement(L *accounting.Ledger, flags flags, args []string) error {
	var incomeAccounts, expenseAccounts []*accounting.Account
	var incomes, expenses []struct {
		name    string
		balance string
	}
	var income, expense, net accounting.Balance
	var nameLen = 8
	var balanceLen = 1

	if len(args) == 0 {
		for _, a := range L.Accounts {
			if strings.HasPrefix(a.FullName(), "Income:") {
				incomeAccounts = append(incomeAccounts, a)
			}
			if strings.HasPrefix(a.FullName(), "Expense:") {
				expenseAccounts = append(expenseAccounts, a)
			}
		}
	} else {
		for _, a := range L.Accounts {
			if !strings.HasPrefix(a.FullName(), "Income") {
				continue
			}
			for _, b := range args {
				if strings.Contains(strings.ToLower(a.FullName()), strings.ToLower(b)) {
					incomeAccounts = append(incomeAccounts, a)
					break
				}
			}
		}
		for _, a := range L.Accounts {
			if !strings.HasPrefix(a.FullName(), "Expense") {
				continue
			}
			for _, b := range args {
				if strings.Contains(strings.ToLower(a.FullName()), strings.ToLower(b)) {
					expenseAccounts = append(expenseAccounts, a)
					break
				}
			}
		}
	}

	for _, a := range incomeAccounts {
		if len(a.Splits) > 0 {
			b := a.Splits[0].Balance.Dup()
			b.SubBalance(a.Splits[len(a.Splits)-1].Balance)
			b.Sub(a.Splits[0].Value)
			incomes = append(incomes, struct {
				name    string
				balance string
			}{a.FullName(), b.String()})
			income.AddBalance(b)
		}
	}
	for _, a := range expenseAccounts {
		if len(a.Splits) > 0 {
			b := a.Splits[len(a.Splits)-1].Balance.Dup()
			b.SubBalance(a.Splits[0].Balance)
			b.Add(a.Splits[0].Value)
			expenses = append(expenses, struct {
				name    string
				balance string
			}{a.FullName(), b.String()})
			expense.AddBalance(b)
		}
	}
	net = income.Dup()
	net.SubBalance(expense)
	for _, i := range incomes {
		if len(i.name) > nameLen {
			nameLen = len(i.name)
		}
		if len(i.balance) > balanceLen {
			balanceLen = len(i.balance)
		}
	}
	for _, i := range expenses {
		if len(i.name) > nameLen {
			nameLen = len(i.name)
		}
		if len(i.balance) > balanceLen {
			balanceLen = len(i.balance)
		}
	}
	if flags.batch {
		fmt.Println(net)
		return nil
	}
	fmt.Println("Income Statement")
	fmt.Println()
	fmt.Print(strings.Repeat("=", nameLen+2), "++", strings.Repeat("=", balanceLen+2), "\n")
	fmt.Printf(" %-*s ||\n", nameLen, "Revenues")
	fmt.Print(strings.Repeat("-", nameLen+2), "++", strings.Repeat("-", balanceLen+2), "\n")
	for _, i := range incomes {
		fmt.Printf(" %-*s || %*s\n", nameLen, i.name, balanceLen, i.balance)
	}
	fmt.Print(strings.Repeat("-", nameLen+2), "++", strings.Repeat("-", balanceLen+2), "\n")
	fmt.Printf(" %s || %*s\n", strings.Repeat(" ", nameLen), balanceLen, income)
	fmt.Print(strings.Repeat("=", nameLen+2), "++", strings.Repeat("=", balanceLen+2), "\n")
	fmt.Printf(" %-*s ||\n", nameLen, "Expenses")
	fmt.Print(strings.Repeat("-", nameLen+2), "++", strings.Repeat("-", balanceLen+2), "\n")
	for _, e := range expenses {
		fmt.Printf(" %-*s || %*s\n", nameLen, e.name, balanceLen, e.balance)
	}
	fmt.Print(strings.Repeat("-", nameLen+2), "++", strings.Repeat("-", balanceLen+2), "\n")
	fmt.Printf(" %s || %*s\n", strings.Repeat(" ", nameLen), balanceLen, expense)
	fmt.Print(strings.Repeat("=", nameLen+2), "++", strings.Repeat("=", balanceLen+2), "\n")
	fmt.Printf(" %-*s || %*s\n", nameLen, "Net:", balanceLen, net)
	return nil
}

func runDelta(L *accounting.Ledger, flags flags, args []string) error {
	var accounts []*accounting.Account
	if len(args) == 0 {
		return nil
	}
	for _, a := range L.Accounts {
		for _, b := range args {
			if strings.Contains(strings.ToLower(a.FullName()), strings.ToLower(b)) {
				accounts = append(accounts, a)
			}
		}
	}
	var balanceBegin accounting.Balance
	var balanceDelta accounting.Balance
	for _, a := range accounts {
		balanceBegin.AddBalance(a.StartBalance)
		for _, s := range a.Splits {
			balanceDelta.Add(s.Value)
		}
	}
	if flags.market {
		var bal1, bal2 accounting.Balance
		for _, v := range balanceBegin {
			bal1.Add(L.Convert(v, flags.beginDate, L.DefaultCurrency))
		}
		var balanceEnd accounting.Balance
		balanceEnd.AddBalance(balanceBegin)
		balanceEnd.AddBalance(balanceDelta)
		for _, v := range balanceEnd {
			bal2.Add(L.Convert(v, flags.endDate, L.DefaultCurrency))
		}
		balanceDelta = bal2
		balanceDelta.SubBalance(bal1)
	}
	if flags.negate {
		var b2 accounting.Balance
		b2.SubBalance(balanceDelta)
		balanceDelta = b2
	}
	fmt.Println(balanceDelta)
	return nil
}

func runPrice(L *accounting.Ledger, flags flags, args []string) error {
	for _, p := range args {
		var v accounting.Value
		v.Amount = accounting.U
		v.Currency, _ = L.GetCurrency(p)
		v2 := L.Convert(v, flags.endDate, L.DefaultCurrency)

		fmt.Printf("Price for %s: %s\n", p, v2.FullString())
	}
	return nil
}

func Usage() {
	log.Fatalln("usage: ledger [options] <command> [args]")
}

type sliceString []string

func (s *sliceString) String() string {
	return ""
}
func (s *sliceString) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func transactionInPivot(t *accounting.Transaction, pivot sliceString) bool {
	for _, s := range t.Splits {
		for _, p := range pivot {
			if strings.Contains(strings.ToLower(s.Account.FullName()), strings.ToLower(p)) {
				return true
			}
		}
	}
	return false
}

func doPivot(L *accounting.Ledger, pivot sliceString) {
	if len(pivot) == 0 {
		return
	}
	for i := 0; i < len(L.Transactions); i++ {
		if !transactionInPivot(L.Transactions[i], pivot) {
			L.Transactions = append(L.Transactions[:i], L.Transactions[i+1:]...)
			i--
		}
	}
	for i := range L.Accounts {
		for j := 0; j < len(L.Accounts[i].Splits); j++ {
			if !transactionInPivot(L.Accounts[i].Splits[j].Transaction, pivot) {
				L.Accounts[i].Splits = append(L.Accounts[i].Splits[:j], L.Accounts[i].Splits[j+1:]...)
				j--
			}
		}
	}
}

func main() {
	var L *accounting.Ledger
	var err error
	var filename string
	os.Args = os.Args[1:]
	if len(os.Args) >= 2 && os.Args[0] == "-f" {
		filename = os.Args[1]
		os.Args = os.Args[2:]
	} else {
		filename = os.Getenv("LEDGER_FILE")
	}
	if filename == "" {
		fmt.Fprintln(os.Stderr, "ledger: no journal file specified.")
		fmt.Fprintln(os.Stderr, "Please use option -f or environment variable LEDGER_FILE")
		os.Exit(1)
	}
	L, err = accounting.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", filename, err.Error())
		os.Exit(1)
	}
	main2(L.Clone(), os.Args)
}

func main2(L *accounting.Ledger, args []string) {
	var flags flags
	var err error
	var txtBeginDate, txtEndDate, txtPeriod string
	flags.endDate = time.Now()
	f := flag.NewFlagSet("ledger", flag.ExitOnError)

	f.StringVar(&txtBeginDate, "b", "", "begin date")
	f.StringVar(&txtEndDate, "e", "", "end date")
	f.StringVar(&txtPeriod, "p", "", "period")
	f.Var(&flags.pivot, "pivot", "restrict transactions to those satisfying this pivot")
	f.BoolVar(&flags.market, "market", false, "show amounts converted to market value")
	f.BoolVar(&flags.batch, "batch", false, "show computer-ready results")
	f.BoolVar(&flags.negate, "negate", false, "change values from negative to positive (and vice versa)")
	f.Parse(args)
	if txtBeginDate != "" {
		if len(txtBeginDate) == 4 {
			txtBeginDate += "-01-01/00:00:00"
		} else if len(txtBeginDate) == 7 {
			txtBeginDate += "-01/00:00:00"
		} else if len(txtBeginDate) == 10 {
			txtBeginDate += "/00:00:00"
		}
		flags.beginDate, err = ledger.GetDate(txtBeginDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ledger: %s\n", err.Error())
			os.Exit(1)
		}
	}
	if txtEndDate != "" {
		var endOfMonth bool
		if len(txtEndDate) == 4 {
			txtEndDate += "-12-31/23:59:59"
		} else if len(txtEndDate) == 7 {
			txtEndDate += "-01/23:59:59"
			endOfMonth = true
		} else if len(txtEndDate) == 10 {
			txtEndDate = txtEndDate + "/23:59:59"
		}
		flags.endDate, err = ledger.GetDate(txtEndDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ledger: %s\n", err.Error())
			os.Exit(1)
		}
		if endOfMonth {
			flags.endDate = flags.endDate.AddDate(0, 1, -1)
		}
	}
	if flags.pivot != nil {
		doPivot(L, flags.pivot)
	}
	if txtBeginDate != "" {
		for i := len(L.Transactions) - 1; i >= 0; i-- {
			if L.Transactions[i].Time.Before(flags.beginDate) {
				L.Transactions = L.Transactions[i+1:]
				break
			}
		}
		//for i, p := range Ledger.Prices {
		//	if p.Time.After(endDate) {
		//		Ledger.Prices = Ledger.Prices[:i]
		//		break
		//	}
		//}
		for i := range L.Accounts {
			for j := len(L.Accounts[i].Splits) - 1; j >= 0; j-- {
				if L.Accounts[i].Splits[j].Time.Before(flags.beginDate) {
					L.Accounts[i].StartBalance = L.Accounts[i].Splits[j].Balance
					L.Accounts[i].Splits = L.Accounts[i].Splits[j+1:]
					break
				}
			}
		}
	}
	if txtEndDate != "" {
		for i, t := range L.Transactions {
			if t.Time.After(flags.endDate) {
				L.Transactions = L.Transactions[:i]
				break
			}
		}
		for i := range L.Accounts {
			for j, s := range L.Accounts[i].Splits {
				if s.Time.After(flags.endDate) {
					L.Accounts[i].Splits = L.Accounts[i].Splits[:j]
					break
				}
			}
		}
	}
	/*
		for i := len(Ledger.Accounts) - 1; i >= 0; i-- {
			a := Ledger.Accounts[i]
			if len(a.Children) == 0 && len(a.Splits) == 0 {
				Ledger.Accounts = append(Ledger.Accounts[:i], Ledger.Accounts[i+1:]...)
				if a.Parent != nil {
					for j := 0; j < len(a.Parent.Children); j++ {
						if a.Parent.Children[j] == a {
							a.Parent.Children = append(a.Parent.Children[:j], a.Parent.Children[j+1:]...)
							break
						}
					}
				}
			}
		}
	*/
	if len(f.Args()) == 0 {
		tableAccounts(L)
		return
	}
	if len(f.Args()) > 0 && commands[f.Args()[0]] == nil {
		log.Fatalf("ledger %s: unknown command\n", f.Args()[0])
	}
	if err = commands[f.Args()[0]](L, flags, f.Args()[1:]); err != nil {
		log.Fatalf("ledger %s: %v\n", f.Args()[0], err.Error())
	}
}

func tableAccounts(ledger *accounting.Ledger) {
	t := tableview.NewTableView()
	t.FillTable([]string{"account", "balance"}, [][]string{})
	t.SetExpansion(0, 1)
	for i, ac := range ledger.Accounts {
		// t.SetCell(i, 0, strconv.Itoa(ac.ID))
		t.SetCell(i, 0, ac.FullName())
		t.SetAlign(1, tableview.AlignRight)
		t.SetCell(i, 1, ledger.GetBalance(ac, time.Time{}).String())
	}
	t.SetSelectedFunc(func(row int) {
		tableTransactions(ledger.Accounts[row-1])
	})
	t.Run()
}

func tableTransactions(account *accounting.Account) {
	fmt.Printf("account %s: %d splits\n", account.FullName(), len(account.Splits))
	t := tableview.NewTableView()
	t.FillTable([]string{"date", "description", "value", "balance"}, [][]string{})
	t.SetExpansion(1, 1)
	for i, sp := range account.Splits {
		t.SetCell(i, 0, sp.Time.Format("02-01-2006"))
		t.SetCell(i, 1, sp.Transaction.Description)
		if v := sp.Value.String(); v != "0" {
			t.SetCell(i, 2, sp.Value.String())
		}
		t.SetAlign(2, tableview.AlignRight)
		t.SetCell(i, 3, sp.Balance.String())
		t.SetAlign(3, tableview.AlignRight)
	}
	t.Run()
}
