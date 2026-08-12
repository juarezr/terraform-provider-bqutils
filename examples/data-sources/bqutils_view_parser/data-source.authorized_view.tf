data "bqutils_view_parser" "view_mytable" {
  sql = file("${path.module}/mydataset1.view_mytable.sql")
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset1"
}

resource "google_bigquery_table" "view_mytable" {
  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  table_id      = data.bqutils_view_parser.view_mytable.table_id
  friendly_name = data.bqutils_view_parser.view_mytable.friendly_name
  description   = data.bqutils_view_parser.view_mytable.description
  labels        = data.bqutils_view_parser.view_mytable.labels

  deletion_protection = false

  view {
    query          = data.bqutils_view_parser.view_mytable.query
    use_legacy_sql = false
  }
}

# Grant authorized-view access on each foreign dataset referenced in the view query.
resource "google_bigquery_dataset_access" "view_mytable" {
  for_each = toset(data.bqutils_view_parser.view_mytable.dataset_references)

  dataset_id = each.key

  view {
    project_id = google_bigquery_table.view_mytable.project
    dataset_id = google_bigquery_table.view_mytable.dataset_id
    table_id   = google_bigquery_table.view_mytable.table_id
  }

  lifecycle {
    replace_triggered_by = [
      google_bigquery_table.view_mytable.view
    ]
  }
}
