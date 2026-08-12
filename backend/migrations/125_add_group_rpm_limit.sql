-- [sqlite-converted] from PostgreSQL migration: 125_add_group_rpm_limit.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add per-group Requests-Per-Minute limit.
-- rpm_limit: 分组统一 RPM 上限（0 = 不限制）。
-- 一旦配置即接管该用户在该分组的限流，覆盖用户级 users.rpm_limit。
-- 计数键：rpm:ug:{user_id}:{group_id}:{minute}。
ALTER TABLE groups ADD COLUMN rpm_limit integer NOT NULL DEFAULT 0;

-- [sqlite] skipped COMMENT ON column

