# Load the VIEW SQL from a file in the same folder as the Terraform code.
data "bqutils_view_parser" "my_simple_view" {
  sql = file("${path.module}/mydataset.my_simple_view.sql")
}

# Get the BigQuery dataset to create the MATERIALIZED VIEW in.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Create the VIEW in BigQuery using the attributes parsed from the SQL file.
resource "google_bigquery_table" "my_simple_view" {

  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  table_id      = data.bqutils_view_parser.my_simple_view.table_id
  friendly_name = data.bqutils_view_parser.my_simple_view.friendly_name
  description   = data.bqutils_view_parser.my_simple_view.description
  labels        = data.bqutils_view_parser.my_simple_view.labels

  deletion_protection = false

  view {
    # The SQL query that defines the materialized view: After the AS element in SQL Syntax
    query = data.bqutils_view_parser.my_simple_view.query

    use_legacy_sql = false
  }
}
