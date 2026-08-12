-- [sqlite-converted] from PostgreSQL migration: 205_channel_monitor_v2_reset_factory_cache_thresholds.sql
UPDATE channel_monitor_v2_config
SET health_thresholds = json_set(
        json_set(
            COALESCE(NULLIF(health_thresholds, ''), '{}'),
            '$.warning_cache_rate', 0
        ),
        '$.critical_cache_rate', 0
    )
WHERE id = 1
  AND COALESCE(json_extract(COALESCE(NULLIF(health_thresholds, ''), '{}'), '$.warning_cache_rate'), 0) = 0.85
  AND COALESCE(json_extract(COALESCE(NULLIF(health_thresholds, ''), '{}'), '$.critical_cache_rate'), 0) = 0.60
  AND version = 1
  AND updated_by IS NULL;
