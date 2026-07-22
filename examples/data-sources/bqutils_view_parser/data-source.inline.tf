# Parse a CREATE VIEW from the inline SQL statement and expose its values in attributes.
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

  trim_indentation = true
}

# Gets the BigQuery dataset where the view is created.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Create the view in BigQuery using the attributes parsed from the SQL above.
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
