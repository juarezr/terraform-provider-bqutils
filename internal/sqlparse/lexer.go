package sqlparse

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// token kinds used by the hand-written + goyacc-facing lexer
const (
	tokEOF = iota
	tokIdent
	tokString
	tokNumber
	tokRawString // r"""...""" body-like
	tokBody      // captured AS body
	tokCreate
	tokOr
	tokReplace
	tokTemporary
	tokTemp
	tokFunction
	tokTable
	tokProcedure
	tokAggregate
	tokView
	tokMaterialized
	tokIf
	tokNot
	tokExists
	tokReturns
	tokLanguage
	tokOptions
	tokAs
	tokRemote
	tokWith
	tokConnection
	tokPartition
	tokBy
	tokCluster
	tokAny
	tokType
	tokIn
	tokOut
	tokInout
	tokTrue
	tokFalse
	tokBegin
	tokEnd
	tokLParen
	tokRParen
	tokLBracket
	tokRBracket
	tokLAngle
	tokRAngle
	tokComma
	tokDot
	tokEq
	tokSemi
	tokStar
)

type token struct {
	kind   int
	lit    string
	line   int
	col    int
	offset int
}

type lexer struct {
	input  string
	pos    int
	line   int
	col    int
	start  int
	tokens []token
	idx    int
	err    *ParseError
}

func newLexer(input string) *lexer {
	l := &lexer{input: input, line: 1, col: 1}
	l.scanAll()
	return l
}

func (l *lexer) peek() token {
	if l.idx >= len(l.tokens) {
		return token{kind: tokEOF, line: l.line, col: l.col, offset: len(l.input)}
	}
	return l.tokens[l.idx]
}

func (l *lexer) next() token {
	t := l.peek()
	if l.idx < len(l.tokens) {
		l.idx++
	}
	return t
}

func (l *lexer) backup() {
	if l.idx > 0 {
		l.idx--
	}
}

func (l *lexer) scanAll() {
	for {
		l.skipSpaceAndHeaderComments()
		if l.pos >= len(l.input) {
			l.tokens = append(l.tokens, token{kind: tokEOF, line: l.line, col: l.col, offset: l.pos})
			return
		}
		line, col, off := l.line, l.col, l.pos
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])

		switch r {
		case '(':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokLParen, lit: "(", line: line, col: col, offset: off})
		case ')':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokRParen, lit: ")", line: line, col: col, offset: off})
		case '[':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokLBracket, lit: "[", line: line, col: col, offset: off})
		case ']':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokRBracket, lit: "]", line: line, col: col, offset: off})
		case '<':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokLAngle, lit: "<", line: line, col: col, offset: off})
		case '>':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokRAngle, lit: ">", line: line, col: col, offset: off})
		case ',':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokComma, lit: ",", line: line, col: col, offset: off})
		case '.':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokDot, lit: ".", line: line, col: col, offset: off})
		case '=':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokEq, lit: "=", line: line, col: col, offset: off})
		case ';':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokSemi, lit: ";", line: line, col: col, offset: off})
		case '*':
			l.advance(size)
			l.tokens = append(l.tokens, token{kind: tokStar, lit: "*", line: line, col: col, offset: off})
		case '`':
			lit := l.scanBacktickIdent()
			l.tokens = append(l.tokens, token{kind: tokIdent, lit: lit, line: line, col: col, offset: off})
		case '\'', '"':
			lit, ok := l.scanQuotedString(r)
			if !ok {
				return
			}
			l.tokens = append(l.tokens, token{kind: tokString, lit: lit, line: line, col: col, offset: off})
		default:
			if unicode.IsDigit(r) {
				lit := l.scanNumber()
				l.tokens = append(l.tokens, token{kind: tokNumber, lit: lit, line: line, col: col, offset: off})
				continue
			}
			if isIdentStart(r) {
				if (r == 'r' || r == 'R') && l.lookingAtRawString(size) {
					lit, ok := l.scanRawTripleString()
					if !ok {
						return
					}
					l.tokens = append(l.tokens, token{kind: tokRawString, lit: lit, line: line, col: col, offset: off})
					continue
				}
				lit := l.scanIdent()
				kind := keywordKind(lit)
				l.tokens = append(l.tokens, token{kind: kind, lit: lit, line: line, col: col, offset: off})
				// Stop header lexing at AS — body is captured from raw input later.
				if kind == tokAs {
					l.tokens = append(l.tokens, token{kind: tokEOF, line: l.line, col: l.col, offset: l.pos})
					return
				}
				continue
			}
			// Allow operators that appear in OPTIONS expressions (INTERVAL literals etc. use strings)
			if r == '!' || r == '<' || r == '>' || r == '|' || r == '&' || r == '+' || r == '-' || r == '/' || r == '%' {
				l.advance(size)
				if r == '!' && l.pos < len(l.input) && l.input[l.pos] == '=' {
					l.advance(1)
				}
				// skip as opaque; not needed in header
				continue
			}
			l.err = &ParseError{Message: fmtUnexpected(r), Line: line, Column: col, Offset: off}
			return
		}
	}
}

