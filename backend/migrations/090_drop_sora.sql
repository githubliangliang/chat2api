-- [sqlite-converted] from PostgreSQL migration: 090_drop_sora.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Migration: 090_drop_sora
-- Remove all Sora-related database objects.
-- Drops tables: sora_tasks, sora_generations, sora_accounts
-- Drops columns from: groups, users, usage_logs

-- ============================================================
-- 1. Drop Sora tables
-- ============================================================
DROP TABLE IF EXISTS sora_tasks;
DROP TABLE IF EXISTS sora_generations;
DROP TABLE IF EXISTS sora_accounts;

-- ============================================================
-- 2. Drop Sora columns from groups table
-- ============================================================
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE groups DROP COLUMN sora_image_price_360;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE groups DROP COLUMN sora_image_price_540;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE groups DROP COLUMN sora_video_price_per_request;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE groups DROP COLUMN sora_video_price_per_request_hd;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE groups DROP COLUMN sora_storage_quota_bytes;

-- ============================================================
-- 3. Drop Sora columns from users table
-- ============================================================
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE users DROP COLUMN sora_storage_quota_bytes;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE users DROP COLUMN sora_storage_used_bytes;

-- ============================================================
-- 4. Drop Sora column from usage_logs table
-- ============================================================
-- [sqlite] DROP COLUMN media_type (best-effort)
-- -- [sqlite] best-effort DROP COLUMN
ALTER TABLE usage_logs DROP COLUMN media_type;
