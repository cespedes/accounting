package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cespedes/accounting"
	"github.com/cespedes/accounting/backend/ledger"
)

type flags struct {
	simulate       bool    // Run simulation
	divide         float64 // how to divide the amount amounf the commodities
	numCommodities int
	numMeasures    int
	measureMonths  int
	measureDays    int
	periodMonths   int
	periodDays     int
	batch          bool // Show computer-ready results
	beginDate      time.Time
	endDate        time.Time
	debug          bool
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

func Usage() {
	log.Fatalln("usage: muscular [options] <command> [args]")
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
		fmt.Fprintln(os.Stderr, "muscular: no journal file specified.")
		fmt.Fprintln(os.Stderr, "Please use option -f or environment variable LEDGER_FILE")
		os.Exit(1)
	}
	L, err = accounting.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", filename, err.Error())
		os.Exit(1)
	}
	begin := 0
	for i := range os.Args {
		if os.Args[i] == "--" {
			if begin != i {
				main2(L.Clone(), os.Args[begin:i])
			}
			begin = i + 1
		}
	}
	if begin == 0 || begin < len(os.Args) {
		main2(L.Clone(), os.Args[begin:])
	}
}

func main2(L *accounting.Ledger, args []string) {
	var flags flags
	var err error
	var txtBeginDate, txtEndDate, txtPeriod, txtMeasurePeriod string
	flags.endDate = time.Now()
	f := flag.NewFlagSet("muscular", flag.ExitOnError)

	f.StringVar(&txtBeginDate, "b", "", "begin date")
	f.StringVar(&txtEndDate, "e", "", "end date")
	f.StringVar(&txtPeriod, "period", "1m0d", "periodicity")
	f.StringVar(&txtMeasurePeriod, "measureperiod", "3m0d", "periodicity")
	f.BoolVar(&flags.batch, "batch", false, "show computer-ready results")
	f.BoolVar(&flags.debug, "debug", false, "show debugging information")
	f.Float64Var(&flags.divide, "divide", 1.0, "how to divide amount amoung commodities")
	f.BoolVar(&flags.simulate, "simulate", false, "run a simulation")
	f.IntVar(&flags.numCommodities, "num", 3, "number of commodities where to invest")
	f.IntVar(&flags.numMeasures, "measures", 1, "number of measures")
	f.Parse(args)
	// flags.period*:
	_, err = fmt.Sscanf(txtPeriod+"_", "%dm%dd_", &flags.periodMonths, &flags.periodDays)
	if err != nil {
		flags.periodDays = 0
		_, err = fmt.Sscanf(txtPeriod+"_", "%dm_", &flags.periodMonths)
	}
	if err != nil {
		flags.periodMonths = 0
		_, err = fmt.Sscanf(txtPeriod+"_", "%dd_", &flags.periodDays)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "muscular: wrong format for period %s\n", txtPeriod)
		os.Exit(1)
	}
	// flags.measure*:
	_, err = fmt.Sscanf(txtMeasurePeriod+"_", "%dm%dd_", &flags.measureMonths, &flags.measureDays)
	if err != nil {
		flags.measureDays = 0
		_, err = fmt.Sscanf(txtMeasurePeriod+"_", "%dm_", &flags.measureMonths)
	}
	if err != nil {
		flags.measureMonths = 0
		_, err = fmt.Sscanf(txtMeasurePeriod+"_", "%dd_", &flags.measureDays)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "muscular: wrong format for measureperiod %s\n", txtMeasurePeriod)
		os.Exit(1)
	}
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
			fmt.Fprintf(os.Stderr, "muscular: %s\n", err.Error())
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
			fmt.Fprintf(os.Stderr, "muscular: %s\n", err.Error())
			os.Exit(1)
		}
		if endOfMonth {
			flags.endDate = flags.endDate.AddDate(0, 1, -1)
		}
	}
	if flags.debug {
		fmt.Printf("flags: %+v\n", flags)
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

	momentum := make([][]accounting.Value, len(f.Args()))
	mom2 := make([]float64, len(f.Args()))
	for i := range momentum {
		momentum[i] = make([]accounting.Value, flags.numMeasures+1)
		var v accounting.Value
		v.Amount = accounting.U
		v.Currency, _ = L.GetCurrency(f.Args()[i])
		momentum[i][0], _ = L.Convert(v, flags.endDate, L.DefaultCurrency)
		t := flags.endDate
		for j := 0; j < flags.numMeasures; j++ {
			t = t.AddDate(0, -flags.measureMonths, -flags.measureDays)
			momentum[i][j+1], _ = L.Convert(v, t, L.DefaultCurrency)
			mom2[i] += float64(momentum[i][0].Amount) / float64(momentum[i][j+1].Amount)
		}
		mom2[i] /= float64(flags.numMeasures)
		mom2[i] -= 1
	}
	if flags.debug {
		fmt.Printf("momentum: %+v\n", momentum)
		fmt.Printf("mom2: %+v\n", mom2)
	}
	for i := 0; i < len(mom2); i++ {
		fmt.Printf("% 2f %s\n", mom2[i], f.Args()[i])
	}
}
