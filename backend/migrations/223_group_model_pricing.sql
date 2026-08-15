-- Per-model group pricing overrides channel and built-in rates.
-- long_context_pricing_enabled defaults ON so existing groups keep official ≥200k ladders.
ALTER TABLE groups ADD COLUMN long_context_pricing_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE groups ADD COLUMN model_pricing TEXT;
