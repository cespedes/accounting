package ledger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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

// ReadFile fills a ledger with the data from a journal file.
func (l ledger) Read() error {
	s := NewScanner()
	s.NewFile(l.file)

	for {
		line := s.Line()
		if line.Err != nil {
			if line.Err != io.EOF {
				return line.Err
			}
			return nil
		}
		fmt.Printf("%s:%d: \"%s\"\n", line.Filename, line.LineNum, line.Text)
		text := line.Text
		var comment string
		indented := false
		if len(text) > 0 && (text[0] == ' ' || text[0] == '\t') {
			indented = true
		}
		text = strings.TrimSpace(text)
		if len(text) == 0 {
			fmt.Printf("%s:%d: empty line\n", line.Filename, line.LineNum)
			continue
		}
		if text[0] == '*' || text[0] == '#' || text[0] == ';' {
			comment = text[1:]
			fmt.Printf("%s:%d: File comment: \"%s\"\n", line.Filename, line.LineNum, comment)
			continue
		}
		if i := strings.IndexByte(text, ';'); i >= 0 {
			comment = text[i+1:]
			text = text[0:i]
		}
		pieces := strings.Fields(text)
		if len(pieces) == 0 {
			panic("len(pieces)==0 (cannot happen)")
		}
		if indented {
			fmt.Printf("%s:%d: indented line (unimplemented)\n", line.Filename, line.LineNum)
			continue
		}
		if pieces[0] == "include" {
			if len(pieces) >= 2 {
				err := s.NewFile(pieces[1])
				if err != nil {
					panic(err)
				}
			}
		}
	}
}
