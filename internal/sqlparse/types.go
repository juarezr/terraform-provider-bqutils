package sqlparse

import (
	"encoding/json"
	"strings"
)

// sqlTypeToJSON converts a BigQuery SQL type string into StandardSqlDataType JSON.
func sqlTypeToJSON(typeStr string) (string, error) {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return "", nil
	}
	upper := strings.ToUpper(typeStr)
	if upper == "ANY TYPE" || upper == "ANY_TYPE" {
		return "", nil // caller sets argument_kind
	}
	t, err := parseType(typeStr)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type stdType struct {
	TypeKind         string   `json:"typeKind"`
	ArrayElementType *stdType `json:"arrayElementType,omitempty"`
	StructType       *structT `json:"structType,omitempty"`
}

type structT struct {
	Fields []structField `json:"fields,omitempty"`
}

type structField struct {
	Name string   `json:"name,omitempty"`
	Type *stdType `json:"type,omitempty"`
}

func parseType(s string) (*stdType, error) {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	if strings.HasPrefix(upper, "ARRAY<") || strings.HasPrefix(upper, "ARRAY <") {
		inner := extractAngleContents(s)
		elem, err := parseType(inner)
		if err != nil {
			return nil, err
		}
		return &stdType{TypeKind: "ARRAY", ArrayElementType: elem}, nil
	}

	if strings.HasPrefix(upper, "STRUCT<") || strings.HasPrefix(upper, "STRUCT <") {
		inner := extractAngleContents(s)
		fields, err := parseStructFields(inner)
		if err != nil {
			return nil, err
		}
		return &stdType{TypeKind: "STRUCT", StructType: &structT{Fields: fields}}, nil
	}

	if strings.HasPrefix(upper, "TABLE<") || strings.HasPrefix(upper, "TABLE <") {
		inner := extractAngleContents(s)
		fields, err := parseStructFields(inner)
		if err != nil {
			return nil, err
		}
		// Represent TABLE similarly to struct for StandardSqlDataType tableType
		return &stdType{TypeKind: "TABLE", StructType: &structT{Fields: fields}}, nil
	}

	// Simple type: take first identifier (ignore params like NUMERIC(10,2))
	base := s
	if i := strings.IndexAny(s, "<("); i >= 0 {
		base = s[:i]
	}
	base = strings.TrimSpace(base)
	kind := normalizeTypeKind(base)
	return &stdType{TypeKind: kind}, nil
}

func normalizeTypeKind(name string) string {
	u := strings.ToUpper(strings.TrimSpace(name))
	aliases := map[string]string{
		"INT":        "INT64",
		"INTEGER":    "INT64",
		"SMALLINT":   "INT64",
		"BIGINT":     "INT64",
		"BYTEINT":    "INT64",
		"FLOAT":      "FLOAT64",
		"DOUBLE":     "FLOAT64",
		"BOOLEAN":    "BOOL",
		"DEC":        "NUMERIC",
		"BIGDECIMAL": "BIGNUMERIC",
	}
	if v, ok := aliases[u]; ok {
		return v
	}
	return u
}

func extractAngleContents(s string) string {
	start := strings.Index(s, "<")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				return s[start+1 : i]
			}
		}
	}
	return strings.TrimPrefix(s[start+1:], "")
}

func parseStructFields(inner string) ([]structField, error) {
	parts := splitTopLevel(inner, ',')
	var fields []structField
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// name type  OR  type alone
		name, typ := splitNameType(p)
		st, err := parseType(typ)
		if err != nil {
			return nil, err
		}
		fields = append(fields, structField{Name: name, Type: st})
	}
	return fields, nil
}

func splitNameType(p string) (name, typ string) {
	p = strings.TrimSpace(p)
	// If starts with type keyword without name
	upper := strings.ToUpper(p)
	typeStarts := []string{"ARRAY", "STRUCT", "TABLE", "STRING", "INT64", "INT", "INTEGER", "FLOAT64", "FLOAT", "BOOL", "BOOLEAN", "BYTES", "DATE", "DATETIME", "TIME", "TIMESTAMP", "NUMERIC", "BIGNUMERIC", "GEOGRAPHY", "JSON", "INTERVAL", "RANGE"}
	for _, t := range typeStarts {
		if upper == t || strings.HasPrefix(upper, t+"<") || strings.HasPrefix(upper, t+" ") || strings.HasPrefix(upper, t+"(") {
			return "", p
		}
	}
	// name is first ident
	i := 0
	for i < len(p) && (isIdentPart(rune(p[i])) || p[i] == '`') {
		if p[i] == '`' {
			i++
			for i < len(p) && p[i] != '`' {
				i++
			}
			if i < len(p) {
				i++
			}
			break
		}
		i++
	}
	name = strings.Trim(p[:i], "` ")
	typ = strings.TrimSpace(p[i:])
	return name, typ
}

func splitTopLevel(s string, sep rune) []string {
	var parts []string
	depth := 0
	start := 0
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			inStr = c
			continue
		}
		switch c {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if c == byte(sep) && depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func tableTypeToJSON(columns []ColumnDef) (string, error) {
	// StandardSqlTableType-ish for return_table_type
	type col struct {
		Name string   `json:"name"`
		Type *stdType `json:"type"`
	}
	type tableType struct {
		Columns []col `json:"columns"`
	}
	var cols []col
	for _, c := range columns {
		cols = append(cols, col{
			Name: c.Name,
			Type: &stdType{TypeKind: "STRING"},
		})
	}
	b, err := json.Marshal(tableType{Columns: cols})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func columnsToSchemaJSON(cols []ColumnDef) (string, error) {
	type field struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Mode        string `json:"mode"`
		Description string `json:"description,omitempty"`
	}
	var fields []field
	for _, c := range cols {
		fields = append(fields, field{
			Name:        c.Name,
			Type:        "STRING",
			Mode:        "NULLABLE",
			Description: c.Description,
		})
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// tableArgTypeJSON builds data_type JSON for a TABLE argument.
func tableArgTypeJSON(columns []ColumnDef) (string, error) {
	type field struct {
		Name string   `json:"name,omitempty"`
		Type *stdType `json:"type"`
	}
	type tableType struct {
		Columns []field `json:"columns"`
	}
	payload := map[string]any{
		"typeKind": "TABLE",
	}
	if len(columns) > 0 {
		var cols []field
		for _, c := range columns {
			cols = append(cols, field{Name: c.Name, Type: &stdType{TypeKind: "STRING"}})
		}
		payload["tableType"] = tableType{Columns: cols}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
