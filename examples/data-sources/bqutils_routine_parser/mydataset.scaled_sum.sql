CREATE AGGREGATE FUNCTION mydataset.scaled_sum
(
  dividend FLOAT64,
  divisor FLOAT64 NOT AGGREGATE
) RETURNS FLOAT64 AS (
  SUM(dividend) / divisor
);
