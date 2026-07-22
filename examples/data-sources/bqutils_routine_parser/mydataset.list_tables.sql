CREATE OR REPLACE TABLE FUNCTION mydataset1.list_tables
(
    table_name_filter STRING,
    max_results INT64
)
OPTIONS (
    description = 'Used to show tables in another dataset.'
) AS (
  SELECT
    t.table_name,
    t.table_type,
    t.creation_time,
    t.is_typed
    -- Notice that the dataset of the tables is not the same as the one of the function
  FROM `mydataset2`.INFORMATION_SCHEMA.TABLES AS t
  WHERE t.table_name LIKE CONCAT('%', table_name_filter, '%')
  QUALIFY ROW_NUMBER() OVER(ORDER BY t.table_name) <= max_results
);
