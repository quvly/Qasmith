package syntax

import (
	"strconv"
	"strings"

	"github.com/quvly/Qasmith/internal/diagnostic"
)

type token struct {
	kind, value string
	at          Pos
}

type lexer struct {
	source string
	at     Pos
}

func (l *lexer) take() byte {
	c := l.source[0]
	l.source = l.source[1:]
	l.at.Column++
	if c == '\n' {
		l.at.Line++
		l.at.Column = 1
	}
	return c
}

func letter(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_' }
func digit(c byte) bool  { return '0' <= c && c <= '9' }

func (l *lexer) nextIsDigit() bool { return len(l.source) > 0 && digit(l.source[0]) }

func (l *lexer) digits() {
	for l.nextIsDigit() {
		l.take()
	}
}

func lex(file, source string) ([]token, error) {
	l := lexer{source, Pos{File: file, Line: 1, Column: 1}}
	var tokens []token
	for len(l.source) > 0 {
		at, rest := l.at, l.source
		c := l.take()
		switch {
		case strings.ContainsRune(" \t\r\n", rune(c)):
			continue
		case strings.HasPrefix(rest, "//"):
			for len(l.source) > 0 && l.source[0] != '\n' {
				l.take()
			}
			continue
		case strings.HasPrefix(rest, "/*"):
			l.take()
			for len(l.source) > 0 && !strings.HasPrefix(l.source, "*/") {
				l.take()
			}
			if len(l.source) == 0 {
				return tokens, diagnostic.At(at, "E1001", "unterminated block comment")
			}
			l.take()
			l.take()
			continue
		case c == '"':
			for len(l.source) > 0 && l.source[0] != '\n' && l.source[0] != '"' {
				if l.take() == '\\' && len(l.source) > 0 && l.source[0] != '\n' {
					l.take()
				}
			}
			if len(l.source) == 0 || l.take() != '"' {
				return tokens, diagnostic.At(at, "E1001", "unterminated string")
			}
			value, err := strconv.Unquote(rest[:len(rest)-len(l.source)])
			if err != nil {
				return tokens, diagnostic.At(at, "E1001", "invalid string: %v", err)
			}
			tokens = append(tokens, token{"string", value, at})
		case letter(c):
			for len(l.source) > 0 && (letter(l.source[0]) || digit(l.source[0])) {
				l.take()
			}
			value := rest[:len(rest)-len(l.source)]
			kind := "identifier"
			switch value {
			case "fn", "pub", "import", "from", "as", "let", "return", "for", "in", "if", "else", "true", "false":
				kind = value
			}
			tokens = append(tokens, token{kind, value, at})
		case digit(c) || c == '.' && l.nextIsDigit():
			kind := "integer"
			l.digits()
			if c == '.' || strings.HasPrefix(l.source, ".") && !strings.HasPrefix(l.source, "..") {
				kind = "float"
				if c != '.' {
					l.take()
					l.digits()
				}
			}
			if len(l.source) > 0 && strings.ContainsRune("eE", rune(l.source[0])) {
				kind = "float"
				l.take()
				if len(l.source) > 0 && strings.ContainsRune("+-", rune(l.source[0])) {
					l.take()
				}
				if !l.nextIsDigit() {
					return tokens, diagnostic.At(at, "E1001", "numeric exponent requires digits")
				}
				l.digits()
			}
			if len(l.source) > 0 && (letter(l.source[0]) || l.source[0] == '.' && !strings.HasPrefix(l.source, "..")) {
				return tokens, diagnostic.At(at, "E1001", "malformed numeric literal")
			}
			tokens = append(tokens, token{kind, rest[:len(rest)-len(l.source)], at})
		case len(rest) >= 2 && compoundOperator(rest[:2]):
			l.take()
			tokens = append(tokens, token{rest[:2], rest[:2], at})
		case strings.ContainsRune("{}[](),:;=+-*/%<>!", rune(c)):
			tokens = append(tokens, token{string(c), string(c), at})
		default:
			return tokens, diagnostic.At(at, "E1001", "unexpected character %q", c)
		}
	}
	return append(tokens, token{"end of file", "", l.at}), nil
}

func compoundOperator(op string) bool {
	switch op {
	case "->", "..", "==", "!=", "<=", ">=", "&&", "||":
		return true
	default:
		return false
	}
}

// Lexeme exposes the compiler lexer tokens to editor clients.
type Lexeme struct {
	Kind, Value string
	At          Pos
}

func Lexemes(file, source string) []Lexeme {
	tokens, _ := lex(file, source)
	out := make([]Lexeme, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, Lexeme{t.kind, t.value, t.at})
	}
	return out
}
