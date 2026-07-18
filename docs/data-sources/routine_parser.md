---
page_title: "bqutils_routine_parser Data Source - bqutils"
subcategory: ""
description: |-
  Parses BigQuery CREATE FUNCTION / PROCEDURE SQL for google_bigquery_routine.
---

# bqutils_routine_parser (Data Source)

Parses a BigQuery `CREATE FUNCTION`, `CREATE TABLE FUNCTION`, `CREATE PROCEDURE`, or `CREATE AGGREGATE FUNCTION` statement.

TEMPORARY routines produce an error.

## Example Usage

### Loading SQL from a file

```terraform
data "bqutils_routine_parser" "example" {
  sql           = file("${path.module}/routine.sql")
  trim_body     = true
  trim_comments = false
}
```

### Parsing SQL and creating a TABLE FUNCTION in BigQuery

```terraform
data "bqutils_routine_parser" "list_partitions" {

  sql = <<EOF
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'Used to evaluate partition details in a partitioned table inside the base1 dataset.'
    ) AS (
      SELECT
        table_schema,
        table_name,
        partition_id,
        total_rows
      FROM `mydataset`.INFORMATION_SCHEMA.PARTITIONS
      WHERE partition_id != '__NULL__'  -- Ignore partitions without rows
        AND t.table_name    = table_name_filter
    );
  EOF

  trim_comments = false
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

resource "google_bigquery_routine" "list_partitions" {

  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  routine_id   = data.bqutils_routine_parser.list_partitions.routine_id
  routine_type = data.bqutils_routine_parser.list_partitions.routine_type
  language     = data.bqutils_routine_parser.list_partitions.language
  description  = data.bqutils_routine_parser.list_partitions.description

  definition_body = data.bqutils_routine_parser.list_partitions.definition_body

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.list_partitions.arguments
    content {
      name      = arguments.value.name
      data_type = arguments.value.data_type
    }
  }
}
```

### Parsing SQL and creating a JavaScript FUNCTION in BigQuery

```terraform
data "bqutils_routine_parser" "parse_json_to_array" {

  sql = <<EOF
    CREATE OR REPLACE FUNCTION parse_json_to_array(json_str STRING)
    RETURNS ARRAY<STRING>
    LANGUAGE js AS r"""
      try {
        let parsed = JSON.parse(json_str);
        return parsed.tags || [];
      } catch (e) {
        return [];
      }
    """;
  EOF

  trim_body = true
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

resource "google_bigquery_routine" "parse_json_to_array" {

  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  routine_id   = data.bqutils_routine_parser.parse_json_to_array.routine_id
  routine_type = data.bqutils_routine_parser.parse_json_to_array.routine_type
  language     = data.bqutils_routine_parser.parse_json_to_array.language

  # Define the input parameters

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.parse_json_to_array.arguments
    content {
      name      = arguments.value.name
      data_type = arguments.value.data_type
    }
  }

  # Define the return data type (ARRAY of STRINGs)
  # return_type = "{\"typeKind\": \"ARRAY\", \"arrayElementType\": {\"typeKind\": \"STRING\"}}"
  return_type = data.bqutils_routine_parser.parse_json_to_array.return_type

  definition_body = data.bqutils_routine_parser.parse_json_to_array.definition_body
}
```

## Schema

### Required

- `sql` (String) Full CREATE statement SQL text.

### Optional

- `trim_body` (Boolean) Trim leading/trailing whitespace and empty lines from `definition_body`. Defaults to `true`.
- `trim_comments` (Boolean) Remove SQL comments from `definition_body`. Defaults to `false`.

### Read-Only

- `id` (String)
- `project` (String) Project from a three-part name, if present.
- `dataset_id` (String) Dataset from a qualified name, if present.
- `routine_id` (String)
- `routine_type` (String) `SCALAR_FUNCTION`, `TABLE_FUNCTION`, `PROCEDURE`, or `AGGREGATE_FUNCTION`.
- `definition_body` (String)
- `language` (String)
- `return_type` (String) StandardSqlDataType JSON.
- `return_table_type` (String) JSON for `RETURNS TABLE<...>`.
- `description` (String)
- `imported_libraries` (List of String)
- `determinism_level` (String)
- `data_governance_type` (String)
- `arguments` (Attributes List)
  - `name` (String)
  - `data_type` (String) StandardSqlDataType JSON.
  - `argument_kind` (String)
  - `mode` (String)
- `remote_function_options` (Attributes)
  - `connection` (String)
  - `endpoint` (String)
- `spark_options` (Attributes)
  - `raw` (String)
