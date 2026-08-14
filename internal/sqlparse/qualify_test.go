package sqlparse

import (
	"reflect"
	"strings"
	"testing"
)

func TestQualifyBody_arrayDistinctCall(t *testing.T) {
	body := `SELECT t.category
        , sum(price) AS total
        , mydataset5.array_distinct(ARRAY_CONCAT_AGG(t.items)) AS unique_items
     FROM tab AS t
    WHERE t.id <= max_value
    GROUP BY t.category`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "myproj",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset5"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if !strings.Contains(got.Body, "`myproj`.`mydataset5`.`array_distinct`(") {
		t.Fatalf("body not rewritten: %q", got.Body)
	}
	if strings.Contains(got.Body, " myproj.tab ") || strings.Contains(got.Body, "FROM myproj.tab") {
		t.Fatalf("CTE/alias tab should not be qualified: %q", got.Body)
	}
}

func TestQualifyBody_fromJoinMultiDataset(t *testing.T) {
	body := `SELECT *
FROM mydataset4.vehicle AS v
JOIN mydataset7.customer AS c ON v.id = c.id
LEFT JOIN mydataset6.driver d ON d.id = c.driver_id
LEFT JOIN mydataset7.target t ON t.id = d.target_id
`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "env-proj",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	wantRefs := []string{"mydataset4", "mydataset6", "mydataset7"}
	if !reflect.DeepEqual(got.DatasetReferences, wantRefs) {
		t.Fatalf("refs=%v want=%v", got.DatasetReferences, wantRefs)
	}
	for _, ds := range wantRefs {
		needle := "`env-proj`.`" + ds + "`."
		if !strings.Contains(got.Body, needle) {
			t.Fatalf("missing qualify for %s in %q", ds, got.Body)
		}
	}
}

func TestQualifyBody_alreadyQualified(t *testing.T) {
	body := `SELECT * FROM otherproj.mydataset4.vehicle`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "env-proj",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset4"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if got.Body != body {
		t.Fatalf("three-part must not be rewritten: %q", got.Body)
	}
}

func TestQualifyBody_sameDatasetOnly(t *testing.T) {
	body := `SELECT * FROM mydataset1.events e JOIN mydataset1.other o ON e.id = o.id`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "env-proj",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if len(got.DatasetReferences) != 0 {
		t.Fatalf("expected empty refs, got %v", got.DatasetReferences)
	}
	if got.Body != body {
		t.Fatalf("same-dataset body must be unchanged: %q", got.Body)
	}
}

func TestQualifyBody_placeholder(t *testing.T) {
	body := "SELECT id FROM `${project}`.`mydataset2`.`mytable`"
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "env-proj",
		HomeDataset:   "mydataset",
		Rewrite:       true,
	})
	if !strings.Contains(got.Body, "`env-proj`.`mydataset2`.`mytable`") {
		t.Fatalf("placeholder not replaced: %q", got.Body)
	}
	if strings.Contains(got.Body, "${project}") {
		t.Fatalf("placeholder remains: %q", got.Body)
	}
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset2"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
}

func TestQualifyBody_procedurePatterns(t *testing.T) {
	body := `
BEGIN
  DECLARE env_name STRING DEFAULT mydataset5.get_env_name(@@project_id);
  SET env_name = mydataset5.get_env_name(@@project_id);
  CALL mydataset2.myprocedure2(arg1, arg2);
  INSERT INTO mydataset2.mytable3 VALUES (1, 2, 4);
  UPDATE mydataset2.mytable3 SET myfield1 = 5;
  DELETE FROM mydataset3.mytable4 WHERE myfield2 = 6;
  TRUNCATE TABLE mydataset3.mytable5;
  MERGE INTO mydataset2.mytable5 AS t4 USING mydataset3.mytable6 AS t6 ON t4.id = t6.id
  WHEN MATCHED THEN UPDATE SET x = 1;
END`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "p",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	want := []string{"mydataset2", "mydataset3", "mydataset5"}
	if !reflect.DeepEqual(got.DatasetReferences, want) {
		t.Fatalf("refs=%v want=%v", got.DatasetReferences, want)
	}
	checks := []string{
		"`p`.`mydataset5`.`get_env_name`(",
		"`p`.`mydataset2`.`myprocedure2`(",
		"`p`.`mydataset2`.`mytable3`",
		"`p`.`mydataset3`.`mytable4`",
		"`p`.`mydataset3`.`mytable5`",
		"`p`.`mydataset3`.`mytable6`",
	}
	for _, c := range checks {
		if !strings.Contains(got.Body, c) {
			t.Fatalf("missing %q in %q", c, got.Body)
		}
	}
}

