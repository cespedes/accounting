package ledger

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cespedes/accounting"
)

/* Syntax of ledger files using EBNF:

line    = ( directive | transaction_line | split_line ) .
directive = ( include_line | account_line | price_line | default_currency_line | commodity_line ) .

letter = unicode_letter .
digit  = "0" … "9" .
digits = digit { digit } .
punct = "." | "," | " " | "_" .
currency_char = letter | digit | "$" | "/" | "_" | "." .
currency = currency_char { currency_char } .
integer = ( digit { digit} ) | ( digit [ digit [ digit ] ] { punct digit digit digit } ) .
number = [ "-" ] integer [ punct digit { digit } ]
value = ( currency number ) | ( currency " " number ) | ( number currency ) | (number " " currency ) .
date = digit digit digit digit ( "-" | "/" ) digit digit ( "-" | "/" ) digit digit
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
		if !indented && strings.HasPrefix(text, "include ") {
			lastLine = lineInclude
			newFile := strings.TrimSpace(text[8:])
			err := s.NewFile(newFile)
			if err != nil {
				log.Printf("%s:%d: couldn't include file: %s\n", line.Filename, line.LineNum, err.Error())
			}
			continue
		}
		if !indented && len(text) > 11 && text[10] == ' ' && getDate(text[0:10]) != nil {
			var transaction accounting.Transaction
			transaction.Time = *getDate(text[0:10])
			transaction.Description = strings.TrimSpace(text[10:])
			transaction.Comment = comment
			l.transactions = append(l.transactions, transaction)
			lastLine = lineTransaction
			continue
		}
		log.Printf("%s:%d: UNIMPLEMENTED: \"%s\" (%s)\n", line.Filename, line.LineNum, text, comment)
	}
}

func getDate(s string) *time.Time {
	d, e := time.Parse("2006-01-02", s)
	if e != nil {
		d, e = time.Parse("2006/01/02", s)
	}
	if e != nil {
		return nil
	}
	return &d
}
