package sqlparse

import (
	"sort"
	"strings"
	"unicode"
)

// QualifyOptions controls body reference collection and optional project rewriting.
type QualifyOptions struct {
	// TargetProject is prepended to two-part refs when Rewrite is true.
	// Typically the optional target_project argument, else the CREATE-name project.
	TargetProject string
	// HomeDataset is the CREATE object's dataset; only foreign datasets are listed
	// and rewritten. Empty HomeDataset treats every body dataset as foreign.
	HomeDataset string
	// HomeObject is the CREATE object's id (routine_id / table_id). Used to exclude
	// self-references from References. Empty skips self filtering.
	HomeObject string
	// Rewrite enables project qualification and ${project} placeholder replacement.
	Rewrite bool
}

// ObjectReference is a classified dataset.object dependency found in a SQL body.
type ObjectReference struct {
	DatasetID    string
	ObjectID     string
	ObjectType   string // SCALAR_FUNCTION | TABLE_VALUED_FUNCTION | PROCEDURE | VIEW | TABLE
	ResourceType string // ROUTINE | VIEW
}

// QualifyResult is the outcome of QualifyBody.
type QualifyResult struct {
	Body              string
	DatasetReferences []string          // sorted, unique, excluding HomeDataset; empty if none foreign
	References        []ObjectReference // sorted unique by (dataset_id, object_id); excludes self
}

// bodyRef is one detected project.dataset.object or dataset.object span.
type bodyRef struct {
	start, end int
	project    string // empty for two-part
	dataset    string
	object     string
	parts      int
	backticked bool // original used backticks on at least one segment
	joinedBT   bool // single `a.b` / `a.b.c` form
	infoSchema bool // dataset.INFORMATION_SCHEMA.object (rewrite still two-part)
}

// expectKind tracks why the scanner expects a relation/name next.
type expectKind int

const (
	expectNone        expectKind = iota
	expectFromJoin               // FROM / JOIN — table or TVF
	expectTableClause            // INTO / UPDATE / MERGE / DELETE / TRUNCATE / USING
	expectCall                   // CALL — procedure
)

const projectPlaceholder = "${project}"

// QualifyBody scans a routine/view body for dataset-qualified entity references,
// optionally rewriting two-part refs with TargetProject.
func QualifyBody(body string, opts QualifyOptions) QualifyResult {
	out := body
	if opts.Rewrite && opts.TargetProject != "" {
		out = strings.ReplaceAll(out, projectPlaceholder, opts.TargetProject)
	}

	scan := findBodyRefs(out)
	foreignSet := map[string]struct{}{}
	for _, r := range scan.refs {
		if r.dataset == "" {
			continue
		}
		if opts.HomeDataset != "" && strings.EqualFold(r.dataset, opts.HomeDataset) {
			continue
		}
		foreignSet[r.dataset] = struct{}{}
	}

	foreign := make([]string, 0, len(foreignSet))
	for ds := range foreignSet {
		foreign = append(foreign, ds)
	}
	sort.Strings(foreign)

	if opts.Rewrite && opts.TargetProject != "" && len(foreign) > 0 {
		out = rewriteTwoPartRefs(out, scan.refs, opts)
	}

	if foreign == nil {
		foreign = []string{}
	}
	return QualifyResult{
		Body:              out,
		DatasetReferences: foreign,
		References:        filterObjectReferences(scan.objectRefs, opts),
	}
}

func filterObjectReferences(in []ObjectReference, opts QualifyOptions) []ObjectReference {
	seen := map[string]struct{}{}
	out := make([]ObjectReference, 0, len(in))
	for _, r := range in {
		if r.DatasetID == "" || r.ObjectID == "" {
			continue
		}
		if opts.HomeDataset != "" && opts.HomeObject != "" &&
			r.DatasetID == opts.HomeDataset && r.ObjectID == opts.HomeObject {
			continue
		}
		key := r.DatasetID + "\x00" + r.ObjectID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DatasetID != out[j].DatasetID {
			return out[i].DatasetID < out[j].DatasetID
		}
		return out[i].ObjectID < out[j].ObjectID
	})
	if out == nil {
		out = []ObjectReference{}
	}
	return out
}

