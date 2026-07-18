CREATE OR REPLACE TABLE FUNCTION mydataset.list_partitions
(
    table_name_filter STRING
)
OPTIONS (
  description = 'Used to evaluate partition details in a partitioned table.'
) AS (
  SELECT
    table_schema,
    table_name,
    partition_id,
    total_rows
  FROM `event1`.INFORMATION_SCHEMA.PARTITIONS
  WHERE partition_id != '__NULL__'
    AND table_name = table_name_filter
);
