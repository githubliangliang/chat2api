-- [sqlite-converted] from PostgreSQL migration: 143_group_models_list_config.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 分组级自定义 /v1/models 展示列表配置。
-- 仅用于控制 GET /v1/models 的展示结果，不参与账号白名单、模型映射或网关调度。

ALTER TABLE groups ADD COLUMN models_list_config TEXT NOT NULL DEFAULT '{}';
