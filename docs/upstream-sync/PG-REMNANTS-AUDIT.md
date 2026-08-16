# PostgreSQL 残留审计

审计日期：**2026-08-16**　　代码基线：`a8ccd19`　　线上实例：1.1.1

**结论：有残留，但不影响使用。** 线上 24 小时日志零 SQL 报错（无 syntax error、no such table/column/function、SQLSTATE、unsupported Scan）。

本文件是一次性的核查存档，不是长期规则。长期规则见 [README 第 4 节「硬约束」](./README.md#4-硬约束)，方言对照见 [第 5 节](./README.md#5-pg--sqlite-转换速查)。

---

## 结论速览

| 位置 | 残留 | 影响 | 依据 |
|------|------|------|------|
| `go.mod` | `lib/pq v1.10.9`（`pgx` 为间接依赖） | 无 | 仅 6 个 `_test.go` 引用；生产侧 `error_translate.go` 只在注释里提到它，用 `interface{ SQLState() string }` 反射判 23505，刻意不 import |
| 驱动选择 | — | 无 | `NormalizedDriver()` 硬返回 sqlite；`sql.Open` 全是 `"sqlite"`；`driver: postgres` 真的被忽略 |
| `migrations_runner.go:523` | `information_schema.tables` | 无 | 在 `tableExists()` 的 `if sqlite {}` else 分支里，SQLite 走 `sqlite_master`，是死代码 |
| `user_platform_quota_repo.go:120/198/419` | `ON CONFLICT (user_id, platform) WHERE deleted_at IS NULL` | 无 | 部分索引推断，SQLite 支持；`userplatformquota_user_id_platform_uq` 索引在线上确实存在（建同构表实测通过） |
| `migrations/027` | `PARTITION BY` | 无 | 是窗口函数 `ROW_NUMBER() OVER (PARTITION BY ...)`，不是表分区 |
| `migrations/054` | `to_tsvector` | 无 | 只是一行「已跳过」的注释 |
| `frontend/src/api/admin/dataManagement.ts` | `BackupType='postgres'`、`SourceType='postgres'` | 无 | 死代码：无路由、无组件引用，UI 里点不到 |
| `/api/v1/admin/data-management/*` | 17 个路由仍注册 | 无 | `EnsureAgentEnabled()` **无条件返回** `ErrDataManagementDeprecated`(503)，不连 PG、不碰库 |
| `deploy/docker-compose{,.dev,.local,.build}.yml`、`.env.example` | PG 服务定义 | 无（前提是别用） | 上游遗留，README:245 已标注；推荐的 `docker-compose.sqlite.yml` / `.env.sqlite.example` 干净 |

**生产代码中不存在**：`COPY`、`pg_dump`、`psql`、`PGPASSWORD`、`gen_random_uuid()`、`SERIAL`、`DISTINCT ON`、`ILIKE`、`::` 转换、`FOR UPDATE`、`pg_advisory_lock`。

---

## 例外：一个真实存在的 PG 兼容层

`internal/repository/sqlite_pg_compat.go` 用 `sqlite.RegisterScalarFunction` 给 SQLite **补了 5 个 PG 函数**，让大量原样保留的 PG 风格 raw SQL 能跑：

| 函数 | 实现 | 用在哪 |
|------|------|--------|
| `NOW()` | 返回 `time.Now().UTC().Format(time.RFC3339Nano)`——**字符串，带 `Z` 后缀** | **55 处**仓储 SQL 的 `updated_at` / `deleted_at` 戳记 |
| `GREATEST` / `LEAST` | 多参数数值比较 | 余额增减等 |
| `TO_CHAR(ts, fmt)` | 只实现了 4 种用量分析用的格式，其余尽力映射 | 用量趋势按小时/天/周/月分桶 |
| `HOST(inet)` | 原样返回（SQLite 里 `client_ip` 就是 TEXT） | 审计日志 |

**这不是残留，是有意的适配层**，但有两个必须知道的后果：

1. **`NOW()` 写进去的是 RFC3339Nano 字符串**，和 `CURRENT_TIMESTAMP` / `datetime('now')` 的 `2006-01-02 15:04:05` 格式**不一样**。同一张表可能混进多种时间格式，读取端必须全覆盖（见 `parseSchedulerOutboxTimeString` 及其回归测试）。本次审计就发现该函数漏了带 `Z` 的形态，已补。
2. **`TO_CHAR` 只是尽力而为**，不是完整 PG 语义。移植上游新的 `TO_CHAR` 用法前，先确认格式串在这 4 种之内，否则会静默走到 fallback 分支给出错结果。

移植上游 raw SQL 时：**看到 `NOW()` / `GREATEST` / `TO_CHAR` 不用改**，但要意识到它们走的是这层 shim；用到 shim 之外的 PG 函数就必须改写或扩展这个文件。

---

## 已被证伪的旧说法

CLAUDE.md 原第 7 条（已于 `a8ccd19` 修正）称「部分后台任务仍发 PG SQL（`COPY`、`$1` 占位符），ops 应保持 `enabled: false`」。三处都不成立：

1. **没有 `COPY`。** 按多种写法扫过生产代码，只匹配到 Go 的 `copy multipart part` 这类文本。
2. **`$1` 不是 PG 残留。** SQLite 原生支持 `$N` 参数语法（实测通过）。全仓大量使用是正常的，改成 `?` 属于白费功夫。
3. **`ops.enabled: true` 在线上运行正常**，24 小时零 SQL 报错。

这条描述的存在助长了「SQLite 上后台任务不可靠 → 干脆关掉」的判断，而那正是 `c35a482` 调度事故的第一层根因。

---

## 复查方法

```bash
cd backend

# 1. 静态方言审计（24 类 PG 专属语法，扫 internal/ + cmd/ 全部字符串字面量）
go test ./internal/repository/ -run TestProductionSQLUsesSQLiteDialect -count=1

# 2. 审计规则未覆盖的构造
#    注意 NOW() 要排除 Go 的 time.Now()，否则会匹配上万处
grep -rnE '(^|[^.[:alnum:]_])NOW\(\)|COPY [a-z_]+ FROM|pg_dump|psql |PGPASSWORD|information_schema|pg_catalog|gen_random_uuid|BIGSERIAL' \
  --include="*.go" internal/ cmd/ | grep -v _test
# NOW() 的命中是预期的（见上文兼容层），其余应只剩 migrations_runner.go 那条死代码

# 3. PG 驱动是否泄进生产
grep -rln "lib/pq\|jackc/pgx" --include="*.go" internal/ cmd/ | grep -v _test

# 4. 运行时确实只开 sqlite
grep -rn 'sql.Open\|dialect.Postgres' --include="*.go" internal/repository/ internal/setup/ | grep -v _test
```

线上侧（最直接的证据）：

```bash
ssh <vps> 'journalctl --user -u sub2api --since "24 hours ago" --no-pager \
  | grep -iE "syntax error|no such (table|column|function)|SQLSTATE|unsupported Scan|does not exist"'
```

---

## 什么情况下结论会变

- **合入上游新代码**后需重跑第 1、2 步 —— 上游是 PG-first 的。
- **打开任何当前被 `skipSQLiteBackgroundJobs` 跳过的服务**（`UsageCleanupService`、`AccountExpiryService`、`ScheduledTestRunnerService`）—— 这三个的 SQL **尚未逐条核实**，打开前必须先验证。
- **静态审计抓不到 [README 5.3](./README.md#53-最危险的一类语法通过语义不同) 那一类**（语法通过但语义不同、列类型不匹配）。那类只能靠在真实库副本上跑出来，`c35a482` 就是这么找到的。