func classifyObjectRef(r bodyRef, kind expectKind, isParen bool) ObjectReference {
	ref := ObjectReference{
		DatasetID: r.dataset,
		ObjectID:  r.object,
	}
	switch {
	case kind == expectCall:
		ref.ObjectType = "PROCEDURE"
		ref.ResourceType = "ROUTINE"
	case kind == expectFromJoin && isParen:
		ref.ObjectType = "TABLE_VALUED_FUNCTION"
		ref.ResourceType = "ROUTINE"
	case (kind == expectFromJoin || kind == expectTableClause) && !isParen:
		if r.infoSchema || strings.Contains(strings.ToUpper(r.object), "INFORMATION_SCHEMA") {
			ref.ObjectType = "TABLE"
		} else {
			ref.ObjectType = "VIEW"
		}
		ref.ResourceType = "VIEW"
	case isParen:
		ref.ObjectType = "SCALAR_FUNCTION"
		ref.ResourceType = "ROUTINE"
	default:
		// Should not happen for recorded refs; treat as VIEW relation.
		ref.ObjectType = "VIEW"
		ref.ResourceType = "VIEW"
	}
	return ref
}

func rewriteTwoPartRefs(body string, refs []bodyRef, opts QualifyOptions) string {
	// Rewrite from the end so offsets stay valid.
	var toRewrite []bodyRef
	for _, r := range refs {
		if r.parts != 2 || r.project != "" {
			continue
		}
		if opts.HomeDataset != "" && strings.EqualFold(r.dataset, opts.HomeDataset) {
			continue
		}
		toRewrite = append(toRewrite, r)
	}
	sort.Slice(toRewrite, func(i, j int) bool {
		return toRewrite[i].start > toRewrite[j].start
	})

	out := body
	for _, r := range toRewrite {
		replacement := formatQualifiedRef(opts.TargetProject, r)
		out = out[:r.start] + replacement + out[r.end:]
	}
	return out
}

func formatQualifiedRef(project string, r bodyRef) string {
	// Always backtick-quote each segment. Unquoted project IDs with hyphens
	// (e.g. my-project-123) are invalid SQL for the Routines API.
	if r.infoSchema {
		// dataset.INFORMATION_SCHEMA.object → `project`.`dataset`.INFORMATION_SCHEMA.object
		return "`" + project + "`." + "`" + r.dataset + "`." + r.object
	}
	return "`" + project + "`." + "`" + r.dataset + "`." + "`" + r.object + "`"
}

type bodyScanResult struct {
	refs       []bodyRef
	objectRefs []ObjectReference
}

func findBodyRefs(s string) bodyScanResult {
	var refs []bodyRef
	var objectRefs []ObjectReference
	expect := expectNone
	pendingExtract := false
	parenDepth := 0
	var extractDepths []int // paren depths of open EXTRACT( ... ) args
	i := 0
	for i < len(s) {
		// Whitespace
		if isSpaceByte(s[i]) {
			i++
			continue
		}
		// Comments
		if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
			i = consumeLineComment(s, i)
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i = consumeBlockComment(s, i)
			continue
		}
		// String literals
		if s[i] == '\'' || s[i] == '"' {
			pendingExtract = false
			i = consumeQuoted(s, i, nil)
			continue
		}

		// Backtick-qualified name
		if s[i] == '`' {
			pendingExtract = false
			ref, next, ok := tryParseDottedRef(s, i)
			if ok {
				i = consumeRefOrSkip(s, ref, next, &expect, &refs, &objectRefs)
				continue
			}
			i++
			continue
		}

		// Bare ident: keyword, dotted ref / call, or single-ident table
		if isIdentStartByte(s[i]) {
			kw, next := readBareIdent(s, i)
			upper := strings.ToUpper(kw)
			j := skipSpacesAndComments(s, next)

			// Dotted name or project.dataset.object starting here
			if j < len(s) && s[j] == '.' {
				pendingExtract = false
				ref, end, ok := tryParseDottedRef(s, i)
				if ok {
					i = consumeRefOrSkip(s, ref, end, &expect, &refs, &objectRefs)
					continue
				}
			}

			switch upper {
			case "EXTRACT":
				// EXTRACT(part FROM expr) — FROM here is not a table clause.
				pendingExtract = true
				i = next
				continue
			case "FROM":
				if inExtractArgs(extractDepths, parenDepth) {
					i = next
					continue
				}
				pendingExtract = false
				expect = expectFromJoin
				i = next
				continue
			case "JOIN":
				pendingExtract = false
				expect = expectFromJoin
				i = next
				continue
			case "INTO", "UPDATE", "USING", "MERGE", "DELETE", "TRUNCATE":
				pendingExtract = false
				expect = expectTableClause
				i = next
				continue
			case "CALL":
				pendingExtract = false
				expect = expectCall
				i = next
				continue
			case "TABLE":
				// Keep expect from TRUNCATE TABLE ...
				pendingExtract = false
				i = next
				continue
			case "UNNEST":
				pendingExtract = false
				expect = expectNone
				i = next
				continue
			case "WITH":
				pendingExtract = false
				expect = expectNone
				i = next
				continue
			case "ON", "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING", "SET", "VALUES":
				pendingExtract = false
				expect = expectNone
				i = next
				continue
			case "LEFT", "RIGHT", "FULL", "INNER", "CROSS", "OUTER", "AS", "BY",
				"SELECT", "DEFAULT", "DECLARE", "BEGIN", "END", "IF", "THEN", "ELSE",
				"ELSEIF", "LOOP", "WHILE", "DO", "FOR", "IN", "WHEN", "AND", "OR", "NOT",
				"UNION", "INTERSECT", "EXCEPT", "QUALIFY", "WINDOW", "TRUE", "FALSE",
				"NULL", "CASE", "MATCHED", "INSERT":
				pendingExtract = false
				i = next
				continue
			}

			pendingExtract = false
			// Single-ident relation (CTE / unqualified table) in table-ref context
			if expect != expectNone {
				i = next
				i = skipOptionalAlias(s, i)
				if i < len(s) && s[i] == ',' {
					i++
					// keep expect
				} else {
					expect = expectNone
				}
				continue
			}
			i = next
			continue
		}

		// Punctuation
		if s[i] == ',' && expect != expectNone {
			i++
			continue
		}
		if s[i] == '(' {
			parenDepth++
			if pendingExtract {
				extractDepths = append(extractDepths, parenDepth)
				pendingExtract = false
			}
			expect = expectNone
			i++
			continue
		}
		if s[i] == ')' {
			if n := len(extractDepths); n > 0 && extractDepths[n-1] == parenDepth {
				extractDepths = extractDepths[:n-1]
			}
			if parenDepth > 0 {
				parenDepth--
			}
			expect = expectNone
			pendingExtract = false
			i++
			continue
		}
		if s[i] == ';' {
			expect = expectNone
			pendingExtract = false
		}
		i++
	}
	return bodyScanResult{refs: refs, objectRefs: objectRefs}
}

