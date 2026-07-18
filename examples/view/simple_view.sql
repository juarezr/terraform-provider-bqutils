CREATE OR REPLACE VIEW IF NOT EXISTS `mydataset.my_simple_view`
(
  TABLE_SCHEMA OPTIONS(description="The schema of the table"),
  TABLE_NAME OPTIONS(description="The name of the table")
) OPTIONS(
  description="Simple view created by Terraform"
) AS
  SELECT TABLE_SCHEMA, TABLE_NAME
  FROM mydataset.INFORMATION_SCHEMA.TABLES;