func fmtUnexpected(r rune) string {
	return "unexpected character '" + string(r) + "'"
}

func (l *lexer) advance(n int) {
	for i := 0; i < n && l.pos < len(l.input); i++ {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *lexer) skipSpaceAndHeaderComments() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if unicode.IsSpace(r) {
			l.advance(size)
			continue
		}
		if r == '-' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '-' {
			// line comment
			l.advance(2)
			for l.pos < len(l.input) && l.input[l.pos] != '\n' {
				l.advance(1)
			}
			continue
		}
		if r == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '*' {
			l.advance(2)
			for l.pos+1 < len(l.input) {
				if l.input[l.pos] == '*' && l.input[l.pos+1] == '/' {
					l.advance(2)
					break
				}
				l.advance(1)
			}
			continue
		}
		return
	}
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_' || r == '@'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$'
}

func (l *lexer) scanIdent() string {
	start := l.pos
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !isIdentPart(r) {
			break
		}
		l.advance(size)
	}
	return l.input[start:l.pos]
}

func (l *lexer) scanBacktickIdent() string {
	l.advance(1) // `
	start := l.pos
	for l.pos < len(l.input) {
		if l.input[l.pos] == '`' {
			lit := l.input[start:l.pos]
			l.advance(1)
			return lit
		}
		l.advance(1)
	}
	l.err = &ParseError{Message: "unterminated backtick identifier", Line: l.line, Column: l.col, Offset: l.pos}
	return l.input[start:]
}

func (l *lexer) scanQuotedString(quote rune) (string, bool) {
	// Support ''' and """ triple quotes as well as single-char quotes
	if l.pos+2 < len(l.input) {
		s := l.input[l.pos : l.pos+3]
		if (quote == '\'' && s == "'''") || (quote == '"' && s == "\"\"\"") {
			l.advance(3)
			start := l.pos
			end := s
			for l.pos+2 < len(l.input) {
				if l.input[l.pos:l.pos+3] == end {
					lit := l.input[start:l.pos]
					l.advance(3)
					return lit, true
				}
				l.advance(1)
			}
			l.err = &ParseError{Message: "unterminated string", Line: l.line, Column: l.col, Offset: l.pos}
			return "", false
		}
	}
	l.advance(1)
	var b strings.Builder
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if r == '\\' && l.pos+1 < len(l.input) {
			l.advance(1)
			nr, nsize := utf8.DecodeRuneInString(l.input[l.pos:])
			b.WriteRune(nr)
			l.advance(nsize)
			continue
		}
		if r == quote {
			l.advance(size)
			return b.String(), true
		}
		b.WriteRune(r)
		l.advance(size)
	}
	l.err = &ParseError{Message: "unterminated string", Line: l.line, Column: l.col, Offset: l.pos}
	return "", false
}

