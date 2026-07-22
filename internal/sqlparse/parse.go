package sqlparse

import (
	"fmt"
	"strconv"
	"strings"
)

// Options controls post-processing of the parsed body.
type Options struct {
	TrimBody     bool
	TrimComments bool
}

// Parse parses a single BigQuery CREATE routine or view statement.
func Parse(sql string, opts Options) (*ParseResult, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil, &ParseError{Message: "empty SQL", Line: 1, Column: 1}
	}

	l := newLexer(sql)
	if l.err != nil {
		return nil, l.err
	}
	p := &parser{lex: l, input: sql}
	res, err := p.parseStatement()
	if err != nil {
		if pe, ok := err.(*ParseError); ok && pe == nil {
			return nil, &ParseError{Message: "internal parse error (nil)", Line: 1, Column: 1}
		}
		return nil, err
	}
	if res == nil {
		return nil, &ParseError{Message: "internal parse error (nil result)", Line: 1, Column: 1}
	}

	if res.IsTemporary {
		return nil, &ParseError{
			Message: "TEMPORARY/TEMP routines are not supported for persistent BigQuery objects managed by Terraform",
			Line:    1,
			Column:  1,
		}
	}

	body := res.DefinitionBody
	if res.Query != "" {
		body = res.Query
	}
	if opts.TrimComments {
		body = TrimComments(body)
	}
	if opts.TrimBody {
		body = TrimBody(body)
	}
	if res.Query != "" {
		res.Query = body
	} else {
		res.DefinitionBody = body
	}

	return res, nil
}

// ParseRoutine parses a CREATE routine statement.
func ParseRoutine(sql string, opts Options) (*ParseResult, error) {
	res, err := Parse(sql, opts)
	if err != nil {
		return nil, err
	}
	switch res.Kind {
	case KindScalarFunction, KindTableFunction, KindProcedure, KindAggregateFunction:
		return res, nil
	default:
		return nil, &ParseError{Message: fmt.Sprintf("expected CREATE FUNCTION/PROCEDURE, got %s", res.Kind)}
	}
}

// ParseView parses a CREATE VIEW statement.
func ParseView(sql string, opts Options) (*ParseResult, error) {
	res, err := Parse(sql, opts)
	if err != nil {
		return nil, err
	}
	switch res.Kind {
	case KindView, KindMaterializedView:
		return res, nil
	default:
		return nil, &ParseError{Message: fmt.Sprintf("expected CREATE VIEW, got %s", res.Kind)}
	}
}

type parser struct {
	lex   *lexer
	input string
}

func (p *parser) peek() token { return p.lex.peek() }
func (p *parser) next() token { return p.lex.next() }

func (p *parser) expect(kind int, what string) (token, error) {
	t := p.next()
	if t.kind != kind {
		return t, &ParseError{
			Message: fmt.Sprintf("expected %s, got %q", what, t.lit),
			Line:    t.line,
			Column:  t.col,
			Offset:  t.offset,
		}
	}
	return t, nil
}

func (p *parser) parseStatement() (*ParseResult, error) {
	t := p.next()
	if t.kind != tokCreate {
		return nil, &ParseError{Message: "expected CREATE", Line: t.line, Column: t.col, Offset: t.offset}
	}

	res := &ParseResult{Labels: map[string]string{}}

	// OR REPLACE
	if p.peek().kind == tokOr {
		p.next()
		if _, err := p.expect(tokReplace, "REPLACE"); err != nil {
			return nil, err
		}
	}

	// TEMPORARY | TEMP
	switch p.peek().kind {
	case tokTemporary, tokTemp:
		p.next()
		res.IsTemporary = true
	}

	// MATERIALIZED VIEW | VIEW | TABLE FUNCTION | AGGREGATE FUNCTION | FUNCTION | PROCEDURE
	switch p.peek().kind {
	case tokMaterialized:
		p.next()
		if _, err := p.expect(tokView, "VIEW"); err != nil {
			return nil, err
		}
		res.Kind = KindMaterializedView
		res.IsMaterialized = true
		return p.parseViewRest(res)
	case tokView:
		p.next()
		res.Kind = KindView
		return p.parseViewRest(res)
	case tokTable:
		p.next()
		if _, err := p.expect(tokFunction, "FUNCTION"); err != nil {
			return nil, err
		}
		res.Kind = KindTableFunction
		res.Language = "SQL"
		return p.parseRoutineRest(res)
	case tokAggregate:
		p.next()
		if _, err := p.expect(tokFunction, "FUNCTION"); err != nil {
			return nil, err
		}
		res.Kind = KindAggregateFunction
		res.Language = "SQL"
		return p.parseRoutineRest(res)
	case tokFunction:
		p.next()
		res.Kind = KindScalarFunction
		res.Language = "SQL"
		return p.parseRoutineRest(res)
	case tokProcedure:
		p.next()
		res.Kind = KindProcedure
		res.Language = "SQL"
		return p.parseRoutineRest(res)
	default:
		t := p.peek()
		return nil, &ParseError{
			Message: fmt.Sprintf("expected FUNCTION, TABLE FUNCTION, PROCEDURE, VIEW, or MATERIALIZED VIEW, got %q", t.lit),
			Line:    t.line, Column: t.col, Offset: t.offset,
		}
	}
}

