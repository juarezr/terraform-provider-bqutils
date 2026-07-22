CREATE OR REPLACE VIEW `mydataset.my_simple_view`
(
  table_schema OPTIONS(description="The schema of the table"),
  table_name OPTIONS(description="The name of the table")
) OPTIONS(
  description="Simple view created by Terraform"
) AS
  SELECT table_schema, table_name
  FROM mydataset.INFORMATION_SCHEMA.TABLES;
