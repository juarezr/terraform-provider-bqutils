CREATE OR REPLACE FUNCTION mydataset1.myfunction3()
RETURNS INT64
AS (
  mydataset1.myfunction2() + 1
);
