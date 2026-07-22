---
page_title: "bqutils_routine_parser Data Source - bqutils"
subcategory: ""
description: |-
  Parses BigQuery CREATE FUNCTION / PROCEDURE SQL for google_bigquery_routine.
---

# bqutils_routine_parser (Data Source)

Parses a BigQuery CREATE SQL statement from a string and fills its attributes for use with the BigQuery Terraform `google_bigquery_routine` resource.

Its main use case is creating and updating BigQuery routines with Terraform, loading them from SQL files.

Beneficts:

- It removes the need to slice the routine source code into parts like arguments, body, and return type for filling the `google_bigquery_routine` resource.
- It lets you use the same SQL source file to provision with Terraform or to execute in the BigQuery Console.
- It takes care of applying only the routines that were modified and must be applied in another environment without tracking them manually. Terraform will only modify routines that are modified.
- It allows you to keep your routines always up to date in all BigQuery environments you have.
- You can also use git for versioning the Terraform and SQL files, so you can track the impact of changes and bugs introduced in your routines.

Restriction:

- It can handle the `CREATE FUNCTION`, `CREATE TABLE FUNCTION`, `CREATE PROCEDURE`, or `CREATE AGGREGATE FUNCTION` SQL statements.
- A `CREATE TEMPORARY FUNCTION` SQL statement will produce an error because the Terraform state requires the object to exist and would fail in this case.
- Terraform will "own" your routine source code. Any changes applied outside the plan/apply cycle will be overwritten.

## Example Usage

### Parsing SQL and creating a TABLE FUNCTION in BigQuery

In this example, the attributes of a TABLE FUNCTION defined with SQL are parsed from a inline SQL Statement inside the Terraform code:

```javascript
# Terraform datasource pointing to the BigQuery dataset where the routine will be created/updated.

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Parses the TABLE FUNCTION from the inline SQL statement for filling arguments in google_bigquery_routine.
# Any change in the content of the sql argument will trigger the update of the function in BigQuery.

data "bqutils_routine_parser" "list_partitions" {

  sql = <<EOF
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'Used to evaluate partition details in a partitioned table inside the dataset.'
    ) AS (
      SELECT
        t.table_schema,
        t.table_name,
        t.partition_id,
        t.total_rows,
        t.last_modified_time AS last_modified
      FROM `${data.google_bigquery_dataset.mydataset.project}`.`mydataset`.INFORMATION_SCHEMA.PARTITIONS AS t
      WHERE t.partition_id != '__NULL__'  -- Ignore partitions without rows
        AND t.table_name LIKE CONCAT('%', table_name_filter, '%')
    );
  EOF

  trim_comments = false
}

# Creates the TABLE FUNCTION in the BigQuery dataset using the parsed attributes 
# obtained from the routine_parser data source above.

resource "google_bigquery_routine" "list_partitions" {

  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id
  # dataset_id = data.bqutils_routine_parser.list_partitions.dataset_id # also works in this case

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

This BigQuery javascript FUNCTION will be create from a inline SQL statement inside terraform code:

```javascript
# Parses the javascript FUNCTION from the inline SQL statement for filling arguments in google_bigquery_routine.

data "bqutils_routine_parser" "parse_json_to_array" {

  sql = <<EOF
    CREATE OR REPLACE FUNCTION parse_json_to_array(json_str STRING)
    RETURNS ARRAY<STRING>
    LANGUAGE js AS r"""
      try {
        let parsed = JSON.parse(json_str);
        return parsed || [];
      } catch (e) {
        return [e.message];
      }
    """;
  EOF

  trim_body = true
}

# Terraform datasource pointing to the BigQuery dataset where the routine will be created/updated.

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Creates the javascript FUNCTION in the BigQuery dataset using the parsed attributes 
# obtained from the routine_parser data source above.

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

Usage example:

```sql
WITH raw_data AS (
  SELECT '["production", "web"]' AS json_col UNION ALL
  SELECT '["test"]' AS json_col UNION ALL
  SELECT 'invalid json string' AS json_col
)
SELECT 
  json_col,
  mydataset.parse_json_to_array(json_col) AS parsed
FROM raw_data;
```

