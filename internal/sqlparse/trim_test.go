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
