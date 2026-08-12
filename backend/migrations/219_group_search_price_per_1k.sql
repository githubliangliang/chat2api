-- [sqlite-converted] from PostgreSQL migration: 219_group_search_price_per_1k.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Grok / 通用搜索工具显式定价（per 1000 calls，USD）。
-- NULL = 使用代码默认 $10/1k；显式 0 = 免费；>0 = 分组覆盖价。
ALTER TABLE groups ADD COLUMN search_price_per_1k DECIMAL(20,8);
