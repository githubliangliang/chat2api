-- [sqlite-converted] from PostgreSQL migration: 127_drop_channel_monitor_deleted_at.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Migration: 127_drop_channel_monitor_deleted_at
-- 纠正 110 引入的 SoftDeleteMixin：日志/聚合表无恢复需求，软删会让行和索引只增不减，
-- 徒增磁盘和查询开销。改回分批物理删（由 OpsCleanupService 每天凌晨统一调度，
-- deleteOldRowsByID 模板，batch=5000）。
--
-- 110 尚未跑过聚合/清理（首次 maintenance 在次日 02:00），所以此处不担心业务数据。
-- 直接 DROP 列 + 索引；对应的 Go 侧 ent schema 已移除 SoftDeleteMixin、repo 的
-- raw SQL 已移除 deleted_at IS NULL 过滤。

DROP INDEX IF EXISTS idx_channel_monitor_histories_deleted_at;
-- [sqlite] DROP COLUMN deleted_at (best-effort)
-- -- [sqlite] best-effort DROP COLUMN
ALTER TABLE channel_monitor_histories DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_channel_monitor_daily_rollups_deleted_at;
-- [sqlite] DROP COLUMN deleted_at (best-effort)
-- -- [sqlite] best-effort DROP COLUMN
ALTER TABLE channel_monitor_daily_rollups DROP COLUMN deleted_at;
