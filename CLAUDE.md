# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

Fork of [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) optimized for **personal / single-node** use:

| Fork capability | Detail |
|-----------------|--------|
| SQLite | Only supported database; `backend/migrations/*.sql` is **SQLite dialect** |
| Redis optional | `redis.enabled: false` → in-process miniredis (single node only) |
| Hide menus | Setting `hidden_menu_keys` + admin page `/admin/menu` |
| Usage billing | `usage_billing_dedup` + indexes so successful requests still write usage |
| Simple mode | `run_mode: simple` weakens SaaS/billing UI and hides several sidebar items |

This fork is SQLite-only and prioritizes 1C1G native deploy. Upstream multi-instance production still uses PostgreSQL + external Redis. Deeper notes: [REFACTOR.md](./REFACTOR.md), [DEV_GUIDE.md](./DEV_GUIDE.md), [REMOVED_PAGES.md](./REMOVED_PAGES.md), [deploy/START_NATIVE.md](./deploy/START_NATIVE.md).

**Go version:** `1.26.5` (from `backend/go.mod`). CI asserts this string; bump go.mod and workflow version checks together.

**Frontend package manager:** **pnpm only** (not npm). Commit `frontend/pnpm-lock.yaml` after dependency changes. pnpm v11 needs `frontend/pnpm-workspace.yaml` `allowBuilds` for `esbuild` / `vue-demi` postinstall.

## Common commands

### Frontend (`frontend/`)

```bash
cd frontend
pnpm install
pnpm run dev              # Vite dev server
pnpm run build            # vue-tsc -b && vite build → backend/internal/web/dist
pnpm run lint:check
pnpm run typecheck
pnpm run test:run         # all vitest
pnpm exec vitest run path/to/file.spec.ts   # single test file
```

Root Makefile critical frontend suite:

```bash
make test-frontend-critical
```

### Backend (`backend/`)

```bash
cd backend
go build -o bin/server ./cmd/server
go test ./...
go test -tags=unit ./...
go test -tags=integration ./...
go test -tags=unit ./internal/repository/ -run 'TestName' -count=1   # single test
golangci-lint run ./...   # CI uses v2.9
```

Codegen after schema / wire changes:

```bash
cd backend
go generate ./ent          # after backend/ent/schema/*.go changes — commit generated ent/
go generate ./cmd/server   # Wire DI → wire_gen.go
# or: make -C backend generate
```

### 1C1G native single binary (do not build on 1G VPS)

```bash
cd frontend && pnpm install && pnpm run build && cd ..
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags embed -ldflags="-s -w" \
  -o ../sub2api ./cmd/server
```

- `-tags embed` embeds `backend/internal/web/dist` (see `internal/web/embed_on.go`).
- Root `/sub2api` is gitignored; deploy artifact only.
- Personal config template: `deploy/config.personal.sqlite.yaml`
- systemd unit: `deploy/sub2api-sqlite.service`

Docker (SQLite compose): see README / `deploy/docker-compose.sqlite.yml`.

### Root

```bash
make build                # backend + frontend
make test                 # backend tests + frontend lint/typecheck/critical vitest
make -C backend test-unit
```

## Architecture

```
cmd/server          Entry, flags (-setup, -version), setup wizard vs main server
internal/config     Viper config; run_mode, database.driver, redis.enabled
internal/setup      First-run install (CLI / web / AUTO_SETUP); creates admin user once
internal/server     Gin router, middleware, route registration
internal/handler    HTTP handlers (admin, auth, gateway, payment, …)
internal/service    Business logic
internal/repository Ent client + raw SQL repos; migrations runner; SQLite aux tables
internal/web        Embedded SPA (build tag embed)
ent/schema          Ent schemas → generated ent/*
migrations/         Ordered SQL migrations (currently SQLite dialect)
```

**DI:** Google Wire (`cmd/server/wire.go` → `wire_gen.go`). Prefer regenerating rather than hand-editing `wire_gen.go`.