### Loading SQL from a file to create a BigQuery routine

Take the following SQL CREATE statement stored in the file `mydataset.scaled_sum.sql`:

```sql
CREATE AGGREGATE FUNCTION mydataset.scaled_sum
(
  dividend FLOAT64,
  divisor FLOAT64 NOT AGGREGATE
) RETURNS FLOAT64 AS (
  SUM(dividend) / divisor
);
```

You can load the SQL statement text from the file using the following Terraform code:

```javascript
data "bqutils_routine_parser" "scaled_sum" {

  sql = file("${path.module}/mydataset.scaled_sum.sql")
}

# Terraform datasource pointing to the BigQuery dataset where the routine will be created/updated.

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Parses the AGGREGATE FUNCTION loaded from the file for filling arguments in google_bigquery_routine.

resource "google_bigquery_routine" "scaled_sum" {

  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  routine_id   = data.bqutils_routine_parser.scaled_sum.routine_id
  routine_type = data.bqutils_routine_parser.scaled_sum.routine_type
  language     = data.bqutils_routine_parser.scaled_sum.language

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.scaled_sum.arguments
    content {
      name          = arguments.value.name
      argument_kind = arguments.value.argument_kind
      data_type     = arguments.value.data_type
    }
  }

  return_type     = data.bqutils_routine_parser.scaled_sum.return_type
  definition_body = data.bqutils_routine_parser.scaled_sum.definition_body
}
```

Usage example:

```sql
SELECT scaled_sum(col1, 2) AS sum_value
  FROM (
    SELECT 1 AS col1 UNION ALL
    SELECT 3 AS col1 UNION ALL
    SELECT 5 AS col1
 );
```

### Sync access permissions using Authorized Routines

Every time that changes are made to a BigQuery routine and applied using Terraform, the permissions are removed during the process. This can cause access problems for users and other applications if you don't grant the permissions again.

To deal with this issue, you can automate the granting of permissions using the BigQuery authorized routines mechanism in tandem with Terraform.

Check the Terraform code below to understand how to synchronize the parsing, re-creation, and grant process using Terraform BigQuery resources and the `bqutils_routine_parser` data source.

```javascript
# Terraform datasource pointing to the BigQuery dataset where the routine will be created/updated.

data "google_bigquery_dataset" "mydataset1" {
  dataset_id = "mydataset1"
}

# Terraform datasource pointing to the BigQuery dataset where the routine reads tables or views.

data "google_bigquery_dataset" "mydataset2" {
  dataset_id = "mydataset2"
}

# Parses the javascript FUNCTION from the inline SQL statement for filling arguments in google_bigquery_routine.

data "bqutils_routine_parser" "list_tables" {

  sql = <<EOF
    CREATE OR REPLACE TABLE FUNCTION mydataset1.list_tables
    (
        table_name_filter STRING,
        max_results INT64
    )
    OPTIONS (
        description = 'Used to show tables in another dataset.'
    ) AS (
      SELECT
        t.table_name,
        t.table_type,
        t.creation_time,
        t.is_insertable_into,
        t.is_typed,
        --> Notice that the dataset of the tables is not the same as the one of the function
      FROM `mydataset2`.INFORMATION_SCHEMA.TABLES AS t
      WHERE t.table_name LIKE CONCAT('%', table_name_filter, '%')
      QUALIFY ROW_NUMBER() OVER(ORDER BY t.table_name) <= max_results
    );
  EOF
}

resource "google_bigquery_routine" "list_tables" {

  dataset_id = data.google_bigquery_dataset.mydataset1.dataset_id

  routine_id   = data.bqutils_routine_parser.list_tables.routine_id
  routine_type = data.bqutils_routine_parser.list_tables.routine_type
  language     = data.bqutils_routine_parser.list_tables.language

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.list_tables.arguments
    content {
      name          = arguments.value.name
      argument_kind = arguments.value.argument_kind
      data_type     = arguments.value.data_type
    }
  }

  return_type     = data.bqutils_routine_parser.list_tables.return_type
  definition_body = data.bqutils_routine_parser.list_tables.definition_body
}

# Grants access to the list_tables routine to the mydataset2 dataset.

resource "google_bigquery_dataset_access" "list_tables" {

  dataset_id = data.google_bigquery_dataset.mydataset2.dataset_id

  routine {
    project_id = data.google_bigquery_dataset.mydataset1.dataset_id
    dataset_id = google_bigquery_routine.list_tables.dataset_id
    routine_id = google_bigquery_routine.list_tables.routine_id
  }

  # Trigger the replacement of the permission if the definition_body, return_type,
  # or arguments of the routine changes.
  lifecycle {
    replace_triggered_by = [
      google_bigquery_routine.list_tables.definition_body,
      google_bigquery_routine.list_tables.return_type,
      google_bigquery_routine.list_tables.arguments
    ]
  }
}

```

