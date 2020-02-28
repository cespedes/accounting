package ledger

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/cespedes/accounting"
)

/* Syntax of ledger files using EBNF:

line    = ( directive | transaction_line | split_line ) .
directive = ( include_line | account_line | price_line | default_currency_line | commodity_line ) .

letter = unicode_letter .
digit  = "0" … "9" .
digits = digit { digit } .
punct = "." | "," | "_" | "'" .
currency_char = letter | digit | "$" | "/" | "_" | "-" | "." .
currency = currency_char { currency_char } .
integer = ( digit { digit} ) | ( digit [ digit [ digit ] ] { punct digit digit digit } ) .
number = [ "-" ] integer [ punct digit { digit } ]
value = ( currency number ) | ( currency " " number ) | ( number currency ) | (number " " currency ) .
date = digit digit digit digit ( "-" | "/" | "." ) digit digit ( "-" | "/" | "." ) digit digit
indent = " " { " " }
transaction_price = ( "@" | "@@" ) value .
balance_assertion = ( "=" | "=*" | "==" | "==*" ) value [ transaction_price ] .

include_line = "include" filename .
price_line   = "P" date currency value .
default_currency_line = "D" [ currency | value ] .
transaction_line = date description .
split_line = indent account_name [ two_spaces [ value [ transaction_price ] ] [ balance_assertion ] ] .
commodity_line = "commodity" value .
account_name = ( letter | digit ) { letter | digit | ":" | " " } .
account_line = "account" account_name

*/

type scannerFile struct {
	f        *os.File
	s        *bufio.Scanner
	filename string
	lineNum  int
}

type Scanner struct {
	files []scannerFile
}

type ScannerLine struct {
	Filename string
	LineNum  int
	Text     string
	Err      error
}

const (
	lineNone = iota
	lineAccount
	lineDefaultCurrency
	lineCommodity
	linePrice
	lineTransaction
	lineSplit
	lineInclude
)

func NewScanner() *Scanner {
	s := new(Scanner)
	return s
}

