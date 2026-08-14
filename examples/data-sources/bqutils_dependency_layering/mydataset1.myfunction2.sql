CREATE OR REPLACE FUNCTION mydataset1.myfunction2()
RETURNS INT64
AS (
  mydataset1.myfunction1() + 1
);
