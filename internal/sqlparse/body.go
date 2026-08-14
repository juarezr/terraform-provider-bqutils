package sqlparse

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// captureBody captures the AS/BEGIN body starting at startOffset in the original input.
func captureBody(input string, startOffset int) (body string, endOffset int, err *ParseError) {
	i, line, col := skipBodyPreamble(input, startOffset)
	if i >= len(input) {
		return "", i, &ParseError{Message: "expected body after AS", Line: line, Column: col, Offset: i}
	}

	if input[i] == '(' {
		return captureParenBody(input, i, line, col)
	}
	if body, end, ok, rawErr := captureQuotedBody(input, i, startOffset, line, col); ok || rawErr != nil {
		return body, end, rawErr
	}
	if strings.HasPrefix(strings.ToUpper(input[i:]), "BEGIN") {
		return captureBeginEndBody(input, i, line, col)
	}
	return captureUntilSemi(input, i)
}

func skipBodyPreamble(input string, startOffset int) (i, line, col int) {
	i = startOffset
	line, col = 1, 1
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
	return i, line, col
}

func captureParenBody(input string, i, line, col int) (string, int, *ParseError) {
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
				// Content inside parens, excluding outer parens. Do not
				// TrimSpace here so TrimIndentation can see common indent
				// on every line (including the first). TrimBody handles
				// leading/trailing whitespace when enabled.
				return input[start+1 : i-1], i, nil
			}
		} else if c == '-' && i+1 < len(input) && input[i+1] == '-' {
			i = consumeLineComment(input, i)
			continue
		}
		i++
	}
	return "", i, &ParseError{Message: "unterminated body parentheses", Line: line, Column: col, Offset: start}
}

// captureQuotedBody handles r""" / r”' / """ / ”' bodies.
// ok is true when a quoted body was recognized (success or unterminated error).
func captureQuotedBody(input string, i, startOffset, line, col int) (body string, end int, ok bool, err *ParseError) {
	if input[i] == 'r' || input[i] == 'R' {
		j := i + 1
		for j < len(input) && (input[j] == ' ' || input[j] == '\t') {
			j++
		}
		if j+2 < len(input) {
			if (input[j] == '"' && input[j+1] == '"' && input[j+2] == '"') ||
				(input[j] == '\'' && input[j+1] == '\'' && input[j+2] == '\'') {
				quote := input[j : j+3]
				j += 3
				start := j
				for j+2 < len(input) {
					if input[j:j+3] == quote {
						return input[start:j], j + 3, true, nil
					}
					j++
				}
				return "", j, true, &ParseError{Message: "unterminated raw string body", Line: line, Column: col, Offset: startOffset}
			}
		}
	}
	if i+2 < len(input) {
		if input[i:i+3] == "\"\"\"" || input[i:i+3] == "'''" {
			quote := input[i : i+3]
			i += 3
			start := i
			for i+2 < len(input) {
				if input[i:i+3] == quote {
					return input[start:i], i + 3, true, nil
				}
				i++
			}
			return "", i, true, &ParseError{Message: "unterminated string body", Line: line, Column: col, Offset: startOffset}
		}
	}
	return "", i, false, nil
}

func captureBeginEndBody(input string, i, line, col int) (string, int, *ParseError) {
	start := i
	depth := 0
	caseDepth := 0
	var ctrl struct{ iff, while, loop, forLoop, repeat int }
	ctrlOpen := func() bool {
		return ctrl.iff > 0 || ctrl.while > 0 || ctrl.loop > 0 || ctrl.forLoop > 0 || ctrl.repeat > 0
	}
	for i < len(input) {
		c := input[i]
		if c == '\'' || c == '"' {
			i = consumeQuoted(input, i, nil)
			continue
		}
		if c == '`' {
			i = consumeBacktick(input, i, nil)
			continue
		}
		if c == '-' && i+1 < len(input) && input[i+1] == '-' {
			i = consumeLineComment(input, i)
			continue
		}
		if c == '/' && i+1 < len(input) && input[i+1] == '*' {
			i = consumeBlockComment(input, i)
			continue
		}
		if isIdentStart(rune(c)) {
			wordStart := i
			_, size := utf8.DecodeRuneInString(input[i:])
			i += size
			for i < len(input) {
				r, size := utf8.DecodeRuneInString(input[i:])
				if !isIdentPart(r) {
					break
				}
				i += size
			}
			w := strings.ToUpper(input[wordStart:i])
			switch w {
			case "BEGIN":
				depth++
			case "CASE":
				caseDepth++
			case "IF":
				ctrl.iff++
			case "WHILE":
				ctrl.while++
			case "LOOP":
				ctrl.loop++
			case "FOR":
				ctrl.forLoop++
			case "REPEAT":
				ctrl.repeat++
			case "END":
				if next, end := peekSQLKeyword(input, i); isBeginEndCloser(next) {
					switch next {
					case "IF":
						if ctrl.iff > 0 {
							ctrl.iff--
						}
					case "WHILE":
						if ctrl.while > 0 {
							ctrl.while--
						}
					case "LOOP":
						if ctrl.loop > 0 {
							ctrl.loop--
						}
					case "FOR":
						if ctrl.forLoop > 0 {
							ctrl.forLoop--
						}
					case "REPEAT":
						if ctrl.repeat > 0 {
							ctrl.repeat--
						}
					case "CASE":
						if caseDepth > 0 {
							caseDepth--
						}
					}
					i = end
					continue
				}
				if caseDepth > 0 {
					caseDepth--
					continue
				}
				if ctrlOpen() {
					continue
				}
				depth--
				if depth == 0 {
					j := i
					for j < len(input) && unicode.IsSpace(rune(input[j])) {
						j++
					}
					if j < len(input) && input[j] == ';' {
						j++
					}
					return input[start:i], j, nil
				}
			}
			continue
		}
		i++
	}
	return "", i, &ParseError{Message: "unterminated BEGIN/END body", Line: line, Column: col, Offset: start}
}

func captureUntilSemi(input string, i int) (string, int, *ParseError) {
	start := i
	for start > 0 && (input[start-1] == ' ' || input[start-1] == '\t') {
		start--
	}
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
			i = consumeBacktick(input, i, nil)
			continue
		}
		if c == ';' {
			return input[start:i], i + 1, nil
		}
		i++
	}
	return input[start:], i, nil
}
