package sqlparse

import (
	"strings"
	"testing"
)

// Dogfood cases adapted from real CREATE scripts (sketch/samples), embedded so
// CI does not depend on the gitignored sketch/ tree.

func TestParseProcedure(t *testing.T) {
	// Shape from sketch/samples/mydataset.myprocedure.sql: PROCEDURE + OPTIONS + BEGIN…END (no AS).
	sql := `
    CREATE OR REPLACE PROCEDURE mydataset.myprocedure
    (
        started  DATE,
        finished DATE
    ) OPTIONS (
      description = 'daily processing'
    )
    BEGIN
      DECLARE started_min, finished_max TIMESTAMP;

      SET (started_min, finished_max) = (TIMESTAMP(started), TIMESTAMP(finished));

      DELETE FROM mydataset.dailymileage
       WHERE localdate BETWEEN started AND finished;

      INSERT INTO mydataset.dailymileage (localdate, vehicleid)
      SELECT started, 1;
    END;
    `

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindProcedure {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.DatasetID != "mydataset" || res.ObjectID != "myprocedure" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	if res.Language != "SQL" {
		t.Fatalf("lang=%s", res.Language)
	}
	if res.Description != "daily processing" {
		t.Fatalf("desc=%q", res.Description)
	}
	if len(res.Arguments) != 2 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[0].Name != "started" || !strings.Contains(res.Arguments[0].DataTypeJSON, "DATE") {
		t.Fatalf("started=%+v", res.Arguments[0])
	}
	if res.Arguments[1].Name != "finished" || !strings.Contains(res.Arguments[1].DataTypeJSON, "DATE") {
		t.Fatalf("finished=%+v", res.Arguments[1])
	}
	if !strings.Contains(res.DefinitionBody, "BEGIN") {
		t.Fatalf("expected BEGIN in body: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "DELETE FROM mydataset.dailymileage") {
		t.Fatalf("expected DELETE in body: %q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "INSERT INTO mydataset.dailymileage") {
		t.Fatalf("expected INSERT in body: %q", res.DefinitionBody)
	}
}

func TestParseTableFunctionMyReport(t *testing.T) {
	// Shape from sketch/samples/mydataset.myreport.sql: wide TABLE FUNCTION signature + AS (...).
	sql := `
    CREATE OR REPLACE TABLE FUNCTION mydataset.myreport
    (
        clientid           INTEGER,      -- All fields below are required
        firstday           DATE,
        lastday            DATE,
        dawnstarts         SMALLINT,     -- From configuration of journey
        morningstart       SMALLINT,
        noonstarts         SMALLINT,
        nightstarts        SMALLINT,
        firstdayjourney    SMALLINT,
        firsthourjourney   SMALLINT,
        lastdayjourney     SMALLINT,
        lasthourjourney    SMALLINT,
        firsthourwork      SMALLINT,
        lasthourwork       SMALLINT,
        defaultdriver      BOOLEAN,      -- TRUE:defaultdriver FALSE:driverid NULL: Both
        groupid            SMALLINT,     -- Exclusive: groupid XOR vehicleid
        vehicleid          ARRAY<INTEGER>  -- Exclusive: groupid XOR vehicleid
    ) OPTIONS (
      description = 'Retrieves the report data'
    ) AS (
      SELECT
        table_schema,
        table_name,
        partition_id,
        total_rows
      FROM mydataset.INFORMATION_SCHEMA.PARTITIONS
      WHERE partition_id != '__NULL__'
    );
    `

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindTableFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.DatasetID != "mydataset" || res.ObjectID != "myreport" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	if res.Language != "SQL" {
		t.Fatalf("lang=%s", res.Language)
	}
	if res.Description != "Retrieves the report data" {
		t.Fatalf("desc=%q", res.Description)
	}
	if len(res.Arguments) != 16 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[0].Name != "clientid" || !strings.Contains(res.Arguments[0].DataTypeJSON, "INT64") {
		t.Fatalf("clientid=%+v", res.Arguments[0])
	}
	if res.Arguments[13].Name != "defaultdriver" || !strings.Contains(res.Arguments[13].DataTypeJSON, "BOOL") {
		t.Fatalf("defaultdriver=%+v", res.Arguments[13])
	}
	if res.Arguments[15].Name != "vehicleid" || !strings.Contains(res.Arguments[15].DataTypeJSON, "ARRAY") {
		t.Fatalf("vehicleid=%+v", res.Arguments[15])
	}
	if !strings.Contains(res.DefinitionBody, "SELECT") {
		t.Fatalf("expected SELECT in body: %q", res.DefinitionBody)
	}
}
