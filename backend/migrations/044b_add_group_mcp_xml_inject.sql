-- [sqlite-converted] from PostgreSQL migration: 044b_add_group_mcp_xml_inject.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add mcp_xml_inject field to groups table (for antigravity platform)
ALTER TABLE groups ADD COLUMN mcp_xml_inject BOOLEAN NOT NULL DEFAULT true;
