-- [sqlite-converted] from PostgreSQL migration: 198_channel_monitor_v2_health_thresholds.sql
-- Configurable Channel Monitor V2 health scoring thresholds.
-- Defaults are intentionally tolerant: small amounts of real traffic errors or
-- zero cache rate should not immediately paint the user page as "abnormal".
ALTER TABLE channel_monitor_v2_config ADD COLUMN health_thresholds TEXT NOT NULL DEFAULT '{
      "minimum_sample": 50,
      "warning_error_rate": 0.05,
      "critical_error_rate": 0.20,
      "target_ttft_ms": 3000,
      "warning_ttft_ms": 8000,
      "critical_ttft_ms": 20000,
      "warning_cache_rate": 0,
      "critical_cache_rate": 0,
      "error_weight": 0.60,
      "ttft_weight": 0.20,
      "cache_weight": 0.20
    }';

UPDATE channel_monitor_v2_config
SET health_thresholds = '{
      "minimum_sample": 50,
      "warning_error_rate": 0.05,
      "critical_error_rate": 0.20,
      "target_ttft_ms": 3000,
      "warning_ttft_ms": 8000,
      "critical_ttft_ms": 20000,
      "warning_cache_rate": 0,
      "critical_cache_rate": 0,
      "error_weight": 0.60,
      "ttft_weight": 0.20,
      "cache_weight": 0.20
    }'
WHERE id = 1;
