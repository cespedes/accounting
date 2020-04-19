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
	var b accounting.Balance
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
			if flags.cost {
				for _, v := range a.Account.Splits[len(a.Account.Splits)-1].Balance {
					thisBal.Add(Ledger.Convert(v, time.Now(), Ledger.DefaultCurrency))
				}
				a.Account.Splits[len(a.Account.Splits)-1].Balance = thisBal
			}
			thisBal = a.Account.Splits[len(a.Account.Splits)-1].Balance
			for _, v := range thisBal {
				length := len(v.String())
				if length > maxLength {
					maxLength = length
				}
				b.Add(v)
			}
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
	for _, v := range b {
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
	var incomes, expenses []struct {
		name    string
		balance string
	}
	var income, expense, net accounting.Balance
	var nameLen = 8
	var balanceLen = 1
	for _, a := range Ledger.Accounts {
		if strings.HasPrefix(a.FullName(), "Income:") {
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
		if strings.HasPrefix(a.FullName(), "Expense:") {
			if len(a.Splits) > 0 {
				log.Printf("expense=%s b1=%s b2=%s v=%s", a.Name, a.Splits[0].Balance, a.Splits[len(a.Splits)-1].Balance, a.Splits[0].Value)
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
	}
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
	net = income.Dup()
	net.SubBalance(expense)
	fmt.Printf(" %-*s || %*s\n", nameLen, "Net:", balanceLen, net)
	return nil
}

func Usage() {
	log.Fatalln("usage: ledger [options] <command> [args]")
}

var flags struct {
	cost bool
}

func main() {
	var err error
	var filename string
	var txtBeginDate, txtEndDate string
	var beginDate, endDate time.Time
	flag.StringVar(&filename, "f", "", "journal file")
	flag.StringVar(&txtBeginDate, "b", "", "begin date")
	flag.StringVar(&txtEndDate, "e", "", "end date")
	flag.BoolVar(&flags.cost, "cost", false, "show amounts converted to their cost")
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
		endDate, err = ledger.GetDate(txtEndDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ledger: %s\n", err.Error())
			os.Exit(1)
		}
	}
	fmt.Printf("%v\n", filename)
	if len(flag.Args()) < 1 {
		Usage()
		os.Exit(1)
	}
	if commands[flag.Args()[0]] == nil {
		log.Fatalf("ledger %s: unknown command\n", flag.Args()[0])
	}
	Ledger, err = accounting.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", filename, err.Error())
		os.Exit(1)
	}
	if beginDate != (time.Time{}) {
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
	if endDate != (time.Time{}) {
		for i, t := range Ledger.Transactions {
			if t.Time.After(endDate) {
				Ledger.Transactions = Ledger.Transactions[:i]
				break
			}
		}
		for i, p := range Ledger.Prices {
			if p.Time.After(endDate) {
				Ledger.Prices = Ledger.Prices[:i]
				break
			}
		}
		for i := range Ledger.Accounts {
			for j, s := range Ledger.Accounts[i].Splits {
				if s.Time.After(endDate) {
					Ledger.Accounts[i].Splits = Ledger.Accounts[i].Splits[:j]
					break
				}
			}
		}
	}
	if err = commands[flag.Args()[0]](flag.Args()[1:]); err != nil {
		log.Fatalf("ledger %s: %v\n", flag.Args()[0], err.Error())
	}
}
