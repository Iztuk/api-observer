package query

import (
	"fmt"
)

type Lexer struct {
	Query    string
	Position int
	Tokens   []Token
}

func ParseQuery(rawString string) {
	var l Lexer = Lexer{
		Query:    rawString,
		Position: 0,
		Tokens:   make([]Token, 0),
	}

	_ = l
}

func (l *Lexer) Process() ([]Token, error) {
	for i, ch := range l.Query {
		if i != l.Position {
			continue
		}

		if !isAllowedChar(ch) {
			return nil, newQueryError(
				l.Query,
				l.Position,
				fmt.Sprintf(
					"invalid character %q",
					ch,
				),
			)
		}

		switch ch {
		case '"':
			l.readString()
		case '=', '<', '>', '!':
			l.readOperation()
		default:
			if isDigit(ch) {
				l.readNumber()
			} else {
				l.readIdentifier()
			}
		}

		l.Position++
	}

	return l.Tokens, nil
}

func (l *Lexer) readString() {
	start := l.Position

	// Skip opening quote.
	l.Position++

	arr := make([]rune, 0)

	for l.Position < len(l.Query) {
		ch := rune(l.Query[l.Position])

		if ch == '"' && !isEscaped(arr) {
			// Skip closing quote.
			l.Position++
			break
		}

		arr = append(arr, ch)
		l.Position++
	}

	l.Tokens = append(l.Tokens, Token{
		Type:     TokenString,
		Literal:  string(arr),
		Position: start,
	})
}

func isEscaped(chars []rune) bool {
	if len(chars) == 0 {
		return false
	}

	backslashes := 0

	for i := len(chars) - 1; i >= 0; i-- {
		if chars[i] != '\\' {
			break
		}

		backslashes++
	}

	return backslashes%2 != 0
}

func (l *Lexer) readNumber() error {
	start := l.Position

	arr := make([]rune, 0)

	for l.Position < len(l.Query) {
		ch := rune(l.Query[l.Position])

		if !isAllowedChar(ch) {
			return newQueryError(
				l.Query,
				l.Position,
				fmt.Sprintf(
					"invalid character %q",
					ch,
				),
			)
		}

		if !isDigit(ch) {
			break
		}

		arr = append(arr, ch)
		l.Position++
	}

	l.Tokens = append(l.Tokens, Token{
		Type:     TokenNumber,
		Literal:  string(arr),
		Position: start,
	})

	return nil
}

func (l *Lexer) readOperation() {}

func (l *Lexer) readIdentifier() {}

func isAllowedChar(ch rune) bool {
	return isLetter(ch) ||
		isDigit(ch) ||
		ch == '_' ||
		ch == '"' ||
		ch == '=' ||
		ch == '<' ||
		ch == '>' ||
		ch == '!' ||
		ch == '(' ||
		ch == ')' ||
		ch == ' '
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
