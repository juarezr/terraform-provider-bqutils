# Load Studio-friendly SQL that uses unqualified dataset.entity references.
data "bqutils_routine_parser" "test_array_distinct" {
  sql = file("${path.module}/mydataset1.test_array_distinct.sql")

  # Qualify foreign dataset refs for the Routines API without editing the SQL file.
  target_project = data.google_bigquery_dataset.mydataset.project

  trim_body = true
}

data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset1"
}

resource "google_bigquery_routine" "test_array_distinct" {
  project      = data.google_bigquery_dataset.mydataset.project
  dataset_id   = data.google_bigquery_dataset.mydataset.dataset_id
  routine_id   = data.bqutils_routine_parser.test_array_distinct.routine_id
  routine_type = data.bqutils_routine_parser.test_array_distinct.routine_type
  language     = data.bqutils_routine_parser.test_array_distinct.language

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.test_array_distinct.arguments
    content {
      name          = arguments.value.name
      argument_kind = arguments.value.argument_kind
      data_type     = arguments.value.data_type
    }
  }

  # Body is rewritten so mydataset1.array_distinct becomes `project`.`mydataset1`.`array_distinct`
  definition_body = data.bqutils_routine_parser.test_array_distinct.definition_body

  security_mode = "INVOKER"
}

# Grant authorized-routine access on every foreign dataset referenced in the SQL body.
resource "google_bigquery_dataset_access" "test_array_distinct" {
  for_each = toset(data.bqutils_routine_parser.test_array_distinct.dataset_references)

  dataset_id = each.key

  routine {
    project_id = google_bigquery_routine.test_array_distinct.project
    dataset_id = google_bigquery_routine.test_array_distinct.dataset_id
    routine_id = google_bigquery_routine.test_array_distinct.routine_id
  }

  lifecycle {
    replace_triggered_by = [
      google_bigquery_routine.test_array_distinct.definition_body,
      google_bigquery_routine.test_array_distinct.arguments
    ]
  }
}
