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

var Ledger *accounting.Ledger

var commands = map[string]func(args []string) error{
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
}

func runAccounts(args []string) error {
	var treeFlag bool
	f := flag.NewFlagSet("accounts", flag.ExitOnError)
	f.BoolVar(&treeFlag, "tree", false, "show short account names, as a tree")
	f.Parse(args)

	for _, a := range Ledger.Accounts {
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

func runBalance(args []string) error {
	var maxLength int
	var total accounting.Balance
	var accounts []account
	if len(args) == 0 {
		for _, a := range Ledger.Accounts {
			accounts = append(accounts, account{Name: a.Name, Level: a.Level, Account: a})
		}
	} else {
		for _, a := range Ledger.Accounts {
			for _, b := range args {
				if strings.Contains(strings.ToLower(a.FullName()), strings.ToLower(b)) {
					insertAccount(&accounts, a.FullName(), 0, a)
					break
				}
			}
		}
	}
	for _, a := range accounts {
		var thisBal accounting.Balance
		if len(a.Account.Splits) > 0 {
			if flags.market {
				for _, v := range a.Account.Splits[len(a.Account.Splits)-1].Balance {
					thisBal.Add(Ledger.Convert(v, flags.endDate, Ledger.DefaultCurrency))
				}
				a.Account.Splits[len(a.Account.Splits)-1].Balance = thisBal
			}
			thisBal = a.Account.Splits[len(a.Account.Splits)-1].Balance
			for _, v := range thisBal {
				length := len(v.String())
				if length > maxLength {
					maxLength = length
				}
				total.Add(v)
			}
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

func runStats(args []string) error {
	if len(Ledger.Transactions) == 0 {
		fmt.Println("No transactions in ledger")
	} else {
		first := Ledger.Transactions[0].Time
		last := Ledger.Transactions[len(Ledger.Transactions)-1].Time
		firstYear, firstMonth, firstDay := first.Date()
		lastYear, lastMonth, lastDay := last.Date()
		end := time.Date(lastYear, lastMonth, lastDay, 0, 0, 0, 0, time.UTC)
		start := time.Date(firstYear, firstMonth, firstDay, 0, 0, 0, 0, time.UTC)
		days := int(end.Sub(start).Hours()/24.0) + 1

		fmt.Printf("Transaction span : %s to %s (%d days)\n", first.Format("2006-01-02"),
			last.Format("2006-01-02"), days)
		fmt.Printf("Transactions     : %d (%.1f per day)\n", len(Ledger.Transactions), float64(len(Ledger.Transactions))/float64(days))
		fmt.Printf("Accounts         : %d\n", len(Ledger.Accounts))
		fmt.Printf("Commodities      : %d (", len(Ledger.Currencies))
		for i, c := range Ledger.Currencies {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(c.Name)
		}
		fmt.Println(")")
		fmt.Printf("Market prices    : %d\n", len(Ledger.Prices))
	}
	return nil
}

func runPrint(args []string) error {
	ledger.Export(os.Stdout, Ledger)
	return nil
}

func runIncomeStatement(args []string) error {
	var incomeAccounts, expenseAccounts []*accounting.Account
	var incomes, expenses []struct {
		name    string
		balance string
	}
	var income, expense, net accounting.Balance
	var nameLen = 8
	var balanceLen = 1

	if len(args) == 0 {
		for _, a := range Ledger.Accounts {
			if strings.HasPrefix(a.FullName(), "Income:") {
				incomeAccounts = append(incomeAccounts, a)
			}
			if strings.HasPrefix(a.FullName(), "Expense:") {
				expenseAccounts = append(expenseAccounts, a)
			}
		}
	} else {
		for _, a := range Ledger.Accounts {
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
		for _, a := range Ledger.Accounts {
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

func runDelta(args []string) error {
	var accounts []*accounting.Account
	if len(args) == 0 {
		return nil
	}
	for _, a := range Ledger.Accounts {
		for _, b := range args {
			if strings.Contains(strings.ToLower(a.FullName()), strings.ToLower(b)) {
				accounts = append(accounts, a)
			}
		}
	}
	var balance accounting.Balance
	for _, a := range accounts {
		for _, s := range a.Splits {
			balance.Add(s.Value)
		}
	}
	if flags.negate {
		var b2 accounting.Balance
		b2.SubBalance(balance)
		balance = b2
	}
	fmt.Println(balance)
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

var flags struct {
	batch   bool
	market  bool
	negate  bool
	pivot   sliceString
	endDate time.Time
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

func doPivot(pivot sliceString) {
	if len(pivot) == 0 {
		return
	}
	for i := 0; i < len(Ledger.Transactions); i++ {
		if !transactionInPivot(Ledger.Transactions[i], pivot) {
			Ledger.Transactions = append(Ledger.Transactions[:i], Ledger.Transactions[i+1:]...)
			i--
		}
	}
	for i := range Ledger.Accounts {
		for j := 0; j < len(Ledger.Accounts[i].Splits); j++ {
			if !transactionInPivot(Ledger.Accounts[i].Splits[j].Transaction, pivot) {
				Ledger.Accounts[i].Splits = append(Ledger.Accounts[i].Splits[:j], Ledger.Accounts[i].Splits[j+1:]...)
				j--
			}
		}
	}
}

func main() {
	var err error
	var filename string
	var txtBeginDate, txtEndDate string
	var beginDate time.Time
	flags.endDate = time.Now()
	flag.StringVar(&filename, "f", "", "journal file")
	flag.StringVar(&txtBeginDate, "b", "", "begin date")
	flag.StringVar(&txtEndDate, "e", "", "end date")
	flag.Var(&flags.pivot, "pivot", "restrict transactions to those satisfying this pivot")
	flag.BoolVar(&flags.market, "market", false, "show amounts converted to market value")
	flag.BoolVar(&flags.batch, "batch", false, "show computer-ready results")
	flag.BoolVar(&flags.negate, "negate", false, "change values from negative to positive (and vice versa)")
	flag.Parse()
	if filename == "" {
		filename = os.Getenv("LEDGER_FILE")
	}
	if filename == "" {
		fmt.Fprintln(os.Stderr, "ledger: no journal file specified.  Please use option -f")
		os.Exit(1)
	}
	if txtBeginDate != "" {
		if len(txtBeginDate) == 10 {
			txtBeginDate = txtBeginDate + "/00:00:00"
		}
		beginDate, err = ledger.GetDate(txtBeginDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ledger: %s\n", err.Error())
			os.Exit(1)
		}
	}
	if txtEndDate != "" {
		if len(txtEndDate) == 10 {
			txtEndDate = txtEndDate + "/23:59:59"
		}
		flags.endDate, err = ledger.GetDate(txtEndDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ledger: %s\n", err.Error())
			os.Exit(1)
		}
	}
	if len(flag.Args()) > 0 && commands[flag.Args()[0]] == nil {
		log.Fatalf("ledger %s: unknown command\n", flag.Args()[0])
	}
	Ledger, err = accounting.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", filename, err.Error())
		os.Exit(1)
	}
	if flags.pivot != nil {
		doPivot(flags.pivot)
	}
	if txtBeginDate != "" {
		for i := len(Ledger.Transactions) - 1; i >= 0; i-- {
			if Ledger.Transactions[i].Time.Before(beginDate) {
				Ledger.Transactions = Ledger.Transactions[i+1:]
				break
			}
		}
		//for i, p := range Ledger.Prices {
		//	if p.Time.After(endDate) {
		//		Ledger.Prices = Ledger.Prices[:i]
		//		break
		//	}
		//}
		for i := range Ledger.Accounts {
			for j := len(Ledger.Accounts[i].Splits) - 1; j >= 0; j-- {
				if Ledger.Accounts[i].Splits[j].Time.Before(beginDate) {
					Ledger.Accounts[i].Splits = Ledger.Accounts[i].Splits[j+1:]
					break
				}
			}
		}
	}
	if txtEndDate != "" {
		for i, t := range Ledger.Transactions {
			if t.Time.After(flags.endDate) {
				Ledger.Transactions = Ledger.Transactions[:i]
				break
			}
		}
		for i := range Ledger.Accounts {
			for j, s := range Ledger.Accounts[i].Splits {
				if s.Time.After(flags.endDate) {
					Ledger.Accounts[i].Splits = Ledger.Accounts[i].Splits[:j]
					break
				}
			}
		}
	}
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
	if len(flag.Args()) == 0 {
		tableAccounts()
		return
	}
	if err = commands[flag.Args()[0]](flag.Args()[1:]); err != nil {
		log.Fatalf("ledger %s: %v\n", flag.Args()[0], err.Error())
	}
}

func tableAccounts() {
	t := tableview.NewTableView()
	t.FillTable([]string{"account", "balance"}, [][]string{})
	t.SetExpansion(0, 1)
	for i, ac := range Ledger.Accounts {
		// t.SetCell(i, 0, strconv.Itoa(ac.ID))
		t.SetCell(i, 0, ac.FullName())
		t.SetAlign(1, tableview.AlignRight)
		t.SetCell(i, 1, Ledger.GetBalance(ac, time.Time{}).String())
	}
	t.SetSelectedFunc(func(row int) {
		tableTransactions(Ledger.Accounts[row-1])
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