func TestQualifyBody_backticks(t *testing.T) {
	body := "SELECT * FROM `mydataset7.driver` JOIN `mydataset4`.`vehicle` d ON true"
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "p",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset4", "mydataset7"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if !strings.Contains(got.Body, "`p`.`mydataset4`.`vehicle`") {
		t.Fatalf("per-segment backticks: %q", got.Body)
	}
	if !strings.Contains(got.Body, "`p`.`mydataset7`.`driver`") {
		t.Fatalf("joined backtick source must rewrite to per-segment: %q", got.Body)
	}
}

func TestQualifyBody_noRewriteViews(t *testing.T) {
	body := `SELECT e1.id FROM mydataset8.eventkind AS e1 JOIN mydataset9.eventclass AS t2 ON e1.id = t2.id`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "p",
		HomeDataset:   "mydataset1",
		Rewrite:       false,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset8", "mydataset9"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if got.Body != body {
		t.Fatalf("rewrite=false must keep body: %q", got.Body)
	}
}

func TestQualifyBody_unnestAndCTE(t *testing.T) {
	body := `
WITH tab AS (
  SELECT 1 AS id
)
SELECT * FROM tab, UNNEST([1,2,3]) AS number
CROSS JOIN mydataset5.lookup l`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "p",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset5"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if strings.Contains(got.Body, "`p`.`tab`") || strings.Contains(got.Body, "p.tab") {
		t.Fatalf("CTE tab must not be qualified: %q", got.Body)
	}
	if !strings.Contains(got.Body, "`p`.`mydataset5`.`lookup`") {
		t.Fatalf("lookup not qualified: %q", got.Body)
	}
}

func TestQualifyBody_tvfCallInFrom(t *testing.T) {
	body := `SELECT * FROM mydataset4.test_returns_table(3)`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "p",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset4"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if !strings.Contains(got.Body, "`p`.`mydataset4`.`test_returns_table`(3)") {
		t.Fatalf("body=%q", got.Body)
	}
}

func TestQualifyBody_hyphenatedProject(t *testing.T) {
	body := `SELECT mydataset2.fn1([1, 1, 2]) AS items`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "my-project-123",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	want := "SELECT `my-project-123`.`mydataset2`.`fn1`([1, 1, 2]) AS items"
	if got.Body != want {
		t.Fatalf("body=%q want=%q", got.Body, want)
	}
	if strings.Contains(got.Body, "my-project-123.mydataset2") {
		t.Fatalf("unquoted hyphenated project must not appear: %q", got.Body)
	}
}

func TestQualifyBody_noTargetProject(t *testing.T) {
	body := `SELECT mydataset5.array_distinct([1])`
	got := QualifyBody(body, QualifyOptions{
		HomeDataset: "mydataset1",
		Rewrite:     true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset5"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if got.Body != body {
		t.Fatalf("without target project body unchanged: %q", got.Body)
	}
}

func TestQualifyBody_informationSchema(t *testing.T) {
	body := "SELECT t.table_name FROM `mydataset2`.INFORMATION_SCHEMA.TABLES AS t"
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "env-proj",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset2"}) {
		t.Fatalf("refs=%v", got.DatasetReferences)
	}
	if !strings.Contains(got.Body, "`env-proj`.`mydataset2`.INFORMATION_SCHEMA.TABLES") {
		t.Fatalf("body=%q", got.Body)
	}
}

