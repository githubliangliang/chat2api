-- [sqlite-converted] from PostgreSQL migration: 203_channel_monitor_v2_default_ignore_and_cache.sql
UPDATE channel_monitor_v2_config
SET ignored_error_categories = '["authentication","client_cancelled","content_policy","context_limit","group_access","model_unsupported","not_found","quota_or_balance"]'
WHERE id = 1
  AND (
    ignored_error_categories IS NULL
    OR ignored_error_categories = ''
    OR ignored_error_categories = '[]'
    OR ignored_error_categories = '{}'
  );

UPDATE channel_monitor_v2_config
SET health_thresholds = json_set(
        json_set(
            COALESCE(NULLIF(health_thresholds, ''), '{}'),
            '$.warning_cache_rate', 0.85
        ),
        '$.critical_cache_rate', 0.60
    )
WHERE id = 1
  AND COALESCE(json_extract(COALESCE(NULLIF(health_thresholds, ''), '{}'), '$.warning_cache_rate'), 0) = 0
  AND COALESCE(json_extract(COALESCE(NULLIF(health_thresholds, ''), '{}'), '$.critical_cache_rate'), 0) = 0;