func inExtractArgs(extractDepths []int, parenDepth int) bool {
	for _, d := range extractDepths {
		if parenDepth >= d {
			return true
		}
	}
	return false
}

// consumeRefOrSkip records a multi-part ref when in table context or a call, then advances.
func consumeRefOrSkip(s string, ref bodyRef, next int, expect *expectKind, refs *[]bodyRef, objectRefs *[]ObjectReference) int {
	isParen := next < len(s) && s[next] == '('
	if (*expect != expectNone || isParen) && ref.parts >= 2 {
		*refs = append(*refs, ref)
		*objectRefs = append(*objectRefs, classifyObjectRef(ref, *expect, isParen))
		i := next
		if *expect != expectNone {
			i = skipOptionalAlias(s, i)
			if i < len(s) && s[i] == ',' {
				i++
				// keep expect
			} else {
				*expect = expectNone
			}
		}
		return i
	}
	if *expect != expectNone && ref.parts == 1 {
		i := next
		i = skipOptionalAlias(s, i)
		if i < len(s) && s[i] == ',' {
			i++
		} else {
			*expect = expectNone
		}
		return i
	}
	return next
}

// tryParseDottedRef parses ident(.ident)* possibly with backticks, starting at i.
func tryParseDottedRef(s string, i int) (bodyRef, int, bool) {
	start := i
	var parts []string
	joinedBT := false
	anyBT := false

	// Single backtick containing dots: `a.b` or `a.b.c`
	if s[i] == '`' {
		end := i + 1
		for end < len(s) && s[end] != '`' {
			end++
		}
		if end >= len(s) {
			return bodyRef{}, i, false
		}
		inner := s[i+1 : end]
		if strings.Contains(inner, ".") {
			segs := strings.Split(inner, ".")
			if len(segs) < 1 || len(segs) > 3 {
				return bodyRef{}, i, false
			}
			for _, seg := range segs {
				if seg == "" || !isIdentString(seg) {
					return bodyRef{}, i, false
				}
			}
			ref := makeRef(start, end+1, segs, true, true)
			return ref, end + 1, true
		}
		// Single backtick ident; may continue with .`b` or .b
		parts = append(parts, inner)
		anyBT = true
		i = end + 1
	} else {
		id, next := readBareIdent(s, i)
		if id == "" {
			return bodyRef{}, i, false
		}
		parts = append(parts, id)
		i = next
	}

	for {
		j := skipSpacesAndComments(s, i)
		if j >= len(s) || s[j] != '.' {
			break
		}
		j++
		j = skipSpacesAndComments(s, j)
		if j >= len(s) {
			break
		}
		if s[j] == '`' {
			end := j + 1
			for end < len(s) && s[end] != '`' {
				end++
			}
			if end >= len(s) {
				break
			}
			inner := s[j+1 : end]
			if !isIdentString(inner) {
				break
			}
			parts = append(parts, inner)
			anyBT = true
			i = end + 1
			continue
		}
		if !isIdentStartByte(s[j]) {
			break
		}
		id, next := readBareIdent(s, j)
		if id == "" {
			break
		}
		parts = append(parts, id)
		i = next
	}

	if len(parts) == 0 {
		return bodyRef{}, start, false
	}
	// dataset.INFORMATION_SCHEMA.object or project.dataset.INFORMATION_SCHEMA.object
	if len(parts) == 4 && strings.EqualFold(parts[2], "INFORMATION_SCHEMA") {
		ref := bodyRef{
			start: start, end: i, parts: 3,
			project: parts[0], dataset: parts[1],
			object:     parts[2] + "." + parts[3],
			backticked: anyBT, joinedBT: joinedBT,
		}
		return ref, i, true
	}
	if len(parts) == 3 && strings.EqualFold(parts[1], "INFORMATION_SCHEMA") {
		ref := bodyRef{
			start: start, end: i, parts: 2, // needs project prefix
			dataset:    parts[0],
			object:     parts[1] + "." + parts[2],
			backticked: anyBT, joinedBT: joinedBT, infoSchema: true,
		}
		return ref, i, true
	}
	if len(parts) > 3 {
		return bodyRef{}, start, false
	}
	ref := makeRef(start, i, parts, anyBT, joinedBT)
	return ref, i, true
}