func (p *parser) parseIfNotExists() error {
	if p.peek().kind != tokIf {
		return nil
	}
	p.next()
	if _, err := p.expect(tokNot, "NOT"); err != nil {
		return err
	}
	if _, err := p.expect(tokExists, "EXISTS"); err != nil {
		return err
	}
	return nil
}

func (p *parser) parseQualifiedName() (string, error) {
	var parts []string
	for {
		t := p.peek()
		if t.kind != tokIdent && !isKeywordToken(t.kind) {
			if len(parts) == 0 {
				return "", &ParseError{Message: "expected identifier", Line: t.line, Column: t.col, Offset: t.offset}
			}
			break
		}
		t = p.next()
		parts = append(parts, t.lit)
		if p.peek().kind != tokDot {
			break
		}
		p.next()
	}
	return strings.Join(parts, "."), nil
}

func isKeywordToken(kind int) bool {
	switch kind {
	case tokCreate, tokOr, tokReplace, tokTemporary, tokTemp, tokFunction, tokTable, tokProcedure,
		tokAggregate, tokView, tokMaterialized, tokIf, tokNot, tokExists, tokReturns, tokLanguage,
		tokOptions, tokAs, tokRemote, tokWith, tokConnection, tokPartition, tokBy, tokCluster,
		tokAny, tokType, tokIn, tokOut, tokInout, tokTrue, tokFalse, tokBegin, tokEnd:
		return true
	}
	return false
}

func (p *parser) parseRoutineRest(res *ParseResult) (*ParseResult, error) {
	if err := p.parseIfNotExists(); err != nil {
		return nil, err
	}
	name, err := p.parseQualifiedName()
	if err != nil {
		return nil, err
	}
	res.Project, res.DatasetID, res.ObjectID = SplitQualifiedName(name)

	// optional argument list
	if p.peek().kind == tokLParen {
		args, err := p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		if res.Kind == KindAggregateFunction {
			for i := range args {
				if args[i].IsAggregate == nil {
					agg := true
					args[i].IsAggregate = &agg
				}
			}
		}
		res.Arguments = args
	}

	// RETURNS ...
	if p.peek().kind == tokReturns {
		p.next()
		if p.peek().kind == tokTable || (p.peek().kind == tokIdent && strings.EqualFold(p.peek().lit, "TABLE")) {
			p.next()
			typeStr, err := p.parseTypeString()
			if err != nil {
				return nil, err
			}
			if !strings.HasPrefix(strings.ToUpper(typeStr), "TABLE") {
				typeStr = "TABLE" + typeStr
			}
			if strings.Contains(typeStr, "<") {
				inner := extractAngleContents(typeStr)
				fields, err := parseStructFields(inner)
				if err != nil {
					return nil, err
				}
				var cols []ColumnDef
				for _, f := range fields {
					cols = append(cols, ColumnDef{Name: f.Name})
				}
				js, err := tableTypeToJSON(cols)
				if err != nil {
					return nil, err
				}
				res.ReturnTableTypeJSON = js
			}
		} else {
			typeStr, err := p.parseTypeString()
			if err != nil {
				return nil, err
			}
			js, err := sqlTypeToJSON(typeStr)
			if err != nil {
				return nil, &ParseError{Message: err.Error(), Line: p.peek().line, Column: p.peek().col}
			}
			res.ReturnTypeJSON = js
		}
	}

	// LANGUAGE
	if p.peek().kind == tokLanguage {
		p.next()
		t := p.next()
		lang := strings.ToUpper(t.lit)
		switch lang {
		case "JS", "JAVASCRIPT":
			res.Language = "JAVASCRIPT"
		case "SQL", "PYTHON", "JAVA", "SCALA":
			res.Language = lang
		default:
			res.Language = lang
		}
	}

	// REMOTE WITH CONNECTION ...
	if p.peek().kind == tokRemote {
		p.next()
		if _, err := p.expect(tokWith, "WITH"); err != nil {
			return nil, err
		}
		if _, err := p.expect(tokConnection, "CONNECTION"); err != nil {
			return nil, err
		}
		conn, err := p.parseQualifiedName()
		if err != nil {
			return nil, err
		}
		res.RemoteConnection = conn
	}

	// OPTIONS (...)
	if p.peek().kind == tokOptions {
		if err := p.parseOptions(res); err != nil {
			return nil, err
		}
	}

	// AS body
	if p.peek().kind != tokAs {
		t := p.peek()
		return nil, &ParseError{Message: "expected AS", Line: t.line, Column: t.col, Offset: t.offset}
	}
	asTok := p.next()
	body, _, cerr := captureBody(p.input, asTok.offset+len(asTok.lit))
	if cerr != nil {
		return nil, cerr
	}
	res.DefinitionBody = body
	if res.Language == "" {
		res.Language = "SQL"
	}

	if res.RemoteConnection != "" || res.RemoteEndpoint != "" {
		res.RemoteFunctionOptionsJSON = fmt.Sprintf(
			`{"connection":%q,"endpoint":%q}`,
			res.RemoteConnection, res.RemoteEndpoint,
		)
	}

	return res, nil
}