**Request path (API gateway):** client → Gin gateway handlers → account/group scheduling + upstream providers (`internal/service`, `internal/platform`) → usage/billing write path (dedup tables matter on SQLite).

**Frontend:** Vue 3 + Pinia + Vue Router + Tailwind. Admin views under `frontend/src/views/admin/`, user views under `views/user/`. Sidebar filters by `run_mode` (`hideInSimpleMode`) and `hidden_menu_keys`. API clients under `frontend/src/api/`.

**Auth / admin bootstrap:**

- `default.admin_email` / `default.admin_password` in YAML are **not** live account updates. Login uses `users` table (bcrypt). Setup / `AUTO_SETUP` creates admin once; later password changes need DB or admin UI (UI may be hidden in simple mode).
- Direct URL `/admin/users` still works in simple mode even when sidebar hides it; router blocks other simple-restricted paths (`/admin/groups`, subscriptions, redeem, …).

## SQLite-specific constraints (this fork)

1. **Migrations are SQLite SQL.** PG features (partitioning, trgm, plpgsql, some backfills) are no-ops (`SELECT 1`). Do not assume PG-only SQL runs. Converter: `backend/scripts/pg_sql_to_sqlite.py`.
2. **Timestamps:** use `DATETIME` (not `TEXT`) for `*_at` columns so `modernc.org/sqlite` can scan into `time.Time`. DSN must include `_time_format=sqlite` (`DatabaseConfig.sqliteDSN`, setup DSN builder).
3. **Statement splitting:** `splitSQLStatements` in `migrations_runner.go` must ignore `;` inside comments/strings. Naive split breaks large files (e.g. `033_ops_monitoring_vnext.sql`).
4. **Checksum immutability:** changing an already-applied migration file fails checksum on existing DBs. Prefer new migration files; for personal wipe-and-recreate, deleting `*.db` is acceptable.
5. **Safety-net tables:** `EnsureSQLiteAuxTables` creates tables some no-op migrations omitted (e.g. `user_allowed_groups`). Login loads allowed groups — missing table → 503 on login.
6. **Embedded Redis:** fine for one process; multi-instance needs real Redis.
7. **Some background jobs still emit PG SQL** (`COPY`, `$1` placeholders) on SQLite — often logged as non-fatal; ops should stay `enabled: false` for 1C1G.

## Development pitfalls (repo-specific)

- Changing a Go interface → update all test stubs/mocks (`DEV_GUIDE` 坑 6).
- Ent schema change without `go generate ./ent` + committing `ent/` → dead code paths.
- Frontend: if `pnpm install` blocks on build scripts, set `allowBuilds.esbuild` / `vue-demi` to `true` in `frontend/pnpm-workspace.yaml`.
- bcrypt hashes contain `$` — shell/PowerShell expands them; write SQL/scripts via files.
- Bulk-editing accounts across platforms can wipe model mappings (OpenAI Codex, etc.); group bulk ops by platform (`DEV_GUIDE` 坑 10).

## Config touchpoints

| Concern | Where |
|---------|--------|
| Personal 1C1G defaults | `deploy/config.personal.sqlite.yaml` |
| Full example | `deploy/config.example.yaml` |
| Data dir | `DATA_DIR` env (config + sqlite path + install lock) |
| Install lock | `$DATA_DIR/.installed` + `config.yaml` → skips setup wizard |
| Simple mode | `run_mode: simple` |
| Hide ops / batch image | `ops.enabled`, `batch_image.enabled` |

## Admin CLI skill

Repo skill `skills/sub2api-admin` wraps admin HTTP API (`node scripts/sub2api-admin.js …`) with `SUB2API_BASE_URL` + `SUB2API_ADMIN_API_KEY` or `SUB2API_JWT`. Prefer it over ad-hoc curl for account/group bulk ops.
