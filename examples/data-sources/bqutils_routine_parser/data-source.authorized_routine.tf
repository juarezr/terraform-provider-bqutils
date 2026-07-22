# Load the routine SQL from a file in the same folder as the Terraform code.
data "bqutils_routine_parser" "list_tables" {
  sql = file("${path.module}/mydataset.list_tables.sql")

  trim_body = true
}

# Gets the BigQuery dataset where the routine is created.
data "google_bigquery_dataset" "mydataset1" {
  dataset_id = "mydataset1"
}

# Gets the BigQuery dataset where the routine reads from (authorize the routine on this dataset).
data "google_bigquery_dataset" "mydataset2" {
  dataset_id = "mydataset2"
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

# Grant authorized-routine access on mydataset2 after the routine is created/modified
# The lifecycle block triggers modification when the routine SQL content changes in
# the previous google_bigquery_routine resource.
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
