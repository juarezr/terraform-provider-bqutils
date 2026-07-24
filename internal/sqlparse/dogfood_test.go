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