func (p *parser) parseViewRest(res *ParseResult) (*ParseResult, error) {
	if err := p.parseIfNotExists(); err != nil {
		return nil, err
	}
	name, err := p.parseQualifiedName()
	if err != nil {
		return nil, err
	}
	res.Project, res.DatasetID, res.ObjectID = SplitQualifiedName(name)

	// optional column list
	if p.peek().kind == tokLParen {
		cols, err := p.parseViewColumnList()
		if err != nil {
			return nil, err
		}
		res.Columns = cols
		if js, err := columnsToSchemaJSON(cols); err == nil {
			res.SchemaJSON = js
		}
	}

	// PARTITION BY ...
	if p.peek().kind == tokPartition {
		p.next()
		if _, err := p.expect(tokBy, "BY"); err != nil {
			return nil, err
		}
		if err := p.parsePartitionBy(res); err != nil {
			return nil, err
		}
	}

	// CLUSTER BY ...
	if p.peek().kind == tokCluster {
		p.next()
		if _, err := p.expect(tokBy, "BY"); err != nil {
			return nil, err
		}
		for {
			t := p.next()
			if t.kind != tokIdent && !isKeywordToken(t.kind) {
				return nil, &ParseError{Message: "expected cluster column", Line: t.line, Column: t.col}
			}
			res.Clustering = append(res.Clustering, t.lit)
			if p.peek().kind != tokComma {
				break
			}
			p.next()
		}
	}

	// OPTIONS
	if p.peek().kind == tokOptions {
		if err := p.parseOptions(res); err != nil {
			return nil, err
		}
	}

	if p.peek().kind != tokAs {
		t := p.peek()
		return nil, &ParseError{Message: "expected AS", Line: t.line, Column: t.col, Offset: t.offset}
	}
	asTok := p.next()
	body, _, cerr := captureBody(p.input, asTok.offset+len(asTok.lit))
	if cerr != nil {
		return nil, cerr
	}
	res.Query = body
	return res, nil
}

