# Parses a JavaScript FUNCTION from inline SQL.
data "bqutils_routine_parser" "parse_json_to_array" {
  sql = <<EOF
    CREATE OR REPLACE FUNCTION parse_json_to_array(json_str STRING)
    RETURNS ARRAY<STRING>
    LANGUAGE js AS r"""
      try {
        let parsed = JSON.parse(json_str);
        return parsed || [];
      } catch (e) {
        return [e.message];
      }
    """;
  EOF

  trim_indentation = true
}

# Gets the BigQuery dataset where the routine is created.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Create the routine in BigQuery using the attributes parsed from the SQL above.
resource "google_bigquery_routine" "parse_json_to_array" {
  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  routine_id   = data.bqutils_routine_parser.parse_json_to_array.routine_id
  routine_type = data.bqutils_routine_parser.parse_json_to_array.routine_type
  language     = data.bqutils_routine_parser.parse_json_to_array.language

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.parse_json_to_array.arguments
    content {
      name      = arguments.value.name
      data_type = arguments.value.data_type
    }
  }

  return_type     = data.bqutils_routine_parser.parse_json_to_array.return_type
  definition_body = data.bqutils_routine_parser.parse_json_to_array.definition_body
}
