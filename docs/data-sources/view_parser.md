---
page_title: "bqutils_view_parser Data Source - bqutils"
subcategory: ""
description: |-
  Parses BigQuery CREATE VIEW SQL for google_bigquery_table.
---

# bqutils_view_parser (Data Source)

Parses a BigQuery `CREATE VIEW` or `CREATE MATERIALIZED VIEW` statement.

## Example Usage

### Loading SQL from a file

```terraform
data "bqutils_view_parser" "example" {
  sql = file("${path.module}/view.sql")
}
```

### Parsing SQL and creating a VIEW in BigQuery

```terraform
data "bqutils_view_parser" "simple_view" {

  sql = <<EOF
    CREATE OR REPLACE VIEW  `mydataset`.my_simple_view
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
      FROM appfleet.INFORMATION_SCHEMA.TABLES;
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
    # The SQL query that defines the materialized view: After AS elment in SQL Syntax
    query = data.bqutils_view_parser.simple_view.query

    use_legacy_sql = false
  }
}
```

### Parsing SQL and creating a MATERIALIZED VIEW in BigQuery

```terraform
data "bqutils_view_parser" "materialized_view" {

  sql = <<EOF
    CREATE OR REPLACE MATERIALIZED VIEW `mydataset`.my_materialized_view
    PARTITION BY DATE(creation_time)
    CLUSTER BY custormer_name, order_id
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
        , custormer_name
        , delivery_type
        , creation_time
      FROM mydataset.orders AS o;
  EOF
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

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

# Ensure the view depends explicitly on the dataset existence

  depends_on = [
    data.google_bigquery_dataset.mydataset
  ]
}
```

## Schema

### Required

- `sql` (String) Full CREATE VIEW statement SQL text.

### Optional

- `trim_body` (Boolean) Trim leading/trailing whitespace and empty lines from `query`. Defaults to `true`.
- `trim_comments` (Boolean) Remove SQL comments from `query`. Defaults to `false`.

### Read-Only

- `id` (String)
- `project` (String)
- `dataset_id` (String)
- `table_id` (String)
- `query` (String)
- `description` (String)
- `friendly_name` (String)
- `labels` (Map of String)
- `is_materialized` (Boolean)
- `schema` (String) JSON schema from the view column list when present (types default to STRING when not specified in SQL).
- `enable_refresh` (Boolean)
- `allow_non_incremental_definition` (Boolean)
- `refresh_interval_ms` (Number) Converted from `refresh_interval_minutes` when present.
- `max_staleness` (String)
- `kms_key_name` (String)
- `partitioning_type` (String)
- `partitioning_field` (String)
- `clustering` (List of String)