func (p *parser) parsePartitionBy(res *ParseResult) error {
	// DATE(field) | DATETIME_TRUNC(...) | field
	t := p.peek()
	if t.kind == tokIdent || isKeywordToken(t.kind) {
		name := strings.ToUpper(t.lit)
		p.next()
		if p.peek().kind == tokLParen {
			p.next()
			// first arg is field (possibly nested)
			fieldTok := p.next()
			res.PartitioningField = fieldTok.lit
			// skip to matching paren
			depth := 1
			for depth > 0 && p.peek().kind != tokEOF {
				n := p.next()
				if n.kind == tokLParen {
					depth++
				} else if n.kind == tokRParen {
					depth--
				}
			}
			switch name {
			case "DATE", "DATE_TRUNC":
				res.PartitioningType = "DAY"
			case "DATETIME_TRUNC", "TIMESTAMP_TRUNC":
				res.PartitioningType = "DAY"
			default:
				res.PartitioningType = "DAY"
			}
			return nil
		}
		res.PartitioningField = t.lit
		res.PartitioningType = "DAY"
		return nil
	}
	return &ParseError{Message: "expected PARTITION BY expression", Line: t.line, Column: t.col}
}

func (p *parser) parseViewColumnList() ([]ColumnDef, error) {
	p.next() // (
	var cols []ColumnDef
	for p.peek().kind != tokRParen && p.peek().kind != tokEOF {
		t := p.next()
		if t.kind != tokIdent && !isKeywordToken(t.kind) {
			return nil, &ParseError{Message: "expected column name", Line: t.line, Column: t.col}
		}
		col := ColumnDef{Name: t.lit}
		if p.peek().kind == tokOptions {
			p.next()
			opts, err := p.parseOptionsMap()
			if err != nil {
				return nil, err
			}
			if d, ok := opts["description"]; ok {
				col.Description = d
			}
		}
		cols = append(cols, col)
		if p.peek().kind == tokComma {
			p.next()
			continue
		}
		break
	}
	if _, err := p.expect(tokRParen, ")"); err != nil {
		return nil, err
	}
	return cols, nil
}

func (p *parser) parseArgumentList() ([]Argument, error) {
	p.next() // (
	var args []Argument
	for p.peek().kind != tokRParen && p.peek().kind != tokEOF {
		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().kind == tokComma {
			p.next()
			continue
		}
		break
	}
	if _, err := p.expect(tokRParen, ")"); err != nil {
		return nil, err
	}
	return args, nil
}

func (p *parser) parseArgument() (Argument, error) {
	var arg Argument
	// optional mode
	switch p.peek().kind {
	case tokIn, tokOut, tokInout:
		arg.Mode = strings.ToUpper(p.next().lit)
	}

	// name
	t := p.next()
	if t.kind != tokIdent && !isKeywordToken(t.kind) {
		return arg, &ParseError{Message: "expected argument name", Line: t.line, Column: t.col}
	}
	arg.Name = t.lit

	// ANY TYPE
	if p.peek().kind == tokAny {
		p.next()
		if _, err := p.expect(tokType, "TYPE"); err != nil {
			return arg, err
		}
		arg.ArgumentKind = "ANY_TYPE"
		if err := p.parseOptionalNotAggregate(&arg); err != nil {
			return arg, err
		}
		return arg, nil
	}

	// TABLE<...> or type
	typeStr, err := p.parseTypeString()
	if err != nil {
		return arg, err
	}
	upper := strings.ToUpper(strings.TrimSpace(typeStr))
	if strings.HasPrefix(upper, "TABLE") {
		inner := extractAngleContents(typeStr)
		var cols []ColumnDef
		if inner != "" {
			fields, err := parseStructFields(inner)
			if err != nil {
				return arg, err
			}
			for _, f := range fields {
				cols = append(cols, ColumnDef{Name: f.Name})
			}
		}
		js, err := tableArgTypeJSON(cols)
		if err != nil {
			return arg, err
		}
		arg.DataTypeJSON = js
		arg.ArgumentKind = "FIXED_TYPE"
		if err := p.parseOptionalNotAggregate(&arg); err != nil {
			return arg, err
		}
		return arg, nil
	}

	js, err := sqlTypeToJSON(typeStr)
	if err != nil {
		return arg, &ParseError{Message: err.Error()}
	}
	arg.DataTypeJSON = js
	arg.ArgumentKind = "FIXED_TYPE"
	if err := p.parseOptionalNotAggregate(&arg); err != nil {
		return arg, err
	}
	return arg, nil
}

// parseOptionalNotAggregate consumes a trailing "NOT AGGREGATE" clause on a UDAF parameter.
func (p *parser) parseOptionalNotAggregate(arg *Argument) error {
	if p.peek().kind != tokNot {
		return nil
	}
	p.next()
	if _, err := p.expect(tokAggregate, "AGGREGATE"); err != nil {
		return err
	}
	agg := false
	arg.IsAggregate = &agg
	return nil
}

