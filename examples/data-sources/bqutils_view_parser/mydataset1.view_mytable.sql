CREATE OR REPLACE VIEW `mydataset1.view_mytable`
(
  id OPTIONS(description="The key code of the row"),
  name OPTIONS(description="The name of the event type"),
) OPTIONS(
  description="Example of a view using a foreign dataset authorization grant"
) AS
  SELECT id, name
  FROM mydataset2.mytable