func (l *lexer) lookingAtRawString(rSize int) bool {
	rest := l.input[l.pos+rSize:]
	return strings.HasPrefix(rest, "\"\"\"") || strings.HasPrefix(rest, "'''")
}

func (l *lexer) scanRawTripleString() (string, bool) {
	// r""" or r'''
	l.advance(1) // r
	end := l.input[l.pos : l.pos+3]
	l.advance(3)
	start := l.pos
	for l.pos+2 < len(l.input) {
		if l.input[l.pos:l.pos+3] == end {
			lit := l.input[start:l.pos]
			l.advance(3)
			return lit, true
		}
		l.advance(1)
	}
	l.err = &ParseError{Message: "unterminated raw string", Line: l.line, Column: l.col, Offset: l.pos}
	return "", false
}

func (l *lexer) scanNumber() string {
	start := l.pos
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsDigit(r) && r != '.' {
			break
		}
		l.advance(size)
	}
	return l.input[start:l.pos]
}

func keywordKind(lit string) int {
	switch strings.ToUpper(lit) {
	case "CREATE":
		return tokCreate
	case "OR":
		return tokOr
	case "REPLACE":
		return tokReplace
	case "TEMPORARY":
		return tokTemporary
	case "TEMP":
		return tokTemp
	case "FUNCTION":
		return tokFunction
	case "TABLE":
		return tokTable
	case "PROCEDURE":
		return tokProcedure
	case "AGGREGATE":
		return tokAggregate
	case "VIEW":
		return tokView
	case "MATERIALIZED":
		return tokMaterialized
	case "IF":
		return tokIf
	case "NOT":
		return tokNot
	case "EXISTS":
		return tokExists
	case "RETURNS":
		return tokReturns
	case "LANGUAGE":
		return tokLanguage
	case "OPTIONS":
		return tokOptions
	case "AS":
		return tokAs
	case "REMOTE":
		return tokRemote
	case "WITH":
		return tokWith
	case "CONNECTION":
		return tokConnection
	case "PARTITION":
		return tokPartition
	case "BY":
		return tokBy
	case "CLUSTER":
		return tokCluster
	case "ANY":
		return tokAny
	case "TYPE":
		return tokType
	case "IN":
		return tokIn
	case "OUT":
		return tokOut
	case "INOUT":
		return tokInout
	case "TRUE":
		return tokTrue
	case "FALSE":
		return tokFalse
	case "BEGIN":
		return tokBegin
	case "END":
		return tokEnd
	default:
		return tokIdent
	}
}

