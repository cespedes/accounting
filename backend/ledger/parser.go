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
	"regexp"
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
   (only "=" assertions are supported)

include_line = "include" filename .
price_line   = "P" date currency value .
default_currency_line = "D" [ currency | value ] .
transaction_line = date description .
split_line = indent account_name [ "  " [ value [ transaction_price ] ] [ balance_assertion ] ] .
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

type ID struct {
	filename string
	lineNum  int
}

func (id ID) String() string {
	return fmt.Sprintf("%s:%d", id.filename, id.lineNum)
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

type tag struct {
	Name  string
	Value string
}

func getTag(s string) *tag {
	re := regexp.MustCompile(`[a-z]+:.*`)
	t := re.FindString(s)
	if t == "" {
		return nil
	}
	tag := new(tag)
	i := strings.Index(t, ":")
	tag.Name = t[0:i]
	tag.Value = t[i+1:]
	return tag
}

func (l *ledgerConnection) addComment(where interface{}, comment string) {
	tag := getTag(comment)
	if tag == nil {
		l.ledger.Comments[where] = append(l.ledger.Comments[where], comment)
		return
	}
	switch x := where.(type) {
	case *accounting.Account:
		if tag.Name == "code" {
			x.Code = tag.Value
			return
		}
	case *accounting.Split:
		if tag.Name == "date" {
			t, err := GetDate(tag.Value)
			if err != nil {
				log.Printf("%s: Invalid date: %s", x.ID, tag.Value)
			} else {
				x.Time = &t
			}
			return
		}
	case *accounting.Currency:
		if tag.Name == "isin" {
			x.ISIN = tag.Value
		}
		return
	}
	// Unknown tag:
	l.ledger.Comments[where] = append(l.ledger.Comments[where], comment)
}

// Read fills a ledger with the data from a journal file.
func (l *ledgerConnection) readJournal() error {
	l.ledger.Accounts = nil
	l.ledger.Transactions = nil
	l.ledger.Currencies = nil
	l.ledger.Prices = nil
	l.ledger.Comments = make(map[interface{}][]string)
	l.ledger.Assertions = make(map[*accounting.Split]accounting.Value)
	l.ledger.SplitPrices = make(map[*accounting.Split]accounting.Value)
	l.ledger.DefaultCurrency = nil
	s := NewScanner()
	s.NewFile(l.file)

	lastLine := lineNone
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
				//fmt.Printf("%s:%d: File comment: \"%s\"\n", line.Filename, line.LineNum, comment)
			} else {
				switch lastLine {
				case lineAccount:
					var account *accounting.Account = l.ledger.Accounts[len(l.ledger.Accounts)-1]
					l.addComment(account, comment)
				case lineCommodity:
					var currency *accounting.Currency = l.ledger.Currencies[len(l.ledger.Currencies)-1]
					l.addComment(currency, comment)
				case linePrice:
					var price *accounting.Price = l.ledger.Prices[len(l.ledger.Prices)-1]
					l.addComment(price, comment)
				case lineTransaction:
					var transaction *accounting.Transaction = l.ledger.Transactions[len(l.ledger.Transactions)-1]
					l.addComment(transaction, comment)
				case lineSplit:
					var transaction *accounting.Transaction = l.ledger.Transactions[len(l.ledger.Transactions)-1]
					var split *accounting.Split = transaction.Splits[len(transaction.Splits)-1]
					l.addComment(split, comment)
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
		if !indented && word == "P" {
			var price accounting.Price
			var err error
			// set price
			date, rest := firstWord(rest)
			price.Time, err = GetDate(date)
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			if len(l.ledger.Prices) > 0 && l.ledger.Prices[len(l.ledger.Prices)-1].Time.After(price.Time) {
				log.Fatalf("%s:%d: price is not chronologically sorted", line.Filename, line.LineNum)
			}
			currency, rest := firstWord(rest)
			price.ID = &ID{filename: line.Filename, lineNum: line.LineNum}
			var newCurrency bool
			price.Currency, newCurrency = l.ledger.GetCurrency(currency)
			if newCurrency {
				log.Printf("%s:%d undefined currency %s", line.Filename, line.LineNum, price.Currency.Name)
			}
			price.Value, err, newCurrency = l.getValue(rest)
			if comment != "" {
				l.addComment(&price, comment)
			}
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			if newCurrency {
				log.Printf("%s:%d undefined currency %s", line.Filename, line.LineNum, price.Value.Currency.Name)
			}
			l.ledger.Prices = append(l.ledger.Prices, &price)
			lastLine = linePrice
			continue
		}
		if !indented && word == "D" {
			lastLine = lineDefaultCurrency
			price, err, _ := l.getValue(rest)
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			l.ledger.DefaultCurrency = price.Currency
			continue
		}
		if !indented && word == "commodity" {
			lastLine = lineCommodity
			_, err, _ := l.getValue(rest)
			if err != nil {
				log.Printf("%s:%d: Syntax error: %s", line.Filename, line.LineNum, err.Error())
				continue
			}
			continue
		}
		if !indented && word == "account" {
			lastLine = lineAccount
			_, new := l.getAccount(line.Filename, line.LineNum, rest)
			if new == false {
				log.Fatalf("%s:%d: account already defined", line.Filename, line.LineNum)
			}
			continue
		}
		if !indented {
			date, err := GetDate(word)
			if err == nil {
				if len(l.ledger.Transactions) > 0 && l.ledger.Transactions[len(l.ledger.Transactions)-1].Time.After(date) {
					log.Fatalf("%s:%d: transaction is not chronologically sorted", line.Filename, line.LineNum)
				}
				var transaction accounting.Transaction
				transaction.ID = &ID{filename: line.Filename, lineNum: line.LineNum}
				transaction.Time = date
				transaction.Description = rest
				if comment != "" {
					l.addComment(&transaction, comment)
				}
				l.ledger.Transactions = append(l.ledger.Transactions, &transaction)
				lastLine = lineTransaction
				continue
			}
		}
		if indented && (lastLine == lineTransaction || lastLine == lineSplit) {
			// this is a split
			t := l.ledger.Transactions[len(l.ledger.Transactions)-1]
			s := new(accounting.Split)
			s.ID = &ID{filename: line.Filename, lineNum: line.LineNum}
			if comment != "" {
				l.addComment(s, comment)
			}

			var err error
			var accountEnd int
			var hasValue, hasPriceAbs, hasPriceRel, hasAssertion bool
			var valueStart, valueEnd int
			var priceStart, priceEnd int
			var assertionStart, assertionEnd int
			if i := strings.Index(text, "  "); i > 0 {
				accountEnd = i
				hasValue = true
				valueStart = i + 2
				valueEnd = len(text)
			} else {
				accountEnd = len(text)
			}
			var newAccount bool
			s.Account, newAccount = l.getAccount(line.Filename, line.LineNum, text[:accountEnd])
			if newAccount == true {
				log.Printf("%s:%d undefined account %s", line.Filename, line.LineNum, s.Account.FullName())
			}
			if hasValue {
				if i := strings.Index(text[valueStart:], "@@"); i > 0 {
					valueEnd = valueStart + i
					hasPriceAbs = true
					priceStart = valueStart + i + 2
					priceEnd = len(text)
				} else if i := strings.Index(text[valueStart:], "@"); i > 0 {
					valueEnd = valueStart + i
					hasPriceRel = true
					priceStart = valueStart + i + 1
					priceEnd = len(text)
				}
				if i := strings.Index(text[valueStart:], "="); i >= 0 {
					hasAssertion = true
					assertionStart = valueStart + i + 1
					assertionEnd = len(text)
					priceEnd = valueStart + i
					if !hasPriceAbs && !hasPriceRel {
						valueEnd = valueStart + i
					}
				}
				var newCurrency bool
				s.Value, err, newCurrency = l.getValue(strings.TrimSpace(text[valueStart:valueEnd]))
				if err != nil {
					log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
					continue
				}
				if newCurrency {
					log.Printf("%s:%d undefined currency %s", line.Filename, line.LineNum, s.Value.Currency.Name)
				}
			}
			if hasPriceRel || hasPriceAbs {
				value, err, newCurrency := l.getValue(strings.TrimSpace(text[priceStart:priceEnd]))
				if err != nil {
					log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
					continue
				}
				if newCurrency {
					log.Printf("%s:%d undefined currency %s", line.Filename, line.LineNum, value.Currency.Name)
				}
				if hasPriceRel {
					k := big.NewInt(s.Value.Amount)
					k.Mul(k, big.NewInt(value.Amount))
					k.Quo(k, big.NewInt(accounting.U))
					value.Amount = k.Int64()
				}
				l.ledger.SplitPrices[s] = value
			}
			if hasAssertion {
				value, err, newCurrency := l.getValue(strings.TrimSpace(text[assertionStart:assertionEnd]))
				if err != nil {
					log.Printf("%s:%d: %s\n", line.Filename, line.LineNum, err.Error())
					continue
				}
				if newCurrency {
					log.Printf("%s:%d undefined currency %s", line.Filename, line.LineNum, value.Currency.Name)
				}
				l.ledger.Assertions[s] = value
			}
			t.Splits = append(t.Splits, s)
			lastLine = lineSplit
			continue
		}
		log.Printf("%s:%d: UNIMPLEMENTED: \"%s\" (%s)\n", line.Filename, line.LineNum, text, comment)
	}
	return nil
}

func (l *ledgerConnection) getAccount(filename string, lineNum int, str string) (acc *accounting.Account, new bool) {
	for i := range l.ledger.Accounts {
		if str == l.ledger.Accounts[i].FullName() {
			return l.ledger.Accounts[i], false
		}
	}
	var parent *accounting.Account
	if i := strings.LastIndexByte(str, ':'); i > -1 {
		parent, _ = l.getAccount(filename, lineNum, str[:i])
		str = str[i+1:]
	}
	var account accounting.Account
	account.ID = &ID{filename: filename, lineNum: lineNum}
	account.Name = str
	account.Parent = parent
	l.ledger.Accounts = append(l.ledger.Accounts, &account)
	return &account, true
}

func (l *ledgerConnection) getValue(s string) (accounting.Value, error, bool) {
	var value accounting.Value
	value.Currency = new(accounting.Currency)
	var sAmount string

	if s == "" {
		return accounting.Value{}, nil, false // empty value == zero value
	}
	if s[0] == '-' || s[0] == '+' || (s[0] >= '0' && s[0] <= '9') {
		// first amount, then currency
		for i, c := range s {
			if !strings.ContainsRune("-+0123456789.,_'", c) {
				sAmount = s[:i]
				if !unicode.IsSpace(c) {
					value.Currency.WithoutSpace = true
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
				if !unicode.IsSpace(rune(s[i])) {
					value.Currency.WithoutSpace = true
				}
				sAmount = s[i+1:]
				value.Currency.Name = strings.TrimSpace(s[0 : i+1])
				break
			}
		}
		if sAmount == "" {
			return value, errors.New("syntax error: currency without amount"), false
		}
	}
done:
	if strings.ContainsAny(value.Currency.Name, "=@") {
		return value, errors.New("syntax error: invalid character in currency"), false
	}
	newCurrency := true
	if value.Currency.Name == "" {
		if l.ledger.DefaultCurrency == nil {
			l.ledger.DefaultCurrency = value.Currency
		} else {
			value.Currency = l.ledger.DefaultCurrency
			newCurrency = false
		}
	} else {
		for _, c := range l.ledger.Currencies {
			if c.Name == value.Currency.Name {
				value.Currency = c
				newCurrency = false
				goto done2
			}
		}
		l.ledger.Currencies = append(l.ledger.Currencies, value.Currency)
	}
done2:
	var sign int64 = 1
	if sAmount[0] == '-' {
		sign = -1
		sAmount = sAmount[1:]
	} else if sAmount[0] == '+' {
		sAmount = sAmount[1:]
	}
	if len(sAmount) == 0 {
		return value, errors.New("syntax error: empty amount"), newCurrency
	}
	var punct string
	punctPos, thousandPos, decimalPos := -1, -1, -1
	if c := sAmount[len(sAmount)-1]; c < '0' || c > '9' {
		return value, errors.New("syntax error: amount must end with a digit"), newCurrency
	}
	for i, c := range sAmount {
		if c >= '0' && c <= '9' {
			value.Amount *= 10
			value.Amount += int64(c - '0')
			continue
		}
		if i == 0 {
			return value, fmt.Errorf("syntax error: wrong position for punctuation mark '%c'", c), newCurrency
		}
		if c == '-' || c == '+' {
			return value, fmt.Errorf("syntax error: wrong punctuation mark '%c'", c), newCurrency
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
				return value, fmt.Errorf("syntax error: wrong position for thousand sign '%s'", value.Currency.Thousand), newCurrency
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
				return value, fmt.Errorf("syntax error: more than one decimal sign '%s'", value.Currency.Decimal), newCurrency
			}
			if thousandPos > -1 && i-thousandPos != 4 {
				return value, fmt.Errorf("syntax error: wrong position for thousand sign '%s'", value.Currency.Thousand), newCurrency
			}
			decimalPos = i
			continue
		}
		if value.Currency.Decimal != "" && value.Currency.Thousand != "" {
			return value, fmt.Errorf("syntax error: unknown punctuacion '%c' (thousand='%s', decimal='%s')", c, value.Currency.Thousand, value.Currency.Decimal), newCurrency
		}
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
		return value, fmt.Errorf("syntax error: punctuation '%s' can be a thousand or a decimal", punct), newCurrency
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
		return value, fmt.Errorf("syntax error: too many decimal numbers"), newCurrency
	}
	for i := 0; i < shift; i++ {
		value.Amount *= 10
	}
	value.Amount *= sign
	if value.Currency.Decimal == "" {
		if value.Currency.Thousand != "." {
			value.Currency.Decimal = "."
		} else {
			value.Currency.Decimal = ","
		}
	}
	return value, nil, newCurrency
}

func firstWord(s string) (string, string) {
	i := strings.IndexByte(s, ' ')
	if i > 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

// GetDate returns a time from a string.
func GetDate(s string) (time.Time, error) {
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