func TestQualifyBody_extractFromNotTable(t *testing.T) {
	// EXTRACT(part FROM expr) must not treat FROM as a table clause.
	body := `SELECT
  EXTRACT(DAYOFWEEK FROM t1.col1) AS dow,
  EXTRACT(DATE FROM TIMESTAMP(t1.col1)) AS d
FROM mydataset2.table1 AS t1
WHERE EXTRACT(DAYOFWEEK FROM t1.col1) * 100 + t2.col2 BETWEEN 1 AND 7`
	got := QualifyBody(body, QualifyOptions{
		TargetProject: "my-project-123",
		HomeDataset:   "mydataset1",
		Rewrite:       true,
	})
	if !reflect.DeepEqual(got.DatasetReferences, []string{"mydataset2"}) {
		t.Fatalf("refs=%v want=[mydataset2] (t1.col1 is alias.column, not a dataset)", got.DatasetReferences)
	}
	if strings.Contains(got.Body, "`t1`.`col1`") || strings.Contains(got.Body, "my-project-123`.`t1`") {
		t.Fatalf("EXTRACT operand must not be project-qualified: %q", got.Body)
	}
	if !strings.Contains(got.Body, "EXTRACT(DAYOFWEEK FROM t1.col1)") {
		t.Fatalf("EXTRACT expression altered: %q", got.Body)
	}
	if !strings.Contains(got.Body, "`my-project-123`.`mydataset2`.`table1`") {
		t.Fatalf("real FROM table not qualified: %q", got.Body)
	}
}

func TestQualifyBody_objectReferences(t *testing.T) {
	body := `
BEGIN
  DECLARE x STRING DEFAULT mydataset5.get_env_name(@@project_id);
  CALL mydataset2.myprocedure2(arg1);
  INSERT INTO mydataset2.mytable3 VALUES (1);
  SELECT * FROM mydataset4.tvf(1) AS t
  JOIN mydataset7.customer AS c ON true;
  SELECT name FROM mydataset2.INFORMATION_SCHEMA.TABLES;
END`
	got := QualifyBody(body, QualifyOptions{
		HomeDataset: "mydataset1",
		HomeObject:  "myproc",
		Rewrite:     false,
	})
	want := []ObjectReference{
		{DatasetID: "mydataset2", ObjectID: "INFORMATION_SCHEMA.TABLES", ObjectType: "TABLE", ResourceType: "VIEW"},
		{DatasetID: "mydataset2", ObjectID: "myprocedure2", ObjectType: "PROCEDURE", ResourceType: "ROUTINE"},
		{DatasetID: "mydataset2", ObjectID: "mytable3", ObjectType: "VIEW", ResourceType: "VIEW"},
		{DatasetID: "mydataset4", ObjectID: "tvf", ObjectType: "TABLE_VALUED_FUNCTION", ResourceType: "ROUTINE"},
		{DatasetID: "mydataset5", ObjectID: "get_env_name", ObjectType: "SCALAR_FUNCTION", ResourceType: "ROUTINE"},
		{DatasetID: "mydataset7", ObjectID: "customer", ObjectType: "VIEW", ResourceType: "VIEW"},
	}
	if !reflect.DeepEqual(got.References, want) {
		t.Fatalf("got=%#v\nwant=%#v", got.References, want)
	}
}

func TestQualifyBody_objectReferencesExcludeSelf(t *testing.T) {
	body := `SELECT mydataset1.helper(1), mydataset1.self_fn(2)`
	got := QualifyBody(body, QualifyOptions{
		HomeDataset: "mydataset1",
		HomeObject:  "self_fn",
		Rewrite:     false,
	})
	want := []ObjectReference{
		{DatasetID: "mydataset1", ObjectID: "helper", ObjectType: "SCALAR_FUNCTION", ResourceType: "ROUTINE"},
	}
	if !reflect.DeepEqual(got.References, want) {
		t.Fatalf("got=%#v want=%#v", got.References, want)
	}
}

func TestQualifyBody_objectReferencesUniqueSorted(t *testing.T) {
	body := `SELECT mydataset2.fn(1), mydataset2.fn(2) FROM mydataset1.t`
	got := QualifyBody(body, QualifyOptions{
		HomeDataset: "other",
		HomeObject:  "x",
		Rewrite:     false,
	})
	want := []ObjectReference{
		{DatasetID: "mydataset1", ObjectID: "t", ObjectType: "VIEW", ResourceType: "VIEW"},
		{DatasetID: "mydataset2", ObjectID: "fn", ObjectType: "SCALAR_FUNCTION", ResourceType: "ROUTINE"},
	}
	if !reflect.DeepEqual(got.References, want) {
		t.Fatalf("got=%#v want=%#v", got.References, want)
	}
}