func (s *Scanner) NewFile(filename string) error {
	if len(filename) > 0 && filename[0] != '/' && len(s.files) > 0 {
		filename = path.Join(path.Dir(s.files[len(s.files)-1].filename), filename)
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	s2 := bufio.NewScanner(f)
	s.files = append(s.files, scannerFile{f: f, s: s2, filename: filename})
	return nil
}

func (s *Scanner) Line() ScannerLine {
	if len(s.files) == 0 {
		return ScannerLine{Err: io.EOF}
	}
	var line ScannerLine
	file := s.files[len(s.files)-1]
	line.Filename = file.filename
	line.LineNum = file.lineNum
	if file.s.Scan() {
		s.files[len(s.files)-1].lineNum++
		line.LineNum++
		line.Text = file.s.Text()
		return line
	}
	line.Err = file.s.Err()
	if line.Err == nil {
		file.f.Close()
		s.files = s.files[:len(s.files)-1]
		return s.Line()
	}
	return line
}

func addComment(old *string, new string) {
	if *old == "" {
		*old = new
	} else {
		*old += "\n" + new
	}
}

func printTransaction(t accounting.Transaction) {
	var comment string
	if t.Comment != "" {
		comment = " /* " + t.Comment + " */"
	}
	fmt.Printf("%s %s%s\n", t.Time.Format("2006-01-02"), t.Description, comment)
	for _, s := range t.Splits {
		fmt.Printf(" > %-50s %s\n", s.Account.Name, s.Value.String())
	}
}

func (l *ledger) balanceLastTransaction(file string, line int) {
	var unbalancedSplit *accounting.Split
	balance := make(map[*accounting.Currency]int64)
	transaction := l.transactions[len(l.transactions)-1]
	for i, s := range transaction.Splits {
		if s.Value.Currency == nil {
			if unbalancedSplit != nil {
				log.Fatalf("%s:%d: more than one posting without amount", file, line)
			}
			unbalancedSplit = &transaction.Splits[i]
			continue
		}
		if s.EqValue != nil {
			balance[s.EqValue.Currency] += s.EqValue.Amount
		} else {
			balance[s.Value.Currency] += s.Value.Amount
		}
	}
	for c, a := range balance {
		if a == 0 {
			delete(balance, c)
		}
	}
	if len(balance) == 0 {
		// everything is balanced
		return
	}
	if unbalancedSplit != nil && len(balance) == 1 {
		for c, a := range balance {
			unbalancedSplit.Value.Currency = c
			unbalancedSplit.Value.Amount = -a
			return
		}
		panic("balanceLastTransaction(): assertion failed")
	}
	if unbalancedSplit != nil {
		log.Fatalf("%s:%d: could not balance account %q: two or more currencies in transaction", file, line, unbalancedSplit.Account.FullName())
	}
	if len(balance) == 1 {
		var v accounting.Value
		for c, a := range balance {
			v.Amount = a
			v.Currency = c
		}
		log.Fatalf("%s:%d: could not balance transaction: total amount is %s", file, line, v.String())
	}
	if len(balance) == 2 {
		var values []accounting.Value
		for c, a := range balance {
			var value accounting.Value
			value.Amount = a
			value.Currency = c
			values = append(values, value)
		}
		// we add 2 automatic prices, converting one currency to another and vice-versa
		var price accounting.Price
		var i *big.Int
		price.Time = transaction.Time
		price.Comment = `automatic`
		price.Currency = values[0].Currency
		i = big.NewInt(-accounting.U)
		i.Mul(i, big.NewInt(values[1].Amount))
		i.Quo(i, big.NewInt(values[0].Amount))
		price.Value.Amount = i.Int64()
		price.Value.Currency = values[1].Currency
		l.prices = append(l.prices, price)
		price.Currency = values[1].Currency
		i = big.NewInt(-accounting.U)
		i.Mul(i, big.NewInt(values[0].Amount))
		i.Quo(i, big.NewInt(values[1].Amount))
		price.Value.Amount = i.Int64()
		price.Value.Currency = values[0].Currency
		l.prices = append(l.prices, price)
		return
	}
	if len(balance) > 2 {
		log.Fatalf("%s:%d: not able to balance transactions with 3 or more currencies", file, line)
	}
	log.Printf("%s:%d", file, line)
	panic("balanceLastTransaction(): unreachable code")
}

// ReadFile fills a ledger with the data from a journal file.
func (l *ledger) Read() error {
	if l.ready {
		return nil
	}
	l.accounts = nil
	l.transactions = nil
	l.currencies = nil
	l.prices = nil
	l.defaultCurrency = nil
	s := NewScanner()
	s.NewFile(l.file)

	lastLine := lineNone
	var lastTransactionFile string
	var lastTransactionLine int
	for {
		line := s.Line()
		if line.Err != nil {
			if line.Err != io.EOF {
				return line.Err
			}
			break
		}
		// fmt.Printf("%s:%d: \"%s\"\n", line.Filename, line.LineNum, line.Text)
		text := line.Text
		comment := ""
		indented := false
		if len(text) > 0 && (text[0] == ' ' || text[0] == '\t') {
			indented = true
		}
		text = strings.TrimSpace(text)
		if len(text) == 0 {
			// empty line
			continue
		}
		if text[0] == '*' || text[0] == '#' || text[0] == ';' {
			comment = strings.TrimSpace(text[1:])
			if !indented {
				fmt.Printf("%s:%d: File comment: \"%s\"\n", line.Filename, line.LineNum, comment)
			} else {
				switch lastLine {
				case lineAccount:
					addComment(&l.accounts[len(l.accounts)-1].Comment, comment)
				case lineCommodity:
					addComment(&l.currencies[len(l.currencies)-1].Comment, comment)
				case linePrice:
					addComment(&l.prices[len(l.prices)-1].Comment, comment)
				case lineTransaction:
					addComment(&l.transactions[len(l.transactions)-1].Comment, comment)
				case lineSplit:
					addComment(&l.transactions[len(l.transactions)-1].Splits[len(l.transactions[len(l.transactions)-1].Splits)-1].Comment, comment)
				default:
					fmt.Printf("%s:%d: Wrong indented comment: \"%s\"\n", line.Filename, line.LineNum, comment)
				}
			}
			continue
		}
		if i := strings.IndexByte(text, ';'); i >= 0 {
			comment = strings.TrimSpace(text[i+1:])
			text = strings.TrimSpace(text[0:i])
		}
		if !indented && lastLine == lineSplit {
			l.balanceLastTransaction(lastTransactionFile, lastTransactionLine)
		}
		word, rest := firstWord(text)
		if !indented && word == "include" {
			lastLine = lineInclude
			newFile := rest
			err := s.NewFile(newFile)
			if err != nil {
				log.Printf("%s:%d: couldn't include file: %s\n", line.Filename, line.LineNum, err.Error())
			}
			continue
		}
		if !indented {
			date, err := getDate(word)
			if err == nil {
				var transaction accounting.Transaction
				transaction.Time = date
				transaction.Description = rest
				transaction.Comment = comment
				if len(l.transactions) > 1 && l.transactions[len(l.transactions)-1].Time.After(date) {
					log.Fatalf("%s:%d: transaction is not chronologically sorted", line.Filename, line.LineNum)
				}
				l.transactions = append(l.transactions, &transaction)
				lastLine = lineTransaction
				lastTransactionFile = line.Filename
				lastTransactionLine = line.LineNum
				continue
			}
		}
		if !indented && word == "P" {
			var price accounting.Price
			var err error
			// set price
			date, rest := firstWord(rest)
			price.Time, err = getDate(date)
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			currency, rest := firstWord(rest)
			price.Currency = l.getCurrency(currency)
			price.Value, err = l.getValue(rest)
			price.Comment = comment
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			l.prices = append(l.prices, price)
			lastLine = linePrice
			continue
		}
		if !indented && word == "D" {
			lastLine = lineDefaultCurrency
			price, err := l.getValue(rest)
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			l.defaultCurrency = price.Currency
			continue
		}
		if !indented && word == "commodity" {
			lastLine = lineCommodity
			_, err := l.getValue(rest)
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			continue
		}
		if indented && (lastLine == lineTransaction || lastLine == lineSplit) {
			// split
			t := l.transactions[len(l.transactions)-1]
			s := accounting.Split{}
			i := strings.Index(text, "  ")
			if i > 0 {
				var err error
				s.Account = l.getAccount(text[:i])
				j := strings.Index(text[i:], "@")
				if j > 0 {
					s.Value, err = l.getValue(strings.TrimSpace(text[i : i+j]))
					if err != nil {
						log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
						continue
					}
					if len(text[i:])-j < 2 {
						log.Printf("%s:%d: syntax error (no value after '@')", line.Filename, line.LineNum)
						continue
					}
					if text[i+j+1] == '@' {
						s.EqValue = new(accounting.Value)
						*s.EqValue, err = l.getValue(strings.TrimSpace(text[i+j+2:]))
						if err != nil {
							log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
							continue
						}
					} else {
						s.EqValue = new(accounting.Value)
						*s.EqValue, err = l.getValue(strings.TrimSpace(text[i+j+1:]))
						if err != nil {
							log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
							continue
						}
						k := big.NewInt(s.Value.Amount)
						k.Mul(k, big.NewInt(s.EqValue.Amount))
						k.Quo(k, big.NewInt(accounting.U))
						s.EqValue.Amount = k.Int64()
					}
				} else {
					s.Value, err = l.getValue(strings.TrimSpace(text[i:]))
					if err != nil {
						log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
						continue
					}
				}
			} else {
				s.Account = l.getAccount(text)
			}
			t.Splits = append(t.Splits, s)
			lastLine = lineSplit
			continue
		}
		log.Printf("%s:%d: UNIMPLEMENTED: \"%s\" (%s)\n", line.Filename, line.LineNum, text, comment)
	}
	if lastLine == lineSplit {
		l.balanceLastTransaction(lastTransactionFile, lastTransactionLine)
	}
	l.ready = true
	return nil
}

func (l *ledger) getAccount(s string) *accounting.Account {
	for i := range l.accounts {
		if s == l.accounts[i].Name {
			return l.accounts[i]
		}
	}
	var account accounting.Account
	account.Name = s
	account.Balance = make(accounting.Balance)
	l.accounts = append(l.accounts, &account)
	return &account
}

func (l *ledger) getCurrency(s string) *accounting.Currency {
	for i := range l.currencies {
		if s == l.currencies[i].Name {
			return l.currencies[i]
		}
	}
	var currency accounting.Currency
	currency.Name = s
	l.currencies = append(l.currencies, &currency)
	return &currency
}

func (l *ledger) getValue(s string) (accounting.Value, error) {
	var value accounting.Value
	value.Currency = new(accounting.Currency)
	var sAmount string

	if s == "" {
		return value, errors.New("empty value")
	}
	if s[0] == '-' || s[0] == '+' || (s[0] >= '0' && s[0] <= '9') {
		// first amount, then currency
		for i, c := range s {
			if !strings.ContainsRune("-+0123456789.,_'", c) {
				sAmount = s[:i]
				if unicode.IsSpace(c) {
					value.Currency.PrintSpace = true
				}
				value.Currency.Name = strings.TrimSpace(s[i:])
				goto done
			}
		}
		sAmount = s
	} else {
		// first currency, then amount
		value.Currency.PrintBefore = true
		for i := len(s) - 1; i >= 0; i-- {
			if !strings.ContainsRune("-+0123456789.,_", rune(s[i])) {
				if unicode.IsSpace(rune(s[i])) {
					value.Currency.PrintSpace = true
				}
				sAmount = s[i+1:]
				value.Currency.Name = strings.TrimSpace(s[0 : i+1])
				break
			}
		}
		if sAmount == "" {
			return value, errors.New("syntax error: currency without amount")
		}
	}
done:
	newCurrency := true
	if value.Currency.Name == "" {
		if l.defaultCurrency == nil {
			l.defaultCurrency = value.Currency
		} else {
			value.Currency = l.defaultCurrency
			newCurrency = false
		}
	} else {
		for _, c := range l.currencies {
			if c.Name == value.Currency.Name {
				value.Currency = c
				newCurrency = false
				goto done2
			}
		}
		l.currencies = append(l.currencies, value.Currency)
	}
done2:
	var sign int64 = 1
	if sAmount[0] == '-' {
		sign = -1
		sAmount = sAmount[1:]
	} else if sAmount[0] == '+' {
		sAmount = sAmount[1:]
	}
	var punct string
	punctPos, thousandPos, decimalPos := -1, -1, -1
	if c := sAmount[len(sAmount)-1]; c < '0' || c > '9' {
		return value, errors.New("syntax error: amount must end with a digit")
	}
	for i, c := range sAmount {
		if c >= '0' && c <= '9' {
			value.Amount *= 10
			value.Amount += int64(c - '0')
			continue
		}
		if i == 0 {
			return value, fmt.Errorf("syntax error: wrong position for punctuation mark '%c'", c)
		}
		if c == '-' || c == '+' {
			return value, fmt.Errorf("syntax error: wrong punctuation mark '%c'", c)
		}
		if punct == string(c) {
			// we have seen this before: this must be a thousand sign
			value.Currency.Thousand = punct
			thousandPos = punctPos
			punct, punctPos = "", -1
		}
		if value.Currency.Thousand == string(c) || (value.Currency.Thousand == "" && value.Currency.Decimal != "" && value.Currency.Decimal != string(c)) {
			value.Currency.Thousand = string(c)
			if (thousandPos == -1 && i > 3) || i-thousandPos != 4 || decimalPos > -1 {
				return value, fmt.Errorf("syntax error: wrong position for thousand sign '%s'", value.Currency.Thousand)
			}
			thousandPos = i
			continue
		}
		if punct != "" && punct != string(c) {
			// last one must be a thousand sign, and this one a decimal sign
			value.Currency.Thousand = punct
			value.Currency.Decimal = string(c)
			thousandPos = punctPos
			punct, punctPos = "", -1
		}
		if value.Currency.Decimal == string(c) || (value.Currency.Decimal == "" && value.Currency.Thousand != "" && value.Currency.Thousand != string(c)) {
			value.Currency.Decimal = string(c)
			if decimalPos > -1 {
				return value, fmt.Errorf("syntax error: more than one decimal sign '%s'", value.Currency.Decimal)
			}
			if thousandPos > -1 && i-thousandPos != 4 {
				return value, fmt.Errorf("syntax error: wrong position for thousand sign '%s'", value.Currency.Thousand)
			}
			decimalPos = i
			continue
		}
		if value.Currency.Decimal != "" && value.Currency.Thousand != "" {
			return value, fmt.Errorf("syntax error: unknown punctuacion '%c' (thousand='%s', decimal='%s')", c, value.Currency.Thousand, value.Currency.Decimal)
		}
		// TODO FIXME XXX
		// 'c' could be a decimal sign or a thousand sign
		if i > 3 {
			value.Currency.Decimal = string(c)
			decimalPos = i
		} else {
			punct = string(c)
			punctPos = i
		}
	}
	if punct != "" && len(sAmount)-punctPos != 4 {
		value.Currency.Decimal = punct
		decimalPos = punctPos
		punct, punctPos = "", -1
	}
	if punct != "" {
		return value, fmt.Errorf("syntax error: punctuation '%s' can be a thousand or a decimal", punct)
	}
	shift := 0
	if decimalPos == -1 {
		shift = 8
	} else {
		shift = len(sAmount) - decimalPos - 1
		if newCurrency {
			value.Currency.Precision = shift
		}
		shift = 8 - shift
	}
	if shift < 0 || shift > 8 {
		return value, fmt.Errorf("syntax error: too many decimal numbers")
	}
	for i := 0; i < shift; i++ {
		value.Amount *= 10
	}
	value.Amount *= sign
	return value, nil
}

func firstWord(s string) (string, string) {
	i := strings.IndexByte(s, ' ')
	if i > 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

func getDate(s string) (time.Time, error) {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, ".", "-")
	d, e := time.Parse("2006-01-02", s)
	d = d.Add(12 * time.Hour)
	if e != nil {
		d, e = time.Parse("2006-01-02-15", s)
	}
	if e != nil {
		d, e = time.Parse("2006-01-02-15-04", s)
	}
	if e != nil {
		d, e = time.Parse("2006-01-02-15-04-05", s)
	}
	return d, e
}