// captureBodyFrom captures the AS body starting at current lexer position in the original input.
// rawOffset is the byte offset in l.input where body content starts (after AS and optional whitespace).
func captureBody(input string, startOffset int) (body string, endOffset int, err *ParseError) {
	// skip spaces
	i := startOffset
	line, col := 1, 1
	for j := 0; j < startOffset && j < len(input); j++ {
		if input[j] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	for i < len(input) && (input[i] == ' ' || input[i] == '\t' || input[i] == '\n' || input[i] == '\r') {
		if input[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		i++
	}
	if i >= len(input) {
		return "", i, &ParseError{Message: "expected body after AS", Line: line, Column: col, Offset: i}
	}

	// Parenthesized SQL body: ( ... )
	if input[i] == '(' {
		depth := 0
		start := i
		inStr := byte(0)
		triple := false
		for i < len(input) {
			c := input[i]
			if inStr != 0 {
				if triple {
					if i+2 < len(input) && input[i] == inStr && input[i+1] == inStr && input[i+2] == inStr {
						i += 3
						inStr = 0
						triple = false
						continue
					}
					i++
					continue
				}
				if c == '\\' && i+1 < len(input) {
					i += 2
					continue
				}
				if c == inStr {
					inStr = 0
				}
				i++
				continue
			}
			if c == '\'' || c == '"' {
				if i+2 < len(input) && input[i+1] == c && input[i+2] == c {
					inStr = c
					triple = true
					i += 3
					continue
				}
				inStr = c
				i++
				continue
			}
			if c == '(' {
				depth++
			} else if c == ')' {
				depth--
				if depth == 0 {
					i++
					// content inside parens, excluding outer parens
					inner := strings.TrimSpace(input[start+1 : i-1])
					return inner, i, nil
				}
			} else if c == '-' && i+1 < len(input) && input[i+1] == '-' {
				i += 2
				for i < len(input) && input[i] != '\n' {
					i++
				}
				continue
			}
			i++
		}
		return "", i, &ParseError{Message: "unterminated body parentheses", Line: line, Column: col, Offset: start}
	}

	// Raw / triple string body
	if (input[i] == 'r' || input[i] == 'R') && i+4 < len(input) {
		rest := input[i+1:]
		if strings.HasPrefix(rest, "\"\"\"") || strings.HasPrefix(rest, "'''") {
			quote := rest[:3]
			i += 1 + 3
			start := i
			for i+2 < len(input) {
				if input[i:i+3] == quote {
					body := input[start:i]
					i += 3
					return body, i, nil
				}
				i++
			}
			return "", i, &ParseError{Message: "unterminated raw string body", Line: line, Column: col, Offset: startOffset}
		}
	}
	if i+2 < len(input) {
		if input[i:i+3] == "\"\"\"" || input[i:i+3] == "'''" {
			quote := input[i : i+3]
			i += 3
			start := i
			for i+2 < len(input) {
				if input[i:i+3] == quote {
					body := input[start:i]
					i += 3
					return body, i, nil
				}
				i++
			}
			return "", i, &ParseError{Message: "unterminated string body", Line: line, Column: col, Offset: startOffset}
		}
	}

	// BEGIN ... END
	upper := strings.ToUpper(input[i:])
	if strings.HasPrefix(upper, "BEGIN") {
		start := i
		depth := 0
		for i < len(input) {
			// skip comments/strings roughly via scanning words
			for i < len(input) && unicode.IsSpace(rune(input[i])) {
				i++
			}
			if i >= len(input) {
				break
			}
			if input[i] == '-' && i+1 < len(input) && input[i+1] == '-' {
				for i < len(input) && input[i] != '\n' {
					i++
				}
				continue
			}
			wordStart := i
			for i < len(input) && (unicode.IsLetter(rune(input[i])) || input[i] == '_') {
				i++
			}
			if i > wordStart {
				w := strings.ToUpper(input[wordStart:i])
				if w == "BEGIN" {
					depth++
				} else if w == "END" {
					depth--
					if depth == 0 {
						// optional ;
						j := i
						for j < len(input) && unicode.IsSpace(rune(input[j])) {
							j++
						}
						if j < len(input) && input[j] == ';' {
							j++
						}
						return strings.TrimSpace(input[start:i]), j, nil
					}
				}
				continue
			}
			i++
		}
		return "", i, &ParseError{Message: "unterminated BEGIN/END body", Line: line, Column: col, Offset: start}
	}

	// View-style: rest until semicolon (not inside strings)
	start := i
	inStr := byte(0)
	for i < len(input) {
		c := input[i]
		if inStr != 0 {
			if c == '\\' && i+1 < len(input) {
				i += 2
				continue
			}
			if c == inStr {
				inStr = 0
			}
			i++
			continue
		}
		if c == '\'' || c == '"' {
			inStr = c
			i++
			continue
		}
		if c == '`' {
			i++
			for i < len(input) && input[i] != '`' {
				i++
			}
			if i < len(input) {
				i++
			}
			continue
		}
		if c == ';' {
			return strings.TrimSpace(input[start:i]), i + 1, nil
		}
		i++
	}
	return strings.TrimSpace(input[start:]), i, nil
}
