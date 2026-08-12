CREATE OR REPLACE TABLE FUNCTION mydataset1.test_array_distinct(
    max_value INT64
) AS (
    WITH tab AS (
        SELECT 1 AS id, [1,2,3] AS items UNION ALL
        SELECT 2 AS id, [1,2]   AS items
    )
    SELECT t.id
        , mydataset1.array_distinct(ARRAY_CONCAT_AGG(t.items)) AS unique_items
     FROM tab AS t
    WHERE t.id <= max_value
    GROUP BY t.id
);
