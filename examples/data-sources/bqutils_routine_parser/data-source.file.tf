# Load the AGGREGATE FUNCTION SQL from a file in the same folder as the Terraform code.
data "bqutils_routine_parser" "scaled_sum" {
  sql = file("${path.module}/mydataset.scaled_sum.sql")

  trim_body = true
}

# Gets the BigQuery dataset where the routine is created.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Create the routine in BigQuery using the attributes parsed from the SQL file.
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
