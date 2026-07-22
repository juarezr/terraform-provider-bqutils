---
page_title: "bqutils_view_parser Data Source - bqutils"
subcategory: ""
description: |-
  Parses a BigQuery CREATE VIEW or CREATE MATERIALIZED VIEW statement and exposes attributes for google_bigquery_table.
---

# bqutils_view_parser (Data Source)

Parses a BigQuery `CREATE VIEW` or `CREATE MATERIALIZED VIEW` statement.

## Example Usage

### Loading SQL from a file to create a BigQuery view

In this example, the VIEW is created in BigQuery using an SQL loaded from the following file in the Terraform module folder:

```sql
CREATE OR REPLACE VIEW `mydataset.my_simple_view`
(
  table_schema OPTIONS(description="The schema of the table"),
  table_name OPTIONS(description="The name of the table")
) OPTIONS(
  description="Simple view created by Terraform"
) AS
  SELECT table_schema, table_name
  FROM mydataset.INFORMATION_SCHEMA.TABLES;
```

The Terraform code to create the VIEW is:

```terraform
# Load the VIEW SQL from a file in the same folder as the Terraform code.

data "bqutils_view_parser" "example" {
  sql = file("${path.module}/mydataset.my_simple_view.sql")
}
```

### Parsing SQL and creating a VIEW in BigQuery

```terraform
# Parse a VIEW and create google_bigquery_table.

data "bqutils_view_parser" "simple_view" {
  sql = <<EOF
    CREATE OR REPLACE VIEW `mydataset`.my_simple_view
    (
      table_schema       OPTIONS(description="The schema of the table"),
      table_name         OPTIONS(description="The name of the table"),
      creation_time      OPTIONS(description="The creation time of the table"),
      table_type         OPTIONS(description="The type of the table"),
      managed_table_type OPTIONS(description="The managed type of the table")
    ) OPTIONS(
      description="Simple view created by Terraform"
    ) AS
      SELECT table_schema
        , table_name
        , creation_time
        , table_type
        , managed_table_type
      FROM mydataset.INFORMATION_SCHEMA.TABLES;
  EOF
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

resource "google_bigquery_table" "simple_view" {
  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  table_id      = data.bqutils_view_parser.simple_view.table_id
  friendly_name = data.bqutils_view_parser.simple_view.friendly_name
  description   = data.bqutils_view_parser.simple_view.description
  labels        = data.bqutils_view_parser.simple_view.labels

  deletion_protection = false

  view {
    # SQL query that defines the view (after AS in the SQL syntax)
    query          = data.bqutils_view_parser.simple_view.query
    use_legacy_sql = false
  }
}
```

### Creating a MATERIALIZED VIEW in BigQuery

In this example, the VIEW is created in BigQuery using an SQL loaded from the following file in the Terraform module folder:

```sql
CREATE OR REPLACE MATERIALIZED VIEW `mydataset`.my_materialized_view
    PARTITION BY DATE(creation_time)
    CLUSTER BY customer_name, order_id
    OPTIONS(
      description="Materialized view created by Terraform",
      enable_refresh=TRUE,
      allow_non_incremental_definition=FALSE,
      refresh_interval_minutes=60,
      max_staleness=INTERVAL 90 MINUTE,
      kms_key_name="projects/1234567890/locations/global/keyRings/my-key-ring/cryptoKeys/my-key",
      labels=[("org_unit", "development")]
    ) AS
      SELECT order_id
        , customer_name
        , delivery_type
        , creation_time
      FROM mydataset.orders AS o;
```

The Terraform code to create the VIEW is:

```terraform
# Load the MATERIALIZED VIEW SQL from a file in the same folder as the Terraform code.

data "bqutils_view_parser" "example" {
  sql = file("${path.module}/mydataset.my_materialized_view.sql")
}

# Get the BigQuery dataset to create the MATERIALIZED VIEW in.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Create the MATERIALIZED VIEW in BigQuery using the attributes parsed from the SQL file.

resource "google_bigquery_table" "materialized_view" {
  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  table_id      = data.bqutils_view_parser.materialized_view.table_id
  friendly_name = data.bqutils_view_parser.materialized_view.friendly_name
  description   = data.bqutils_view_parser.materialized_view.description
  max_staleness = data.bqutils_view_parser.materialized_view.max_staleness
  labels        = data.bqutils_view_parser.materialized_view.labels

  deletion_protection = false

  materialized_view {
    query = data.bqutils_view_parser.materialized_view.query

    enable_refresh      = data.bqutils_view_parser.materialized_view.enable_refresh
    refresh_interval_ms = data.bqutils_view_parser.materialized_view.refresh_interval_ms

    allow_non_incremental_definition = data.bqutils_view_parser.materialized_view.allow_non_incremental_definition
  }

  time_partitioning {
    type  = data.bqutils_view_parser.materialized_view.partitioning_type
    field = data.bqutils_view_parser.materialized_view.partitioning_field
  }

  clustering = data.bqutils_view_parser.materialized_view.clustering

  encryption_configuration {
    kms_key_name = data.bqutils_view_parser.materialized_view.kms_key_name
  }

  depends_on = [
    data.google_bigquery_dataset.mydataset
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `sql` (String) Full CREATE VIEW statement SQL text.

### Optional

- `trim_body` (Boolean) Trim leading/trailing whitespace and empty lines from query. Defaults to true.
- `trim_comments` (Boolean) Remove SQL comments from query. Defaults to false.

### Read-Only

- `allow_non_incremental_definition` (Boolean) Materialized view allow_non_incremental_definition option when present.
- `clustering` (List of String) Clustering columns from CLUSTER BY when present.
- `dataset_id` (String) Dataset parsed from the SQL statement, if present.
- `description` (String) Description from OPTIONS, if present.
- `enable_refresh` (Boolean) Materialized view enable_refresh option when present.
- `friendly_name` (String) Friendly name from OPTIONS, if present.
- `id` (String) Synthetic id matching google_bigquery_table: projects/<project>/datasets/<dataset_id>/tables/<table_id>. Missing project or dataset segments use the placeholder "any" (not exposed on project/dataset_id).
- `is_materialized` (Boolean) True when the statement is CREATE MATERIALIZED VIEW.
- `kms_key_name` (String) KMS key name from OPTIONS, if present.
- `labels` (Map of String) Labels from OPTIONS, if present.
- `max_staleness` (String) IntervalValue encoding (Y-M D H:M:S) for google_bigquery_table.max_staleness. SQL INTERVAL options such as INTERVAL 90 MINUTE or INTERVAL "4:0:0" HOUR TO SECOND are converted automatically.
- `partitioning_field` (String) Partitioning field derived from PARTITION BY when present.
- `partitioning_type` (String) Time partitioning type derived from PARTITION BY when present.
- `project` (String) Project parsed from a three-part view name, if present.
- `query` (String) View query body after AS.
- `refresh_interval_ms` (Number) Converted from refresh_interval_minutes when present.
- `schema` (String) JSON schema from the view column list when present (types default to STRING when not specified in SQL).
- `table_id` (String) Table/view id parsed from the SQL statement.