Usage example:

```sql
SELECT *
  FROM mydataset1.list_tables('mytable', 5);
```

## Schema

### Required

- `sql` (String) Full CREATE statement SQL text.

### Optional

- `trim_body` (Boolean) Trim leading/trailing whitespace and empty lines from `definition_body`. Defaults to `true`.
- `trim_comments` (Boolean) Remove SQL comments from `definition_body`. Defaults to `false`.

### Read-Only

- `id` (String)
- `project` (String) Project parsed from a three-part name, if present.
- `dataset_id` (String) Routine dataset parsed from the SQL statement, if present.
- `routine_id` (String) - Name of the routine parsed from the SQL statement.
- `routine_type` (String) `SCALAR_FUNCTION`, `TABLE_VALUED_FUNCTION`, `PROCEDURE`, or `AGGREGATE_FUNCTION`.
- `definition_body` (String) - The body of the routine. For functions, this is the expression in the AS clause. If language=SQL, it is the substring inside (but excluding) the parentheses.
- `language` (String) - (Optional) The language of the routine. Possible values are: `SQL`, `JAVASCRIPT`, `PYTHON`, `JAVA`, `SCALA`.
- `return_type` (String) - (Optional) Standard `SqlDataType` as JSON schema. Can be set only if routineType = "TABLE_VALUED_FUNCTION". If absent, the return table type is inferred from definitionBody at query time in each query that references this routine. If present, then the columns in the evaluated table result will be cast to match the column types specified in the return table type, at query time.
- `return_table_type` (String) JSON for `RETURNS TABLE<...>`.
- `description` (String) - Description parsed from the SQL statement.
- `imported_libraries` (List of String) - If language = "JAVASCRIPT", this attribute stores the path of the imported JAVASCRIPT libraries.
- `determinism_level` (String) - (Optional) The determinism level of the JavaScript UDF if defined. Possible values are: `DETERMINISM_LEVEL_UNSPECIFIED`, `DETERMINISTIC`, `NOT_DETERMINISTIC`.
- `data_governance_type` (String) - (Optional) If set to `DATA_MASKING`, the function is validated and made available as a masking function.
- `arguments` (Attributes List)
  - `name` (String) - (Optional) The name of this argument. Can be absent for function return argument.
  - `data_type` (String) Standard `SqlDataType` JSON schema for the data type.
  - `argument_kind` (String) - Default value is `FIXED_TYPE`. Possible values are:`FIXED_TYPE`,`ANY_TYPE`.
  - `mode` (String)
  - `is_aggregate` (Boolean) - For `CREATE AGGREGATE FUNCTION` parameters: `false` when the SQL includes `NOT AGGREGATE`, `true` for aggregate parameters. Null for non-UDAF routines. `google_bigquery_routine` does not expose this field yet.
- `remote_function_options` (Attributes)
  - `connection` (String)
  - `endpoint` (String)
- `spark_options` (Attributes) - (Optional) Optional. If language is one of `PYTHON`, `JAVA`, `SCALA`, this attribute stores the options for spark stored procedure.
  - `raw` (String)
