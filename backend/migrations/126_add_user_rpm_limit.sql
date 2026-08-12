-- [sqlite-converted] from PostgreSQL migration: 126_add_user_rpm_limit.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add per-user Requests-Per-Minute cap.
-- rpm_limit: 用户全局 RPM 兜底（0 = 不限制）。
-- 仅当所访问分组未设置 rpm_limit 且无 user-group rpm_override 时作为兜底生效。
-- 计数键：rpm:u:{user_id}:{minute}。
ALTER TABLE users ADD COLUMN rpm_limit integer NOT NULL DEFAULT 0;

-- [sqlite] skipped COMMENT ON column

