-- [sqlite-converted] from PostgreSQL migration: 134_image_generation_group_controls.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 生图能力与图片倍率模式控制
-- 兼容性原则：
-- 1. 不改写现有 image_price_1k/2k/4k，避免改变已配置分组的最终图片价格。
-- 2. 现有 openai/gemini/antigravity 分组默认保持可生图，避免升级后中断已有图片业务。
-- 3. 现有分组默认共享当前有效分组倍率，保持历史扣费公式。

ALTER TABLE groups ADD COLUMN allow_image_generation BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE groups ADD COLUMN image_rate_independent BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE groups ADD COLUMN image_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

UPDATE groups
SET allow_image_generation = true
WHERE platform IN ('openai', 'gemini', 'antigravity');

UPDATE groups
SET image_rate_independent = false,
    image_rate_multiplier = 1.0;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

