package sqlparse

import (
	"strings"
)

// TrimBody removes leading/trailing whitespace and leading/trailing empty lines.
func TrimBody(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	// drop leading empty
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// TrimIndentation removes the common first-level leading whitespace from every
// non-empty line (classic dedent). Deeper indentation relative to that first
// level is preserved. Blank lines are ignored when computing the common indent.
func TrimIndentation(s string) string {
	lines := strings.Split(s, "\n")
	indent := ""
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		leading := leadingWhitespace(line)
		if !found {
			indent = leading
			found = true
			continue
		}
		indent = commonLeadingWhitespace(indent, leading)
		if indent == "" {
			return s
		}
	}
	if !found || indent == "" {
		return s
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if strings.HasPrefix(line, indent) {
			out[i] = line[len(indent):]
			continue
		}
		out[i] = line
	}
	return strings.Join(out, "\n")
}

func leadingWhitespace(s string) string {
	i := 0
	for i < len(s) {
		if s[i] != ' ' && s[i] != '\t' {
			break
		}
		i++
	}
	return s[:i]
}

func commonLeadingWhitespace(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

// TrimComments removes -- and /* */ comments from SQL body text, preserving string literals.
func TrimComments(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		// strings
		if s[i] == '\'' || s[i] == '"' {
			q := s[i]
			b.WriteByte(q)
			i++
			if i+1 < len(s) && s[i] == q && s[i+1] == q {
				// triple
				b.WriteByte(q)
				b.WriteByte(q)
				i += 2
				for i+2 < len(s) {
					if s[i] == q && s[i+1] == q && s[i+2] == q {
						b.WriteByte(q)
						b.WriteByte(q)
						b.WriteByte(q)
						i += 3
						break
					}
					b.WriteByte(s[i])
					i++
				}
				continue
			}
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					b.WriteByte(s[i])
					b.WriteByte(s[i+1])
					i += 2
					continue
				}
				b.WriteByte(s[i])
				if s[i] == q {
					i++
					break
				}
				i++
			}
			continue
		}
		if s[i] == '`' {
			b.WriteByte('`')
			i++
			for i < len(s) && s[i] != '`' {
				b.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				b.WriteByte('`')
				i++
			}
			continue
		}
		if s[i] == '-' && i+1 < len(s) && s[i+1] == '-' {
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if s[i] == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	// collapse leftover trailing spaces on lines lightly
	return b.String()
}

// SplitQualifiedName splits project.dataset.object or dataset.object or object.
func SplitQualifiedName(name string) (project, dataset, object string) {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "`")
	parts := strings.Split(name, ".")
	for i := range parts {
		parts[i] = strings.Trim(parts[i], "` ")
	}
	switch len(parts) {
	case 1:
		return "", "", parts[0]
	case 2:
		return "", parts[0], parts[1]
	default:
		return parts[0], parts[1], parts[len(parts)-1]
	}
}
