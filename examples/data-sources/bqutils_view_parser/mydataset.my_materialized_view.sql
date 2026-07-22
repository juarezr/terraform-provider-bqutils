    CREATE OR REPLACE MATERIALIZED VIEW `mydataset`.my_materialized_view
    PARTITION BY DATE(creation_time)
    CLUSTER BY customer_name, order_id
    OPTIONS(
      description="Materialized view created by Terraform",
      enable_refresh=TRUE,
      allow_non_incremental_definition=FALSE,
      refresh_interval_minutes=60,
      max_staleness=INTERVAL 90 MINUTE,
      kms_key_name="projects/1234567890/locations/global/keyRings/my-key-ring/cryptoKeys/my-key",
      labels=[("org_unit", "development")]
    ) AS
      SELECT order_id
        , customer_name
        , delivery_type
        , creation_time
      FROM mydataset.orders AS o;
