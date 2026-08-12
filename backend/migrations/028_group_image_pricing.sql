-- [sqlite-converted] from PostgreSQL migration: 028_group_image_pricing.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 为 Antigravity 分组添加图片生成计费配置
-- 支持 gemini-3-pro-image 模型的 1K/2K/4K 分辨率按次计费

ALTER TABLE groups ADD COLUMN image_price_1k DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN image_price_2k DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN image_price_4k DECIMAL(20,8);

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

