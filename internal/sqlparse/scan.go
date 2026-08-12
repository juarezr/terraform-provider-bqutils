package sqlparse

import "strings"

// consumeQuoted advances past a '...', "...", ”'...”', or """...""" literal
// starting at i. If w is non-nil, the literal is copied into w.
func consumeQuoted(s string, i int, w *strings.Builder) int {
	write := func(b byte) {
		if w != nil {
			w.WriteByte(b)
		}
	}
	q := s[i]
	write(q)
	i++
	if i+1 < len(s) && s[i] == q && s[i+1] == q {
		write(q)
		write(q)
		i += 2
		for i+2 < len(s) {
			if s[i] == q && s[i+1] == q && s[i+2] == q {
				write(q)
				write(q)
				write(q)
				return i + 3
			}
			write(s[i])
			i++
		}
		return i
	}
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			write(s[i])
			write(s[i+1])
			i += 2
			continue
		}
		write(s[i])
		if s[i] == q {
			return i + 1
		}
		i++
	}
	return i
}

// consumeBacktick advances past a `...` identifier starting at i.
func consumeBacktick(s string, i int, w *strings.Builder) int {
	write := func(b byte) {
		if w != nil {
			w.WriteByte(b)
		}
	}
	write('`')
	i++
	for i < len(s) && s[i] != '`' {
		write(s[i])
		i++
	}
	if i < len(s) {
		write('`')
		i++
	}
	return i
}

// consumeLineComment advances past a -- comment starting at i.
func consumeLineComment(s string, i int) int {
	i += 2
	for i < len(s) && s[i] != '\n' {
		i++
	}
	return i
}

// consumeBlockComment advances past a /* ... */ comment starting at i.
func consumeBlockComment(s string, i int) int {
	i += 2
	for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
		i++
	}
	if i+1 < len(s) {
		i += 2
	}
	return i
}