func makeRef(start, end int, parts []string, anyBT, joinedBT bool) bodyRef {
	r := bodyRef{start: start, end: end, parts: len(parts), backticked: anyBT, joinedBT: joinedBT}
	switch len(parts) {
	case 1:
		r.object = parts[0]
	case 2:
		r.dataset = parts[0]
		r.object = parts[1]
	case 3:
		r.project = parts[0]
		r.dataset = parts[1]
		r.object = parts[2]
	}
	return r
}

func skipOptionalAlias(s string, i int) int {
	i = skipSpacesAndComments(s, i)
	if i >= len(s) {
		return i
	}
	// AS alias
	if isIdentStartByte(s[i]) || s[i] == '`' {
		kw, next := readIdentOrBacktick(s, i)
		if strings.EqualFold(kw, "AS") {
			i = skipSpacesAndComments(s, next)
			if i >= len(s) {
				return i
			}
			_, i = readIdentOrBacktick(s, i)
			return i
		}
		// Implicit alias (not a clause keyword)
		upper := strings.ToUpper(kw)
		switch upper {
		case "WHERE", "JOIN", "LEFT", "RIGHT", "FULL", "INNER", "CROSS", "OUTER",
			"GROUP", "ORDER", "LIMIT", "HAVING", "ON", "SET", "VALUES", "UNION",
			"INTERSECT", "EXCEPT", "QUALIFY", "WINDOW", "BEGIN", "END", "LOOP",
			"WHILE", "FOR", "IF", "WHEN", "ELSE", "THEN", "USING", "MERGE", "CALL":
			return i
		default:
			return next
		}
	}
	return i
}

func readIdentOrBacktick(s string, i int) (string, int) {
	if i >= len(s) {
		return "", i
	}
	if s[i] == '`' {
		end := i + 1
		for end < len(s) && s[end] != '`' {
			end++
		}
		if end >= len(s) {
			return "", i
		}
		return s[i+1 : end], end + 1
	}
	return readBareIdent(s, i)
}

func readBareIdent(s string, i int) (string, int) {
	if i >= len(s) || !isIdentStartByte(s[i]) {
		return "", i
	}
	j := i + 1
	for j < len(s) && isIdentPartByte(s[j]) {
		j++
	}
	return s[i:j], j
}

func skipSpacesAndComments(s string, i int) int {
	for i < len(s) {
		if isSpaceByte(s[i]) {
			i++
			continue
		}
		if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
			i = consumeLineComment(s, i)
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i = consumeBlockComment(s, i)
			continue
		}
		break
	}
	return i
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isIdentStartByte(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isIdentPartByte(b byte) bool {
	return isIdentStartByte(b) || (b >= '0' && b <= '9') || b == '-' // allow hyphen in project ids when bare? BQ projects often have hyphens
}

func isIdentString(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				// allow digit-start only inside backticks for odd names; keep strict for dotted segments
				if !unicode.IsLetter(r) && r != '_' {
					return false
				}
			}
			continue
		}
		if r != '_' && r != '-' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
