package sqlparse

import (
	"strings"
	"testing"
)

func TestTrimIndentation(t *testing.T) {
	in := "\n      SELECT\n        a,\n        b\n      FROM t\n      WHERE x\n        AND y\n"
	got := TrimIndentation(in)
	want := "\nSELECT\n  a,\n  b\nFROM t\nWHERE x\n  AND y\n"
	if got != want {
		t.Fatalf("got=%q\nwant=%q", got, want)
	}
}

func TestTrimIndentation_noCommonIndent(t *testing.T) {
	in := "SELECT\n  a\nFROM t"
	if got := TrimIndentation(in); got != in {
		t.Fatalf("expected unchanged, got=%q", got)
	}
}

func TestTrimIndentation_tabs(t *testing.T) {
	in := "\tSELECT\n\t\ta\n\tFROM t"
	got := TrimIndentation(in)
	want := "SELECT\n\ta\nFROM t"
	if got != want {
		t.Fatalf("got=%q\nwant=%q", got, want)
	}
}

func TestParseRoutine_trimIndentation(t *testing.T) {
	sql := `
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'desc'
    ) AS (
      SELECT
        t.table_schema,
        t.table_name
      FROM x AS t
      WHERE t.partition_id != '__NULL__'
        AND t.table_name LIKE CONCAT('%', table_name_filter, '%')
    );`

	res, err := ParseRoutine(sql, Options{TrimBody: true, TrimIndentation: true})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(res.DefinitionBody, "\n")
	want := []string{
		"SELECT",
		"  t.table_schema,",
		"  t.table_name",
		"FROM x AS t",
		"WHERE t.partition_id != '__NULL__'",
		"  AND t.table_name LIKE CONCAT('%', table_name_filter, '%')",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines=%q", res.DefinitionBody)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: got=%q want=%q\nfull=%q", i+1, lines[i], want[i], res.DefinitionBody)
		}
	}
}

func TestParseView_trimIndentation(t *testing.T) {
	sql := `
    CREATE OR REPLACE VIEW mydataset.my_view AS
      SELECT
        a,
        b
      FROM t
      WHERE x = 1
    ;`

	res, err := ParseView(sql, Options{TrimBody: true, TrimIndentation: true})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(res.Query, "\n")
	want := []string{
		"SELECT",
		"  a,",
		"  b",
		"FROM t",
		"WHERE x = 1",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines=%q", res.Query)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: got=%q want=%q\nfull=%q", i+1, lines[i], want[i], res.Query)
		}
	}
}

func TestTrimBody(t *testing.T) {
	in := "\n  \n  SELECT 1\n  FROM t\n\n  \n"
	got := TrimBody(in)
	want := "SELECT 1\n  FROM t"
	if got != want {
		t.Fatalf("got=%q\nwant=%q", got, want)
	}
}

func TestTrimComments(t *testing.T) {
	in := `SELECT 1 -- trailing
/* block */
, 'keep -- inside' AS s
, "keep /* too */" AS t`
	got := TrimComments(in)
	if strings.Contains(got, "-- trailing") || strings.Contains(got, "block") {
		t.Fatalf("comments not removed: %q", got)
	}
	if !strings.Contains(got, "keep -- inside") || !strings.Contains(got, `keep /* too */`) {
		t.Fatalf("string literals altered: %q", got)
	}
	if !strings.Contains(got, "SELECT 1") {
		t.Fatalf("SQL removed: %q", got)
	}
}

const listPartitionsSQLWithComment = `
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'desc'
    ) AS (
      SELECT
        t.table_schema,
        t.table_name
      FROM x AS t
      WHERE t.partition_id != '__NULL__'  -- comment
        AND t.table_name LIKE CONCAT('%', table_name_filter, '%')
    );`

