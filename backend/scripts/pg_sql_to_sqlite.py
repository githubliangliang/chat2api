#!/usr/bin/env python3
"""Convert PostgreSQL migration SQL to SQLite dialect.

Usage:
  python3 pg_sql_to_sqlite.py [--dry-run] [--path DIR]

In-place conversion of *.sql under backend/migrations by default.
Complex plpgsql / PG-only blocks are replaced with safe no-ops or simplified SQL.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def strip_dollar_blocks(sql: str, keep_body: bool = False) -> str:
    """Remove DO $$ ... $$ and FUNCTION ... AS $$ ... $$ bodies.

    When keep_body is False (default), replace entire DO blocks with a comment.
    """
    # DO [LANGUAGE ...] $$ ... $$;  or DO $tag$ ... $tag$;
    do_pat = re.compile(
        r"DO\s*(?:LANGUAGE\s+\w+\s*)?\$([A-Za-z_]*)\$.*?\$\1\$\s*;",
        re.IGNORECASE | re.DOTALL,
    )
    sql = do_pat.sub(
        "-- [sqlite] skipped PostgreSQL DO $$ ... $$ block\n",
        sql,
    )

    # CREATE [OR REPLACE] FUNCTION ... AS $$ ... $$ [LANGUAGE ...];
    # Also handles LANGUAGE plpgsql AS $$ ... $$;
    func_pat = re.compile(
        r"CREATE\s+OR\s+REPLACE\s+FUNCTION\b.*?\$([A-Za-z_]*)\$.*?\$\1\$\s*(?:LANGUAGE\s+\w+\s*)?;",
        re.IGNORECASE | re.DOTALL,
    )
    sql = func_pat.sub(
        "-- [sqlite] skipped PostgreSQL FUNCTION definition\n",
        sql,
    )
    func_pat2 = re.compile(
        r"CREATE\s+FUNCTION\b.*?\$([A-Za-z_]*)\$.*?\$\1\$\s*(?:LANGUAGE\s+\w+\s*)?;",
        re.IGNORECASE | re.DOTALL,
    )
    sql = func_pat2.sub(
        "-- [sqlite] skipped PostgreSQL FUNCTION definition\n",
        sql,
    )

    # DROP FUNCTION IF EXISTS ...;
    sql = re.sub(
        r"DROP\s+FUNCTION\s+IF\s+EXISTS\s+[^;]+;",
        "-- [sqlite] skipped DROP FUNCTION\n",
        sql,
        flags=re.IGNORECASE,
    )
    return sql


def convert_types_and_keywords(sql: str) -> str:
    # Comments header
    sql = re.sub(
        r"--\s*PostgreSQL\s*15\+?",
        "-- SQLite (converted from PostgreSQL)",
        sql,
        flags=re.IGNORECASE,
    )

    # CREATE EXTENSION
    sql = re.sub(
        r"CREATE\s+EXTENSION\s+IF\s+NOT\s+EXISTS\s+[^;]+;",
        "-- [sqlite] skipped CREATE EXTENSION\n",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"CREATE\s+EXTENSION\s+[^;]+;",
        "-- [sqlite] skipped CREATE EXTENSION\n",
        sql,
        flags=re.IGNORECASE,
    )

    # SET LOCAL / SET statement_timeout etc.
    sql = re.sub(
        r"SET\s+LOCAL\s+[^;]+;",
        "-- [sqlite] skipped SET LOCAL\n",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"SET\s+statement_timeout\s*=\s*[^;]+;",
        "-- [sqlite] skipped SET statement_timeout\n",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"SET\s+lock_timeout\s*=\s*[^;]+;",
        "-- [sqlite] skipped SET lock_timeout\n",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"SET\s+search_path\s*(TO|=)\s*[^;]+;",
        "-- [sqlite] skipped SET search_path\n",
        sql,
        flags=re.IGNORECASE,
    )

    # CONCURRENTLY
    sql = re.sub(r"\bCONCURRENTLY\b", "", sql, flags=re.IGNORECASE)

    # DROP ... CASCADE / VIEW CASCADE
    sql = re.sub(
        r"\bDROP\s+TABLE\s+IF\s+EXISTS\s+([^;]+?)\s+CASCADE\s*;",
        r"DROP TABLE IF EXISTS \1;",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bDROP\s+VIEW\s+IF\s+EXISTS\s+([^;]+?)\s+CASCADE\s*;",
        r"DROP VIEW IF EXISTS \1;",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bDROP\s+INDEX\s+IF\s+EXISTS\s+([^;]+?)\s+CASCADE\s*;",
        r"DROP INDEX IF EXISTS \1;",
        sql,
        flags=re.IGNORECASE,
    )

    # DROP CONSTRAINT IF EXISTS
    sql = re.sub(
        r"ALTER\s+TABLE\s+(\w+)\s+DROP\s+CONSTRAINT\s+IF\s+EXISTS\s+(\w+)\s*;",
        r"-- [sqlite] DROP CONSTRAINT \2 on \1 → try DROP INDEX\nDROP INDEX IF EXISTS \2;",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"ALTER\s+TABLE\s+(\w+)\s+DROP\s+CONSTRAINT\s+(\w+)\s*;",
        r"-- [sqlite] DROP CONSTRAINT \2 on \1 → try DROP INDEX\nDROP INDEX IF EXISTS \2;",
        sql,
        flags=re.IGNORECASE,
    )

    # ALTER COLUMN ... TYPE ... (unsupported) → comment out
    sql = re.sub(
        r"ALTER\s+TABLE\s+(\w+)\s+ALTER\s+COLUMN\s+(\w+)\s+TYPE\s+[^;]+;",
        r"-- [sqlite] skipped ALTER COLUMN TYPE on \1.\2 (unsupported)\n",
        sql,
        flags=re.IGNORECASE,
    )
    # ALTER COLUMN ... SET/DROP DEFAULT/NOT NULL — limited support; keep SET DEFAULT / DROP DEFAULT if simple
    # ALTER COLUMN ... SET NOT NULL without default often fails on SQLite if nulls exist — leave as comment for SET DATA TYPE variants already handled.

    # BIGSERIAL / SERIAL primary keys
    sql = re.sub(
        r"\bBIGSERIAL\s+PRIMARY\s+KEY\b",
        "INTEGER PRIMARY KEY AUTOINCREMENT",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bSERIAL\s+PRIMARY\s+KEY\b",
        "INTEGER PRIMARY KEY AUTOINCREMENT",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(r"\bBIGSERIAL\b", "INTEGER", sql, flags=re.IGNORECASE)
    # Avoid double-replacing SERIAL inside AUTOINCREMENT — only bare SERIAL type
    sql = re.sub(r"(?<!AUTOINCR)\bSERIAL\b", "INTEGER", sql, flags=re.IGNORECASE)

    # Timestamps → DATETIME so modernc.org/sqlite can Scan into time.Time
    # (TEXT columns stay strings and break Ent field.Time scanning).
    sql = re.sub(r"\bTIMESTAMPTZ\b", "DATETIME", sql, flags=re.IGNORECASE)
    sql = re.sub(
        r"\bTIMESTAMP\s+WITH\s+TIME\s+ZONE\b",
        "DATETIME",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bTIMESTAMP\s+WITHOUT\s+TIME\s+ZONE\b",
        "DATETIME",
        sql,
        flags=re.IGNORECASE,
    )
    # Bare TIMESTAMP as column type (not in function names)
    sql = re.sub(r"\bTIMESTAMP\b", "DATETIME", sql, flags=re.IGNORECASE)

    # JSON / binary / network / uuid
    sql = re.sub(r"\bJSONB\b", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bBYTEA\b", "BLOB", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bINET\b", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bCIDR\b", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bUUID\b", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bDOUBLE\s+PRECISION\b", "REAL", sql, flags=re.IGNORECASE)

    # Arrays → TEXT (store JSON)
    sql = re.sub(r"\bBIGINT\s*\[\s*\]", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bTEXT\s*\[\s*\]", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bINTEGER\s*\[\s*\]", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bINT\s*\[\s*\]", "TEXT", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bVARCHAR\s*\(\s*\d+\s*\)\s*\[\s*\]", "TEXT", sql, flags=re.IGNORECASE)

    # BOOLEAN stays; SQLite accepts TRUE/FALSE. DECIMAL/NUMERIC kept (affinity).

    # NOW() → datetime('now')  (SQLite requires parentheses around non-constant DEFAULTs)
    sql = re.sub(r"\bNOW\s*\(\s*\)", "datetime('now')", sql, flags=re.IGNORECASE)
    # DEFAULT datetime('now') → DEFAULT (datetime('now'))
    sql = re.sub(
        r"DEFAULT\s+datetime\('now'\)",
        "DEFAULT (datetime('now'))",
        sql,
        flags=re.IGNORECASE,
    )

    # CURRENT_TIMESTAMP is fine in SQLite

    # Casts ::type
    sql = re.sub(r"::\s*jsonb\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*json\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*text\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*int(?:eger)?\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*bigint\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*boolean\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*numeric(?:\s*\(\s*\d+\s*,\s*\d+\s*\))?", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*float8\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*float4\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*real\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*date\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*uuid\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"::\s*regclass\b", "", sql, flags=re.IGNORECASE)

    # jsonb_build_object(...) — leave; will need runtime helpers or rewrite later
    # jsonb_typeof, jsonb operators — rewrite common ones
    # col ->> 'key' → json_extract(col, '$.key')
    def repl_jarrow2(m: re.Match) -> str:
        expr, key = m.group(1), m.group(2)
        return f"json_extract({expr}, '$.{key}')"

    # Only simple identifier/expr ->> 'literal'
    sql = re.sub(
        r"([A-Za-z_][\w\.]*)\s*->>\s*'([^']+)'",
        repl_jarrow2,
        sql,
    )
    sql = re.sub(
        r"([A-Za-z_][\w\.]*)\s*->\s*'([^']+)'",
        lambda m: f"json_extract({m.group(1)}, '$.{m.group(2)}')",
        sql,
    )

    # '{}'::jsonb already handled by cast strip → '{}'

    # public.schema prefix
    sql = re.sub(r"\bpublic\.", "", sql)

    # GIN/GIST indexes → regular indexes (drop operator classes)
    # CREATE INDEX ... ON t USING gin (col gin_trgm_ops) → CREATE INDEX ... ON t (col)
    sql = re.sub(
        r"\bUSING\s+gin\s*\(\s*([A-Za-z_\"\w]+)\s+gin_trgm_ops\s*\)",
        r"(\1)",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bUSING\s+gin\s*\(",
        "(",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bUSING\s+gist\s*\(",
        "(",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bUSING\s+btree\s*\(",
        "(",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"\bUSING\s+brin\s*\(",
        "(",
        sql,
        flags=re.IGNORECASE,
    )
    # Remove leftover gin_trgm_ops / jsonb_ops if any
    sql = re.sub(r"\s+gin_trgm_ops\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\s+jsonb_ops\b", "", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\s+jsonb_path_ops\b", "", sql, flags=re.IGNORECASE)

    # ILIKE → LIKE (case-insensitive via COLLATE NOCASE if needed; simple LIKE for now)
    sql = re.sub(r"\bILIKE\b", "LIKE", sql, flags=re.IGNORECASE)

    # FILTER (WHERE cond) — leave; SQLite 3.30+ supports FILTER on aggregates

    # RETURNING — SQLite supports RETURNING since 3.35

    # ON CONFLICT — supported

    # to_regclass('x') → (SELECT 1 FROM sqlite_master WHERE name='x')
    sql = re.sub(
        r"to_regclass\(\s*'([^']+)'\s*\)",
        lambda m: (
            f"(SELECT name FROM sqlite_master WHERE type IN ('table','view') "
            f"AND name = '{m.group(1).split('.')[-1]}')"
        ),
        sql,
        flags=re.IGNORECASE,
    )

    # gen_random_uuid()
    sql = re.sub(
        r"\bgen_random_uuid\s*\(\s*\)",
        "lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || "
        "substr(hex(randomblob(2)),2) || '-' || "
        "substr('89ab',abs(random()) % 4 + 1, 1) || "
        "substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))",
        sql,
        flags=re.IGNORECASE,
    )

    # BTRIM → TRIM
    sql = re.sub(r"\bBTRIM\s*\(", "TRIM(", sql, flags=re.IGNORECASE)

    # date_trunc('month', x) rough equivalent
    sql = re.sub(
        r"date_trunc\(\s*'month'\s*,\s*([^)]+)\)",
        r"strftime('%Y-%m-01', \1)",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"date_trunc\(\s*'day'\s*,\s*([^)]+)\)",
        r"strftime('%Y-%m-%d', \1)",
        sql,
        flags=re.IGNORECASE,
    )
    sql = re.sub(
        r"date_trunc\(\s*'hour'\s*,\s*([^)]+)\)",
        r"strftime('%Y-%m-%d %H:00:00', \1)",
        sql,
        flags=re.IGNORECASE,
    )

    # INTERVAL arithmetic — best-effort comment for complex cases
    # (month_start - INTERVAL '1 month') — leave; may need manual fix

    # EXECUTE format(...) — only inside DO blocks which we strip

    # ANALYZE;
    sql = re.sub(r"^\s*ANALYZE\s*;", "-- [sqlite] ANALYZE;", sql, flags=re.IGNORECASE | re.MULTILINE)

    # partial_hashes TEXT[] already handled

    # Fix double spaces from CONCURRENTLY removal
    sql = re.sub(r"CREATE\s+UNIQUE\s+INDEX\s{2,}", "CREATE UNIQUE INDEX ", sql, flags=re.IGNORECASE)
    sql = re.sub(r"CREATE\s+INDEX\s{2,}", "CREATE INDEX ", sql, flags=re.IGNORECASE)

    return sql


def convert_jsonb_functions(sql: str) -> str:
    """Best-effort jsonb_* → json_* rewrites."""
    sql = re.sub(r"\bjsonb_typeof\s*\(", "json_type(", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bjsonb_array_length\s*\(", "json_array_length(", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bjsonb_object_keys\s*\(", "json_each(", sql, flags=re.IGNORECASE)
    # jsonb_build_object(k1,v1,k2,v2,...) → json_object(k1,v1,...)
    sql = re.sub(r"\bjsonb_build_object\s*\(", "json_object(", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bjsonb_build_array\s*\(", "json_array(", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bjsonb_agg\s*\(", "json_group_array(", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bjson_agg\s*\(", "json_group_array(", sql, flags=re.IGNORECASE)
    sql = re.sub(r"\bjsonb_set\s*\(", "json_set(", sql, flags=re.IGNORECASE)
    # string_agg(x, sep) → group_concat(x, sep)
    sql = re.sub(r"\bstring_agg\s*\(", "group_concat(", sql, flags=re.IGNORECASE)
    return sql


# Legacy data backfills that only make sense on long-lived PostgreSQL installs.
# On SQLite (fresh personal deploys) they are safe no-ops.
LEGACY_BACKFILL_NOOPS = {
    "115_auth_identity_legacy_external_backfill.sql",
    "116_auth_identity_legacy_external_safety_reports.sql",
    "013_log_orphan_allowed_groups.sql",
    "019_migrate_wechat_to_attributes.sql",
    "098_migrate_purchase_subscription_to_custom_menu.sql",
    "099_fix_migrated_purchase_menu_label_icon.sql",
    "104_migrate_notify_emails_to_struct.sql",
    "105_migrate_websearch_emulation_to_tristate.sql",
    "049_unify_antigravity_model_mapping.sql",
    "050_map_opus46_to_opus45.sql",
    "051_migrate_opus45_to_opus46_thinking.sql",
}


def is_effectively_empty_sql(sql: str) -> bool:
    """True if only comments / SELECT 1 / whitespace remain."""
    body = sql
    body = re.sub(r"--[^\n]*", "", body)
    body = re.sub(r"/\*.*?\*/", "", body, flags=re.DOTALL)
    body = body.strip()
    if not body:
        return True
    # only SELECT 1; style placeholders
    if re.fullmatch(r"(SELECT\s+1\s*;?\s*)+", body, flags=re.IGNORECASE):
        return True
    # only skip markers
    meaningful = re.sub(r";", " ", body)
    meaningful = re.sub(r"\s+", " ", meaningful).strip().lower()
    if meaningful in {"", "select 1"}:
        return True
    # no DDL/DML keywords left
    if not re.search(
        r"\b(CREATE|ALTER|DROP|INSERT|UPDATE|DELETE|REPLACE|WITH|PRAGMA)\b",
        body,
        flags=re.IGNORECASE,
    ):
        return True
    return False


def convert_file(content: str, filename: str) -> str:
    # Special-case files that are almost entirely PG-only
    special_noop = {
        "035_usage_logs_partitioning.sql": (
            "-- [sqlite-converted] from PostgreSQL migration: 035_usage_logs_partitioning.sql\n"
            "-- SQLite: table partitioning is not supported; no-op migration.\n"
            "-- Original PostgreSQL migration created monthly partitions for usage_logs.\n"
            "SELECT 1;\n"
        ),
        "065_add_search_trgm_indexes.sql": (
            "-- [sqlite-converted] from PostgreSQL migration: 065_add_search_trgm_indexes.sql\n"
            "-- SQLite: pg_trgm / GIN trigram indexes are not available; no-op migration.\n"
            "-- Fuzzy search will fall back to LIKE without trigram acceleration.\n"
            "SELECT 1;\n"
        ),
        "072_add_usage_billing_dedup_created_at_brin_notx.sql": (
            "-- [sqlite-converted] from PostgreSQL migration: 072_add_usage_billing_dedup_created_at_brin_notx.sql\n"
            "-- SQLite: BRIN indexes are not supported; use a regular index instead.\n"
            "CREATE INDEX IF NOT EXISTS idx_usage_logs_billing_dedup_created_at\n"
            "    ON usage_logs (created_at);\n"
        ),
    }
    if filename in special_noop:
        return special_noop[filename]

    if filename in LEGACY_BACKFILL_NOOPS:
        return (
            f"-- [sqlite-converted] from PostgreSQL migration: {filename}\n"
            f"-- SQLite: legacy PostgreSQL data backfill skipped (fresh DB / no legacy tables).\n"
            f"SELECT 1;\n"
        )

    content = strip_dollar_blocks(content)
    content = convert_types_and_keywords(content)
    content = convert_jsonb_functions(content)

    # Clean excessive blank lines
    content = re.sub(r"\n{4,}", "\n\n\n", content)

    # Ensure file ends with newline
    if content and not content.endswith("\n"):
        content += "\n"

    # Prepend conversion banner if not already present
    if "[sqlite-converted]" not in content[:500]:
        banner = (
            f"-- [sqlite-converted] from PostgreSQL migration: {filename}\n"
            f"-- Auto-converted for SQLite dialect. Review complex logic if needed.\n"
        )
        content = banner + content

    if is_effectively_empty_sql(content):
        return (
            f"-- [sqlite-converted] from PostgreSQL migration: {filename}\n"
            f"-- SQLite: original migration had only PostgreSQL-specific logic; no-op.\n"
            f"SELECT 1;\n"
        )

    return content


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument(
        "--path",
        type=Path,
        default=Path(__file__).resolve().parents[1] / "migrations",
    )
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--only", type=str, default="", help="Comma-separated filenames")
    args = ap.parse_args()

    only = {x.strip() for x in args.only.split(",") if x.strip()}
    files = sorted(args.path.glob("*.sql"))
    if only:
        files = [f for f in files if f.name in only]

    changed = 0
    unchanged = 0
    for f in files:
        raw = f.read_text(encoding="utf-8")
        new = convert_file(raw, f.name)
        if new != raw:
            changed += 1
            if args.dry_run:
                print(f"WOULD CHANGE {f.name} ({len(raw)} -> {len(new)} bytes)")
            else:
                f.write_text(new, encoding="utf-8")
                print(f"UPDATED {f.name}")
        else:
            unchanged += 1
            print(f"unchanged {f.name}")

    print(f"\nDone: {changed} updated, {unchanged} unchanged, {len(files)} total")
    return 0


if __name__ == "__main__":
    sys.exit(main())
