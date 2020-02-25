package ledger

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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

func NewScanner() *Scanner {
	s := new(Scanner)
	return s
}

func (s *Scanner) NewFile(filename string) error {
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
		s.files = s.files[:len(s.files)-1]
		return s.Line()
	}
	return line
}

const (
	lineNone = iota
	lineAccount
	lineCommodity
	linePrice
	lineTransaction
	lineSplit
	lineInclude
)

func addComment(old *string, new string) {
	if *old == "" {
		*old = new
	} else {
		*old += "\n" + new
	}
}

func (l *ledger) balanceLastTransaction(line ScannerLine) {
	var unbalancedSplit *accounting.Split
	balance := make(map[*accounting.Currency]int64)
	transaction := l.transactions[len(l.transactions)-1]
	for _, s := range transaction.Splits {
		if s.Value.Currency == nil {
			if unbalancedSplit != nil {
				log.Fatalf("%s:%d: more than one posting without amount", line.Filename, line.LineNum)
			}
			unbalancedSplit = &s
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
		} else {
			if unbalancedSplit == nil {
				log.Fatalf("%s:%d: could not balance transaction", line.Filename, line.LineNum)
			}
			unbalancedSplit.Value.Currency = c
			unbalancedSplit.Value.Amount = -a
			unbalancedSplit = nil
		}
	}
	if unbalancedSplit != nil {
		log.Fatalf("%s:%d: could not balance transaction", line.Filename, line.LineNum)
	}
}

// ReadFile fills a ledger with the data from a journal file.
func (l *ledger) Read() error {
	s := NewScanner()
	s.NewFile(l.file)

	lastLine := lineNone
	for {
		line := s.Line()
		if line.Err != nil {
			if line.Err != io.EOF {
				return line.Err
			}
			return nil
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
					addComment(&l.transactions[len(l.transactions)-1].Splits[len(l.transactions[len(l.transactions)-1].Splits)].Comment, comment)
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
			l.balanceLastTransaction(line)
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
				l.transactions = append(l.transactions, transaction)
				lastLine = lineTransaction
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
		log.Printf("%s:%d: UNIMPLEMENTED: \"%s\" (%s)\n", line.Filename, line.LineNum, text, comment)
	}
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
	if s[0] == '-' || (s[0] >= '0' && s[0] <= '9') {
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
	d, e := time.Parse("2006-01-02", s)
	if e != nil {
		d, e = time.Parse("2006-01-02-15-04-05", s)
	}
	if e != nil {
		d, e = time.Parse("2006-01-02-15-04", s)
	}
	if e != nil {
		d, e = time.Parse("2006-01-02-15", s)
	}
	return d, e
}
