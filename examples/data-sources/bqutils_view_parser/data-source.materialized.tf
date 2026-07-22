# Parse a MATERIALIZED VIEW and create google_bigquery_table.

data "bqutils_view_parser" "materialized_view" {
  sql = <<EOF
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

  depends_on = [
    data.google_bigquery_dataset.mydataset
  ]
}
