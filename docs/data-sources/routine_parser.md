---
page_title: "bqutils_routine_parser Data Source - bqutils"
subcategory: ""
description: |-
  Parses a BigQuery CREATE SQL statement from a string and supplies its parts as attributes for google_bigquery_routine. Main use case: create and update BigQuery routines from SQL files with Terraform.
---

# bqutils_routine_parser

This data source parses a BigQuery CREATE SQL statement from a string and supplies its parts as attributes that can be used to fill the arguments of the BigQuery Terraform `google_bigquery_routine` resource.

## Caveats

- It can handle the `CREATE FUNCTION`, `CREATE TABLE FUNCTION`, `CREATE PROCEDURE`, or `CREATE AGGREGATE FUNCTION` SQL statements.
- A `CREATE TEMPORARY FUNCTION` SQL statement will produce an error because the Terraform state requires the object to exist and would fail in this case.

## Example Usage

### Parsing SQL and creating a TABLE FUNCTION in BigQuery

In this example, the TABLE FUNCTION is created in BigQuery using an inline SQL statement inside the Terraform code:

```terraform
# Parses a TABLE FUNCTION from inline SQL for google_bigquery_routine.

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

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
      WHERE t.partition_id != '__NULL__'
        AND t.table_name LIKE CONCAT('%', table_name_filter, '%')
    );
  EOF

  trim_comments = false
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

### Loading SQL from a file to create a BigQuery routine

In this example, the TABLE FUNCTION is created in BigQuery using SQL loaded from the following file in the Terraform module folder:

```sql
CREATE AGGREGATE FUNCTION mydataset.scaled_sum
(
  dividend FLOAT64,
  divisor FLOAT64 NOT AGGREGATE
) RETURNS FLOAT64 AS (
  SUM(dividend) / divisor
);
```

The Terraform code to create the TABLE FUNCTION is:

```terraform
# Load SQL from a file (place mydataset.scaled_sum.sql next to this module).