// parseTypeString reads a type expression as raw text from tokens.
func (p *parser) parseTypeString() (string, error) {
	t := p.peek()
	if t.kind == tokEOF {
		return "", &ParseError{Message: "expected type", Line: t.line, Column: t.col}
	}
	var b strings.Builder
	depthAngle := 0
	depthParen := 0
	first := true
	for {
		t := p.peek()
		if t.kind == tokEOF {
			break
		}
		if first {
			first = false
		} else {
			// stop at top-level delimiters
			if depthAngle == 0 && depthParen == 0 {
				switch t.kind {
				case tokComma, tokRParen, tokLanguage, tokOptions, tokAs, tokReturns,
					tokRemote, tokSemi:
					return strings.TrimSpace(b.String()), nil
				}
				// also stop before ident that starts OPTIONS etc already handled
				if t.kind == tokIdent || isKeywordToken(t.kind) {
					// could be next arg name only if we already have type and no open angle —
					// but ARRAY<STRING> finished when angle closes. After type, next could be LANGUAGE.
					u := strings.ToUpper(t.lit)
					if u == "LANGUAGE" || u == "OPTIONS" || u == "AS" || u == "RETURNS" || u == "REMOTE" ||
						u == "DETERMINISTIC" || u == "NOT" {
						return strings.TrimSpace(b.String()), nil
					}
					// After complete simple type, next ident means end (next argument name) when in arg list —
					// but we can't know. Use: if we have content and next is ident and no open brackets, stop
					// UNLESS current type ended mid way... For "STRING" next is "," or ")".
					if b.Len() > 0 && t.kind == tokIdent {
						return strings.TrimSpace(b.String()), nil
					}
				}
			}
		}

		t = p.next()
		switch t.kind {
		case tokLAngle:
			depthAngle++
			b.WriteString("<")
		case tokRAngle:
			depthAngle--
			b.WriteString(">")
			if depthAngle == 0 && depthParen == 0 {
				return strings.TrimSpace(b.String()), nil
			}
		case tokLParen:
			depthParen++
			b.WriteString("(")
		case tokRParen:
			if depthParen == 0 {
				// end of arg list — put back
				p.lex.backup()
				return strings.TrimSpace(b.String()), nil
			}
			depthParen--
			b.WriteString(")")
		case tokComma:
			if depthAngle == 0 && depthParen == 0 {
				p.lex.backup()
				return strings.TrimSpace(b.String()), nil
			}
			b.WriteString(",")
		case tokDot:
			b.WriteString(".")
		case tokStar:
			b.WriteString("*")
		default:
			if b.Len() > 0 {
				last := b.String()[b.Len()-1]
				if last != '<' && last != '(' && last != ',' {
					b.WriteByte(' ')
				}
			}
			b.WriteString(t.lit)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		return "", &ParseError{Message: "expected type", Line: t.line, Column: t.col}
	}
	return s, nil
}

func (p *parser) parseOptions(res *ParseResult) error {
	p.next() // OPTIONS
	m, err := p.parseOptionsMap()
	if err != nil {
		return err
	}
	for k, v := range m {
		key := strings.ToLower(k)
		switch key {
		case "description":
			res.Description = v
		case "friendly_name":
			res.FriendlyName = v
		case "library", "libraries":
			// single or already joined
			res.ImportedLibraries = append(res.ImportedLibraries, v)
		case "determinism_level":
			res.DeterminismLevel = strings.ToUpper(v)
		case "data_governance_type":
			res.DataGovernanceType = strings.ToUpper(v)
		case "endpoint":
			res.RemoteEndpoint = v
		case "enable_refresh":
			b := strings.EqualFold(v, "true") || v == "TRUE"
			res.EnableRefresh = &b
		case "allow_non_incremental_definition":
			b := strings.EqualFold(v, "true") || v == "TRUE"
			res.AllowNonIncrementalDefinition = &b
		case "refresh_interval_minutes":
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				ms := int64(f * 60000)
				res.RefreshIntervalMs = &ms
			}
		case "max_staleness":
			res.MaxStaleness = v
		case "kms_key_name":
			res.KmsKeyName = v
		case "labels":
			// parse [("k","v"), ...]
			res.Labels = parseLabelsLiteral(v)
		default:
			// ignore unmappable e.g. retain_partitions
		}
	}
	return nil
}

