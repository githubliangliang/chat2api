-- [sqlite-converted] from PostgreSQL migration: 029_usage_log_image_fields.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 为使用日志添加图片生成统计字段
-- 用于记录 gemini-3-pro-image 等图片生成模型的使用情况

ALTER TABLE usage_logs ADD COLUMN image_count INT DEFAULT 0;
ALTER TABLE usage_logs ADD COLUMN image_size VARCHAR(10);
