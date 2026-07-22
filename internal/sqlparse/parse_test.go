package sqlparse

import (
	"strings"
	"testing"
)

func TestParseTableFunction(t *testing.T) {
	sql := `
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'Used to evaluate partition details'
    ) AS (
      SELECT
        table_schema,
        table_name
      FROM x
      WHERE partition_id != '__NULL__'  -- comment
        AND t.table_name = table_name_filter
    );`

	res, err := ParseRoutine(sql, Options{TrimBody: true, TrimComments: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindTableFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.DatasetID != "mydataset" || res.ObjectID != "list_partitions" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	if res.Language != "SQL" {
		t.Fatalf("lang=%s", res.Language)
	}
	if res.Description != "Used to evaluate partition details" {
		t.Fatalf("desc=%q", res.Description)
	}
	if len(res.Arguments) != 1 || res.Arguments[0].Name != "table_name_filter" {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if !strings.Contains(res.Arguments[0].DataTypeJSON, "STRING") {
		t.Fatalf("dtype=%s", res.Arguments[0].DataTypeJSON)
	}
	if !strings.Contains(res.DefinitionBody, "SELECT") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
	if !strings.Contains(res.DefinitionBody, "-- comment") {
		t.Fatalf("expected comment preserved: %q", res.DefinitionBody)
	}
}

func TestParseJSFunction(t *testing.T) {
	sql := `
    CREATE OR REPLACE FUNCTION parse_json_to_array(json_str STRING)
    RETURNS ARRAY<STRING>
    LANGUAGE js AS r"""
      try {
        let parsed = JSON.parse(json_str);
        return parsed.tags || [];
      } catch (e) {
        return [];
      }
    """;`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindScalarFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.Language != "JAVASCRIPT" {
		t.Fatalf("lang=%s", res.Language)
	}
	if !strings.Contains(res.ReturnTypeJSON, "ARRAY") {
		t.Fatalf("return=%s", res.ReturnTypeJSON)
	}
	if !strings.Contains(res.DefinitionBody, "JSON.parse") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseAggregateFunction(t *testing.T) {
	sql := `
    CREATE AGGREGATE FUNCTION appfleet.scaled_sum
    (
      dividend FLOAT64,
      divisor FLOAT64
    ) RETURNS FLOAT64 AS (
      SUM(dividend) / SUM(divisor)
    );`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindAggregateFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.DatasetID != "appfleet" || res.ObjectID != "scaled_sum" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	if !strings.Contains(res.ReturnTypeJSON, "FLOAT64") {
		t.Fatalf("return=%s", res.ReturnTypeJSON)
	}
	if len(res.Arguments) != 2 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[0].IsAggregate == nil || !*res.Arguments[0].IsAggregate {
		t.Fatalf("dividend IsAggregate=%v", res.Arguments[0].IsAggregate)
	}
	if res.Arguments[1].IsAggregate == nil || !*res.Arguments[1].IsAggregate {
		t.Fatalf("divisor IsAggregate=%v", res.Arguments[1].IsAggregate)
	}
	if !strings.Contains(res.DefinitionBody, "SUM(dividend)") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseAggregateFunctionNotAggregate(t *testing.T) {
	sql := `
    CREATE AGGREGATE FUNCTION appfleet.weighted_sum
    (
      dividend FLOAT64,
      divisor FLOAT64 NOT AGGREGATE
    ) RETURNS FLOAT64 AS (
      SUM(dividend) / divisor
    );`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindAggregateFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.ObjectID != "weighted_sum" {
		t.Fatalf("id=%s", res.ObjectID)
	}
	if len(res.Arguments) != 2 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[0].Name != "dividend" || res.Arguments[0].IsAggregate == nil || !*res.Arguments[0].IsAggregate {
		t.Fatalf("dividend=%+v", res.Arguments[0])
	}
	if res.Arguments[1].Name != "divisor" || res.Arguments[1].IsAggregate == nil || *res.Arguments[1].IsAggregate {
		t.Fatalf("divisor=%+v", res.Arguments[1])
	}
	if !strings.Contains(res.Arguments[1].DataTypeJSON, "FLOAT64") {
		t.Fatalf("dtype=%s", res.Arguments[1].DataTypeJSON)
	}
	if !strings.Contains(res.DefinitionBody, "SUM(dividend)") {
		t.Fatalf("body=%q", res.DefinitionBody)
	}
}

func TestParseAggregateFunctionNotAggregateFirstArg(t *testing.T) {
	sql := `
    CREATE OR REPLACE AGGREGATE FUNCTION mydataset.f
    (
      scale FLOAT64 NOT AGGREGATE,
      value FLOAT64
    ) RETURNS FLOAT64 AS (
      SUM(value) * scale
    );`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Arguments) != 2 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[0].IsAggregate == nil || *res.Arguments[0].IsAggregate {
		t.Fatalf("scale=%+v", res.Arguments[0])
	}
	if res.Arguments[1].IsAggregate == nil || !*res.Arguments[1].IsAggregate {
		t.Fatalf("value=%+v", res.Arguments[1])
	}
}

func TestParseScalarFunctionNotAggregateSuffix(t *testing.T) {
	// Suffix is accepted whenever present after a type (DDL surface); non-UDAF leaves IsAggregate=false only.
	sql := `CREATE FUNCTION mydataset.f(x INT64 NOT AGGREGATE) RETURNS INT64 AS (x);`

	res, err := ParseRoutine(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindScalarFunction {
		t.Fatalf("kind=%s", res.Kind)
	}
	if len(res.Arguments) != 1 {
		t.Fatalf("args=%+v", res.Arguments)
	}
	if res.Arguments[0].IsAggregate == nil || *res.Arguments[0].IsAggregate {
		t.Fatalf("arg=%+v", res.Arguments[0])
	}
}

func TestTempErrors(t *testing.T) {
	sql := `CREATE TEMP FUNCTION foo(x INT64) AS (x+1);`
	_, err := ParseRoutine(sql, Options{TrimBody: true})
	if err == nil {
		t.Fatal("expected TEMP error")
	}
	if !strings.Contains(err.Error(), "TEMP") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseView(t *testing.T) {
	sql := `
    CREATE OR REPLACE VIEW IF NOT EXISTS ` + "`mydataset.my_simple_view`" + `
    (
      TABLE_SCHEMA OPTIONS(description="The schema of the table"),
      TABLE_NAME OPTIONS(description="The name of the table")
    ) OPTIONS(
      description="Simple view created by Terraform"
    ) AS
      SELECT TABLE_SCHEMA, TABLE_NAME
      FROM mydataset.INFORMATION_SCHEMA.TABLES;`

	res, err := ParseView(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.ObjectID != "my_simple_view" || res.DatasetID != "mydataset" {
		t.Fatalf("name=%s.%s", res.DatasetID, res.ObjectID)
	}
	if res.IsMaterialized {
		t.Fatal("not materialized")
	}
	if res.Description != "Simple view created by Terraform" {
		t.Fatalf("desc=%q", res.Description)
	}
	if len(res.Columns) != 2 {
		t.Fatalf("cols=%+v", res.Columns)
	}
	if res.SchemaJSON == "" {
		t.Fatal("expected schema json")
	}
	if !strings.Contains(res.Query, "SELECT") {
		t.Fatalf("query=%q", res.Query)
	}
}

func TestParseMaterializedView(t *testing.T) {
	sql := `
    CREATE OR REPLACE MATERIALIZED VIEW IF NOT EXISTS ` + "`mydataset.my_materialized_view`" + `
    PARTITION BY DATE(TABLE_CREATION_TIME)
    CLUSTER BY TABLE_SCHEMA, TABLE_NAME
    OPTIONS(
      description="Materialized view created by Terraform",
      enable_refresh=TRUE,
      allow_non_incremental_definition=FALSE,
      refresh_interval_minutes=60,
      max_staleness=INTERVAL "4:0:0" HOUR TO SECOND,
      retain_partitions=true,
      kms_key_name="projects/123/key",
      labels=[("org_unit", "development")]
    ) AS
      SELECT TABLE_SCHEMA, TABLE_NAME
      FROM mydataset.INFORMATION_SCHEMA.TABLES;`

	res, err := ParseView(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsMaterialized {
		t.Fatal("expected materialized")
	}
	if res.PartitioningField != "TABLE_CREATION_TIME" || res.PartitioningType != "DAY" {
		t.Fatalf("partition=%s/%s", res.PartitioningType, res.PartitioningField)
	}
	if len(res.Clustering) != 2 {
		t.Fatalf("cluster=%v", res.Clustering)
	}
	if res.RefreshIntervalMs == nil || *res.RefreshIntervalMs != 3600000 {
		t.Fatalf("refresh=%v", res.RefreshIntervalMs)
	}
	if res.EnableRefresh == nil || !*res.EnableRefresh {
		t.Fatal("enable_refresh")
	}
	if res.Labels["org_unit"] != "development" {
		t.Fatalf("labels=%v", res.Labels)
	}
	if res.KmsKeyName != "projects/123/key" {
		t.Fatalf("kms=%s", res.KmsKeyName)
	}
	if res.MaxStaleness != "0-0 0 4:0:0" {
		t.Fatalf("max_staleness=%q", res.MaxStaleness)
	}
}

func TestParseMaterializedViewSimpleInterval(t *testing.T) {
	sql := `
    CREATE OR REPLACE MATERIALIZED VIEW mydataset.mv
    OPTIONS(
      description="mv",
      max_staleness=INTERVAL 90 MINUTE
    ) AS
      SELECT 1 AS x;`

	res, err := ParseView(sql, Options{TrimBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsMaterialized {
		t.Fatal("expected materialized")
	}
	if res.MaxStaleness != "0-0 0 1:30:0" {
		t.Fatalf("max_staleness=%q", res.MaxStaleness)
	}
}

func TestTypeJSON(t *testing.T) {
	js, err := sqlTypeToJSON("ARRAY<STRING>")
	if err != nil {
		t.Fatal(err)
	}
	if js != `{"typeKind":"ARRAY","arrayElementType":{"typeKind":"STRING"}}` {
		t.Fatalf("%s", js)
	}
}