func (p *parser) parseOptionsMap() (map[string]string, error) {
	if _, err := p.expect(tokLParen, "("); err != nil {
		return nil, err
	}
	m := map[string]string{}
	for p.peek().kind != tokRParen && p.peek().kind != tokEOF {
		keyTok := p.next()
		if keyTok.kind != tokIdent && !isKeywordToken(keyTok.kind) {
			return nil, &ParseError{Message: "expected option name", Line: keyTok.line, Column: keyTok.col}
		}
		key := keyTok.lit
		if p.peek().kind == tokEq {
			p.next()
		}
		val, err := p.parseOptionValue()
		if err != nil {
			return nil, err
		}
		m[key] = val
		if p.peek().kind == tokComma {
			p.next()
			continue
		}
		break
	}
	if _, err := p.expect(tokRParen, ")"); err != nil {
		return nil, err
	}
	return m, nil
}

func (p *parser) parseOptionValue() (string, error) {
	t := p.peek()
	switch t.kind {
	case tokString, tokRawString, tokNumber, tokTrue, tokFalse:
		p.next()
		return t.lit, nil
	case tokLBracket:
		p.next()
		start := t.offset
		depth := 1
		for depth > 0 && p.peek().kind != tokEOF {
			n := p.next()
			if n.kind == tokLBracket {
				depth++
			} else if n.kind == tokRBracket {
				depth--
			}
		}
		_ = start
		return extractBracketValue(p.input, start), nil
	case tokIdent:
		// INTERVAL ... or bare identifier / TRUE-like
		if strings.EqualFold(t.lit, "INTERVAL") {
			return p.parseUntilOptionSep()
		}
		p.next()
		return t.lit, nil
	default:
		if isKeywordToken(t.kind) {
			return p.parseUntilOptionSep()
		}
		return p.parseUntilOptionSep()
	}
}

func (p *parser) parseUntilOptionSep() (string, error) {
	var b strings.Builder
	depth := 0
	for {
		t := p.peek()
		if t.kind == tokEOF {
			break
		}
		if depth == 0 && (t.kind == tokComma || t.kind == tokRParen) {
			break
		}
		t = p.next()
		if t.kind == tokLParen || t.kind == tokLBracket || t.kind == tokLAngle {
			depth++
		}
		if t.kind == tokRParen || t.kind == tokRBracket || t.kind == tokRAngle {
			if depth > 0 {
				depth--
			}
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		// preserve quoted strings with quotes for INTERVAL
		if t.kind == tokString {
			b.WriteByte('"')
			b.WriteString(t.lit)
			b.WriteByte('"')
		} else {
			b.WriteString(t.lit)
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func extractBracketValue(input string, startOffset int) string {
	i := startOffset
	for i < len(input) && input[i] != '[' {
		i++
	}
	if i >= len(input) {
		return ""
	}
	start := i
	depth := 0
	for i < len(input) {
		if input[i] == '[' {
			depth++
		} else if input[i] == ']' {
			depth--
			if depth == 0 {
				return input[start : i+1]
			}
		}
		i++
	}
	return input[start:]
}

func parseLabelsLiteral(v string) map[string]string {
	// [("org_unit", "development")]
	m := map[string]string{}
	v = strings.TrimSpace(v)
	inner := strings.TrimPrefix(v, "[")
	inner = strings.TrimSuffix(inner, "]")
	parts := splitTopLevel(inner, ',')
	// Actually pairs are ("k","v") — split by ), (
	var cur strings.Builder
	depth := 0
	var pairs []string
	for _, r := range inner {
		if r == '(' {
			depth++
		}
		if r == ')' {
			depth--
		}
		if r == ',' && depth == 0 {
			pairs = append(pairs, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		pairs = append(pairs, cur.String())
	}
	_ = parts
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		pair = strings.TrimPrefix(pair, "(")
		pair = strings.TrimSuffix(pair, ")")
		kv := splitTopLevel(pair, ',')
		if len(kv) >= 2 {
			k := strings.Trim(kv[0], " \t\"'")
			val := strings.Trim(kv[1], " \t\"'")
			m[k] = val
		}
	}
	return m
}
