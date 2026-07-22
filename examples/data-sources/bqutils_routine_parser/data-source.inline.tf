# Gets the BigQuery dataset where the routine is created.
data "google_bigquery_dataset" "mydataset" {
  dataset_id = "mydataset"
}

# Parses a TABLE FUNCTION from inline SQL mixed/interpolated with values from other datasources.
data "bqutils_routine_parser" "list_partitions" {
  sql = <<EOF
    CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
    (
        table_name_filter STRING
    )
    OPTIONS (
      description = 'Used to evaluate partition details in a partitioned table inside the dataset.'
    ) AS (
      SELECT
        t.table_schema,
        t.table_name,
        t.partition_id,
        t.total_rows,
        t.last_modified_time AS last_modified
      FROM `${data.google_bigquery_dataset.mydataset.project}`.`mydataset`.INFORMATION_SCHEMA.PARTITIONS AS t
      WHERE t.partition_id != '__NULL__'
        AND t.table_name LIKE CONCAT('%', table_name_filter, '%')
    );
  EOF

  trim_indentation = true
}

# Create the routine in BigQuery using the attributes parsed from the SQL above.
resource "google_bigquery_routine" "list_partitions" {
  dataset_id = data.google_bigquery_dataset.mydataset.dataset_id

  routine_id   = data.bqutils_routine_parser.list_partitions.routine_id
  routine_type = data.bqutils_routine_parser.list_partitions.routine_type
  language     = data.bqutils_routine_parser.list_partitions.language
  description  = data.bqutils_routine_parser.list_partitions.description

  definition_body = data.bqutils_routine_parser.list_partitions.definition_body

  dynamic "arguments" {
    for_each = data.bqutils_routine_parser.list_partitions.arguments
    content {
      name      = arguments.value.name
      data_type = arguments.value.data_type
    }
  }
}
