# Load the MATERIALIZED VIEW SQL from a file in the same folder as the Terraform code.
data "bqutils_view_parser" "example" {
  sql = file("${path.module}/mydataset.my_materialized_view.sql")

  trim_body = true
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
}