data "bqutils_routine_parser" "scaled_sum" {
  sql = file("${path.module}/mydataset.scaled_sum.sql")
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

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

### Creating a JavaScript FUNCTION in BigQuery

```terraform
# Parses a JavaScript FUNCTION from inline SQL.

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

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

resource "google_bigquery_routine" "parse_json_to_array" {
  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  routine_id   = data.bqutils_routine_parser.parse_json_to_array.routine_id
  routine_type = data.bqutils_routine_parser.parse_json_to_array.routine_type
  language     = data.bqutils_routine_parser.parse_json_to_array.language

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.parse_json_to_array.arguments
    content {
      name      = arguments.value.name
      data_type = arguments.value.data_type
    }
  }

  return_type     = data.bqutils_routine_parser.parse_json_to_array.return_type
  definition_body = data.bqutils_routine_parser.parse_json_to_array.definition_body
}
```

### Applying access permissions to routines by using BigQuery authorized routines

When Terraform updates a BigQuery routine, any authorized-routine grants on other datasets are dropped and must be re-granted.

To keep access on the routine, use a `google_bigquery_dataset_access` resource and configure its `lifecycle.replace_triggered_by` argument so that when the routine body, return type, or arguments are modified, Terraform also reapplies the routine grants.

Example SQL file to create a routine (placed next to the Terraform code):

```sql
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
    t.is_typed
    -- Notice that the dataset of the tables is not the same as the one of the function
  FROM `mydataset2`.INFORMATION_SCHEMA.TABLES AS t
  WHERE t.table_name LIKE CONCAT('%', table_name_filter, '%')
  QUALIFY ROW_NUMBER() OVER(ORDER BY t.table_name) <= max_results
);
```

Terraform code that creates the routine and grants access to the dataset:

```terraform
# Dataset where the routine is created.

data "google_bigquery_dataset" "mydataset1" {
  dataset_id = "mydataset1"
}

# Dataset the routine reads from (authorize the routine on this dataset).

data "google_bigquery_dataset" "mydataset2" {
  dataset_id = "mydataset2"
}

# Load the routine SQL from a file in the same folder as the Terraform code.

data "bqutils_routine_parser" "list_tables" {
  sql = file("${path.module}/mydataset.list_tables.sql")
}

# Create the routine in BigQuery using the attributes parsed from the SQL file.

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

# Grant authorized-routine access on mydataset2 when the SQL text of the routine changes.

resource "google_bigquery_dataset_access" "list_tables" {
  dataset_id = data.google_bigquery_dataset.mydataset2.dataset_id

  routine {
    project_id = google_bigquery_routine.list_tables.project
    dataset_id = google_bigquery_routine.list_tables.dataset_id
    routine_id = google_bigquery_routine.list_tables.routine_id
  }

  lifecycle {
    replace_triggered_by = [
      google_bigquery_routine.list_tables.definition_body,
      google_bigquery_routine.list_tables.return_type,
      google_bigquery_routine.list_tables.arguments
    ]
  }

  depends_on = [
    data.google_bigquery_dataset.mydataset1,
    data.google_bigquery_dataset.mydataset2,
    data.bqutils_routine_parser.list_tables
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `sql` (String) Full CREATE statement SQL text.

### Optional

- `trim_body` (Boolean) Trim leading/trailing whitespace and empty lines from definition_body. Defaults to true.
- `trim_comments` (Boolean) Remove SQL comments from definition_body. Defaults to false.

### Read-Only

- `arguments` (Attributes List) Routine arguments parsed from the CREATE statement. (see [below for nested schema](#nestedatt--arguments))
- `data_governance_type` (String) If set to DATA_MASKING, the function is validated and made available as a masking function.
- `dataset_id` (String) Routine dataset parsed from the SQL statement, if present.
- `definition_body` (String) The body of the routine. For functions, this is the expression in the AS clause. If language=SQL, it is the substring inside (but excluding) the parentheses.
- `description` (String) Description parsed from the SQL OPTIONS clause, if present.
- `determinism_level` (String) Determinism level of a JavaScript UDF if defined. Possible values: DETERMINISM_LEVEL_UNSPECIFIED, DETERMINISTIC, NOT_DETERMINISTIC.
- `id` (String) Synthetic id matching google_bigquery_routine: projects/<project>/datasets/<dataset_id>/routines/<routine_id>. Missing project or dataset segments use the placeholder "any" (not exposed on project/dataset_id).
- `imported_libraries` (List of String) If language is JAVASCRIPT, paths of imported JavaScript libraries.
- `language` (String) The language of the routine. Possible values: SQL, JAVASCRIPT, PYTHON, JAVA, SCALA.
- `project` (String) Project parsed from a three-part name, if present.
- `remote_function_options` (Attributes) Remote function options when present. (see [below for nested schema](#nestedatt--remote_function_options))
- `return_table_type` (String) JSON for RETURNS TABLE<...> when present (table-valued functions).
- `return_type` (String) StandardSqlDataType as JSON schema for the function return type when present.
- `routine_id` (String) Name of the routine parsed from the SQL statement.
- `routine_type` (String) SCALAR_FUNCTION, TABLE_VALUED_FUNCTION, PROCEDURE, or AGGREGATE_FUNCTION.
- `spark_options` (Attributes) If language is PYTHON, JAVA, or SCALA, options for a Spark stored procedure. (see [below for nested schema](#nestedatt--spark_options))

<a id="nestedatt--arguments"></a>
### Nested Schema for `arguments`

Read-Only:

- `argument_kind` (String) Default FIXED_TYPE. Possible values: FIXED_TYPE, ANY_TYPE.
- `data_type` (String) StandardSqlDataType JSON schema for the data type.
- `is_aggregate` (Boolean) For CREATE AGGREGATE FUNCTION parameters: false when the SQL includes NOT AGGREGATE, true for aggregate parameters. Null for non-UDAF routines. google_bigquery_routine does not expose this field yet.
- `mode` (String) Argument mode for procedures when present (IN, OUT, INOUT).
- `name` (String) The name of this argument.


<a id="nestedatt--remote_function_options"></a>
### Nested Schema for `remote_function_options`

Read-Only:

- `connection` (String) Connection resource name for the remote function.
- `endpoint` (String) Remote function endpoint URL.


<a id="nestedatt--spark_options"></a>
### Nested Schema for `spark_options`

Read-Only:

- `raw` (String) Raw spark options JSON when present.
