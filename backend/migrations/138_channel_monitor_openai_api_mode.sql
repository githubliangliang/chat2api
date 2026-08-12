-- [sqlite-converted] from PostgreSQL migration: 138_channel_monitor_openai_api_mode.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Migration: 137_channel_monitor_openai_api_mode
-- 为渠道监控和请求模板增加 OpenAI 协议模式：
--   chat_completions -> /v1/chat/completions + messages
--   responses        -> /v1/responses + instructions/input
-- 历史数据默认保持 chat_completions，避免改变现有监控行为。

ALTER TABLE channel_monitors ADD COLUMN api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitor_request_templates ADD COLUMN api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

-- [sqlite] skipped PostgreSQL DO $$ ... $$ block


CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider_api_mode
    ON channel_monitors (provider, api_mode);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_templates_provider_api_mode
    ON channel_monitor_request_templates (provider, api_mode);
