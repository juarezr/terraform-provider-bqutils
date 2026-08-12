package sqlparse

import (
	"strings"
	"testing"
	"time"
)

// parseRoutineWithTimeout fails fast if ParseRoutine hangs (e.g. lexer zero-width loop).
func parseRoutineWithTimeout(t *testing.T, sql string, opts Options) (*ParseResult, error) {
	t.Helper()
	type outcome struct {
		res *ParseResult
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		res, err := ParseRoutine(sql, opts)
		ch <- outcome{res, err}
	}()
	select {
	case out := <-ch:
		return out.res, out.err
	case <-time.After(2 * time.Second):
		t.Fatal("ParseRoutine hung (possible infinite lexer loop)")
		return nil, nil
	}
}

func TestParseProcedure_systemVariableNoHang(t *testing.T) {
	// Regression: '@' was isIdentStart but not isIdentPart, so @@project_id
	// caused scanIdent to advance 0 bytes and freeze Terraform.
	sql := `
CREATE OR REPLACE PROCEDURE mydataset1.export_to_gcs(inicio TIMESTAMP)
BEGIN
  DECLARE env_name STRING DEFAULT mydataset5.get_env_name(@@project_id);
  SELECT env_name;
END;
`
	res, err := parseRoutineWithTimeout(t, sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindProcedure || res.ObjectID != "export_to_gcs" {
		t.Fatalf("kind=%s id=%s", res.Kind, res.ObjectID)
	}
	if !strings.Contains(res.DefinitionBody, "@@project_id") {
		t.Fatalf("body missing @@project_id: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "SELECT env_name") {
		t.Fatalf("body truncated: %q", res.DefinitionBody)
	}
}

func TestParseProcedure_endIfAndCaseExpression(t *testing.T) {
	sql := `
CREATE OR REPLACE PROCEDURE mydataset.demo(separador STRING)
BEGIN
  DECLARE separador2 STRING DEFAULT COALESCE(NULLIF(TRIM(separador),''),';');
  DECLARE replacement STRING DEFAULT CASE WHEN separador2 = ';' THEN ',' WHEN separador2 = ',' THEN ';' ELSE '' END;

  IF separador2 IS NULL OR LENGTH(separador2) < 1 THEN
    RAISE USING message = FORMAT("Error: separador is required.");
  END IF;

  SELECT separador2, replacement;
END;
`
	res, err := parseRoutineWithTimeout(t, sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.DefinitionBody, "END IF") {
		t.Fatalf("body missing END IF: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "CASE WHEN") {
		t.Fatalf("body missing CASE: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "SELECT separador2, replacement") {
		t.Fatalf("body truncated before SELECT: %q", res.DefinitionBody)
	}
}

func TestParseProcedure_exportDataFixtureNoHang(t *testing.T) {
	// Full fixture that previously froze Terraform (@@project_id + END IF + CASE).
	sql := `
CREATE OR REPLACE PROCEDURE mydataset1.export_to_gcs
(
  started              TIMESTAMP,
  finished             TIMESTAMP,
  separator           STRING
) BEGIN

DECLARE file_uuid STRING DEFAULT GENERATE_UUID ();

DECLARE dt_started STRING DEFAULT FORMAT_DATETIME('%Y%m%d', started);
DECLARE dt_finished STRING DEFAULT FORMAT_DATETIME('%Y%m%d', finished);
DECLARE dt_current STRING DEFAULT FORMAT_DATETIME('%Y%m%d_%H%M%S', CURRENT_TIMESTAMP);

DECLARE env_name STRING DEFAULT mydataset3.get_env_name(@@project_id);

DECLARE separator2 STRING DEFAULT COALESCE(NULLIF(TRIM(separator),''),';');
DECLARE replacement STRING DEFAULT CASE WHEN separator2 = ';' THEN ',' WHEN separator2 = ',' THEN ';' ELSE '' END;
DECLARE decimalrepl STRING DEFAULT CASE separator2 WHEN '.' THEN ',' ELSE '.' END;

DECLARE gcs_uri STRING DEFAULT FORMAT(
    'gs://mybucket/env-%s/my-events-at-%s-from-%s-%s-%s-*.csv.gz',
    env_name, dt_current, dt_started, dt_finished, file_uuid);

EXPORT DATA OPTIONS (
    uri=( gcs_uri ),
    overwrite = true,
    compression = 'GZIP',
    format = 'CSV',
    header = true,
    field_delimiter = ( separator2 )
) AS (
  SELECT
        ev.eventid
      , FORMAT_DATETIME('%Y-%m-%d %H:%M:%S', DATETIME_TRUNC(DATETIME(ev.created), SECOND)) AS created
      , (SELECT te.eventname FROM mydataset2.eventtype AS te WHERE te.id = ev.eventtypeid) AS eventname
      , REPLACE(TRIM(CAST(ROUND(ev.latitude,  6) AS STRING FORMAT '99990.099999')), '.', decimalrepl) AS latitude
      , REPLACE(TRIM(CAST(ROUND(ev.longitude, 6) AS STRING FORMAT '99990.099999')), '.', decimalrepl) AS longitude
      , REPLACE(TRIM(CAST(ROUND(ev.mileage / 1000.0, 3) AS STRING FORMAT '9999999990.000')), '.', decimalrepl) AS mileage
      , REPLACE(CAST(ROUND(ev.velocity, 2) AS STRING), '.', decimalrepl) AS velocity
  FROM mydataset2.events AS ev
  WHERE ev.created BETWEEN started AND finished
    AND ev.created IS NOT NULL
  ORDER BY ev.eventid ASC, ev.created ASC, ev.mileage DESC, ev.eventtypeid
  LIMIT 999999999999
);

SELECT gcs_uri AS gcs_uri;

END;
`
	res, err := parseRoutineWithTimeout(t, sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "export_to_gcs" {
		t.Fatalf("object_id=%s", res.ObjectID)
	}
	if !strings.Contains(res.DefinitionBody, "@@project_id") {
		t.Fatalf("body missing @@project_id")
	}
	if !strings.Contains(res.DefinitionBody, "EXPORT DATA") {
		t.Fatalf("body truncated before EXPORT DATA: %q", res.DefinitionBody[:min(200, len(res.DefinitionBody))])
	}
	if !strings.Contains(res.DefinitionBody, "SELECT gcs_uri AS gcs_uri") {
		t.Fatalf("body truncated before final SELECT")
	}
}

func TestParseProcedure_malformedRecoversWithoutHang(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		wantErrSub string
	}{
		{
			name: "missing_outer_END",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  SELECT 1;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "unclosed_CASE_expression",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  DECLARE x STRING DEFAULT CASE WHEN 1 = 1 THEN 'a' ELSE 'b';
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "missing_END_IF",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  IF 1 = 1 THEN
    SELECT 1;
  -- missing END IF
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "missing_END_WHILE",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  WHILE TRUE DO
    SELECT 1;
  -- missing END WHILE
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "missing_END_LOOP",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  LOOP
    SELECT 1;
  -- missing END LOOP
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "missing_END_FOR",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  FOR record IN (SELECT 1 AS x) DO
    SELECT record.x;
  -- missing END FOR
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "missing_END_REPEAT",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  REPEAT
    SELECT 1;
  UNTIL TRUE
  -- missing END REPEAT
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "nested_BEGIN_missing_inner_END",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  BEGIN
    SELECT 1;
  -- missing inner END
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "unterminated_string_in_body",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  SET msg = 'oops no close;
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "unterminated_block_comment_in_body",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  /* never closed
  SELECT 1;
END;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
		{
			name: "missing_BEGIN_or_AS",
			sql: `
CREATE PROCEDURE mydataset.p()
SELECT 1;
`,
			wantErrSub: "expected AS or BEGIN",
		},
		{
			name: "unterminated_parenthesized_AS_body",
			sql: `
CREATE TABLE FUNCTION mydataset.p()
AS (
  SELECT 1
`,
			wantErrSub: "unterminated body parentheses",
		},
		{
			name: "broken_OPTIONS_before_BEGIN",
			sql: `
CREATE PROCEDURE mydataset.p()
OPTIONS (description = 'x'
BEGIN
  SELECT 1;
END;
`,
			wantErrSub: "expected )",
		},
		{
			name: "orphan_END_IF_then_missing_outer_END",
			sql: `
CREATE PROCEDURE mydataset.p()
BEGIN
  END IF;
  SELECT 1;
`,
			wantErrSub: "unterminated BEGIN/END body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRoutineWithTimeout(t, tt.sql, Options{TrimBody: true})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("error=%q, want substring %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

func TestParseProcedure_controlStructuresStillCloseCleanly(t *testing.T) {
	// Completeness counterpart: properly closed IF/WHILE/FOR/LOOP must not error.
	sql := `
CREATE PROCEDURE mydataset.p()
BEGIN
  IF TRUE THEN
    SELECT 1;
  END IF;

  WHILE FALSE DO
    SELECT 2;
  END WHILE;

  LOOP
    BREAK;
  END LOOP;

  FOR record IN (SELECT 1 AS x) DO
    SELECT record.x;
  END FOR;

  REPEAT
    SELECT 3;
  UNTIL TRUE
  END REPEAT;

  SELECT 4;
END;
`
	res, err := parseRoutineWithTimeout(t, sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"END IF", "END WHILE", "END LOOP", "END FOR", "END REPEAT", "SELECT 4"} {
		if !strings.Contains(res.DefinitionBody, want) {
			t.Fatalf("body missing %q: %q", want, res.DefinitionBody)
		}
	}
}