func TestParseRoutine_trimCommentsTrue(t *testing.T) {
	res, err := ParseRoutine(listPartitionsSQLWithComment, Options{TrimBody: true, TrimComments: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.DefinitionBody, "-- comment") {
		t.Fatalf("expected comment removed: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "SELECT") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseRoutine_trimCommentsFalse(t *testing.T) {
	res, err := ParseRoutine(listPartitionsSQLWithComment, Options{TrimBody: true, TrimComments: false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.DefinitionBody, "-- comment") {
		t.Fatalf("expected comment preserved: %q", res.DefinitionBody)
	}
}

func TestParseRoutine_trimBodyFalse(t *testing.T) {
	res, err := ParseRoutine(listPartitionsSQLWithComment, Options{TrimBody: false, TrimIndentation: false})
	if err != nil {
		t.Fatal(err)
	}
	body := res.DefinitionBody
	if strings.TrimSpace(body) == body {
		t.Fatalf("expected leading/trailing whitespace preserved when TrimBody=false: %q", body)
	}
	if !strings.Contains(body, "SELECT") {
		t.Fatalf("body=%q", body)
	}
}

func TestParseRoutine_trimBodyTrue(t *testing.T) {
	res, err := ParseRoutine(listPartitionsSQLWithComment, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	body := res.DefinitionBody
	if strings.HasPrefix(body, "\n") || strings.HasSuffix(body, "\n") {
		t.Fatalf("expected edges trimmed: %q", body)
	}
	lines := strings.Split(body, "\n")
	if strings.TrimSpace(lines[0]) == "" || strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatalf("expected no empty edge lines: %q", body)
	}
}

func TestParseRoutine_trimIndentationFalse(t *testing.T) {
	res, err := ParseRoutine(listPartitionsSQLWithComment, Options{TrimBody: true, TrimIndentation: false})
	if err != nil {
		t.Fatal(err)
	}
	// Capture often leaves the first line flush-left after AS (, but deeper
	// lines retain absolute heredoc/DDL indent until TrimIndentation runs.
	if !strings.Contains(res.DefinitionBody, "\n        t.table_schema") {
		t.Fatalf("expected deep absolute indent preserved when TrimIndentation=false: %q", res.DefinitionBody)
	}
	dedented, err := ParseRoutine(listPartitionsSQLWithComment, Options{TrimBody: true, TrimIndentation: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dedented.DefinitionBody, "\n        t.table_schema") {
		t.Fatalf("expected TrimIndentation=true to remove deep absolute indent: %q", dedented.DefinitionBody)
	}
	if !strings.Contains(dedented.DefinitionBody, "\n  t.table_schema") {
		t.Fatalf("expected relative indent kept after dedent: %q", dedented.DefinitionBody)
	}
}

func TestParseRoutine_trimAll(t *testing.T) {
	res, err := ParseRoutine(listPartitionsSQLWithComment, Options{
		TrimBody:        true,
		TrimComments:    true,
		TrimIndentation: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := res.DefinitionBody
	if strings.Contains(body, "-- comment") {
		t.Fatalf("expected comment removed: %q", body)
	}
	lines := strings.Split(body, "\n")
	wantFirst := "SELECT"
	if lines[0] != wantFirst {
		t.Fatalf("expected dedented first line %q, got %q\nfull=%q", wantFirst, lines[0], body)
	}
	if strings.HasPrefix(body, "\n") || strings.HasSuffix(body, "\n") {
		t.Fatalf("expected edges trimmed: %q", body)
	}
}

func TestParseView_trimComments(t *testing.T) {
	sql := `
    CREATE OR REPLACE VIEW mydataset.my_view AS
      SELECT
        a, -- keep me gone
        b
      FROM t
      WHERE x = 1
    ;`

	res, err := ParseView(sql, Options{TrimBody: true, TrimComments: true, TrimIndentation: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Query, "-- keep me gone") {
		t.Fatalf("expected comment removed: %q", res.Query)
	}
	if !strings.Contains(res.Query, "SELECT") || !strings.Contains(res.Query, "FROM t") {
		t.Fatalf("query=%q", res.Query)
	}
}
