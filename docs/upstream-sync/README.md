# 上游合并指南

本 fork 基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)，但 **git 历史已重写**（最早一条提交就是 SQLite 改造，和上游没有共同祖先）。因此 **不要** `git merge upstream/main`，按功能 cherry-pick / 手工移植。

当前对照与待移植清单见 [PORTING-0.1.176.md](./PORTING-0.1.176.md)。

移植上游代码前先读 [第 4 节「硬约束」](#4-硬约束)，尤其是 9–12 条（SQLite 适配的四个静默陷阱）。这几条的由来见 [第 7 节的事故复盘](#7-案例一次由-sqlite-适配引发的调度事故2026-08-16)。

写 / 改 SQL 时对照 [第 5 节「PG → SQLite 转换速查」](#5-pg--sqlite-转换速查)。上一次 PG 残留核查的结论与复查命令：[PG-REMNANTS-AUDIT.md](./PG-REMNANTS-AUDIT.md)（2026-08-16，基线 `a8ccd19`）。

---

## 1. 仓库关系

| Remote | 地址 | 用途 |
|--------|------|------|
| `origin` | 本 fork（例如 `git@github.com:githubliangliang/sub2api.git`） | 日常推送 |
| `upstream` | `git@github.com:Wei-Shaw/sub2api.git` | 只读拉上游 |

HTTPS 拉上游容易 TLS 中断，用 **SSH**。

```bash
git remote add upstream git@github.com:Wei-Shaw/sub2api.git   # 已有则跳过
git fetch upstream --tags
```

看上游新了什么：

```bash
git log --oneline --decorate upstream/main -30
git log --oneline HEAD..upstream/main | head -50
```

本仓库历史是 squash 过的，`HEAD..upstream/main` **不能**当成「本仓库缺的提交列表」。要以「本仓库有没有这个功能」为准，见 [PORTING-0.1.176.md](./PORTING-0.1.176.md) 的基线对照。

---

## 2. 为什么不能整仓 merge

相对上游，本 fork 已经改了很多：

- 只支持 SQLite，`backend/migrations/*.sql` 是 SQLite 方言
- Redis 可关（进程内 miniredis）
- 菜单隐藏、simple mode、用量落库
- 去掉残留 PostgreSQL 路径

直接 `git merge upstream/main` 几乎找不到共同祖先，会把两边当成两套无关历史硬撞，并容易把已删的 PG 代码、PG 迁移、SaaS 页面带回来。

**这个 fork 只适合 cherry-pick / 手工移植。**

---

## 3. 推荐流程

### 3.1 先看发布说明，再看提交

| 值得合 | 通常不要合 |
|--------|------------|
| 网关 / 调度 / failover bugfix | PostgreSQL 迁移、分区、`pg_dump` |
| 新模型 / 新平台适配 | 多实例 Redis、SaaS 计费 |
| 安全修复 | 已删页面、支付、ops 大功能 |
| 前端小修复（不依赖被删页面） | 把 `database.driver=postgres` 加回来的改动 |
| 与「用量落库」同向的计费修复 | 备份分卷、leader 锁（单机 SQLite 用不上） |

上游发布页：<https://github.com/Wei-Shaw/sub2api/releases>

### 3.2 开同步分支

```bash
git switch -c sync/upstream-$(date +%Y%m%d) main
```

### 3.3 一颗一颗挑，或按功能手工移植

独立、文件少的 bugfix：

```bash
git show <upstream-sha> --stat
git cherry-pick <upstream-sha>
```

大功能（新模型、新字段、新端点）优先 **按文件移植**，不要 cherry-pick 整个 merge commit：

- 上游 PR 常带生成的 `backend/ent/*`，这边只改 `ent/schema` 再 `go generate`
- 上游迁移是 PostgreSQL，这边要 **新建** SQLite 迁移
- `group_repo.go` 等在本 fork 是手写 SQLite SQL，不能整文件覆盖

冲突了先看文件属于哪一类：

| 类型 | 处理 |
|------|------|
| 业务逻辑（`internal/service`、`internal/platform`、gateway handler） | 值得解 |
| `backend/migrations/*.sql` | 转 SQLite 后 **新建** 文件；不要改已应用文件 |
| Ent schema | 改 schema → `go generate ./ent` |
| 本 fork 特有文件 | 优先保留这边：`REFACTOR.md`、`deploy/config.personal.sqlite.yaml`、`EnsureSQLiteAuxTables`、simple mode / hidden menu |

```bash
git add -A
git cherry-pick --continue
# 或
git cherry-pick --skip
git cherry-pick --abort
```

### 3.4 自测再合回 main

```bash
cd backend && go test -tags=unit ./...
cd ../frontend && pnpm run typecheck && pnpm exec vitest run
```

重点盯：

- 登录 / `user_allowed_groups`（缺表会 503）
- 用量写入（`usage_billing_dedup`）
- 调度冷却、账号 failover
- 新迁移能否在现有 SQLite 上跑

---

## 4. 硬约束

1. **不要把 PostgreSQL 路径合回来。** 运行时只开 SQLite。
2. **迁移只增不改。** 已应用的 `*.sql` 改内容会 checksum 失败。个人环境可以删 `*.db` 重建，已有库不行。
3. **上游新迁移先转 SQLite。** `backend/scripts/pg_sql_to_sqlite.py`，逐条对照 [第 5 节的转换速查](#5-pg--sqlite-转换速查)。PG 专属特性（分区、`trgm`、`plpgsql`、`COMMENT ON`、`CREATE EXTENSION`）改写成 SQLite，或继续 `SELECT 1` no-op。**动手前先看 5.1**——`$N` 占位符、`RETURNING`、`IS DISTINCT FROM`、窗口函数 `PARTITION BY` 等一大批构造 SQLite 本来就支持，改了纯属白费功夫还容易引 bug。
4. **迁移编号以本仓库为准。** 上游文件名可能和这边已占用的序号冲突（例如上游 `221_group_model_pricing.sql`，这边 `221` 已是 affiliate）。
5. **Go 版本。** CI 断言 `1.26.5`。上游升 Go，要同时改 `backend/go.mod` 和 workflow。
6. **前端用 pnpm。** 合了 `package.json` 必须提交 `frontend/pnpm-lock.yaml`。
7. **改 interface / Ent schema 要改测试 stub，并 generate。**

```bash
cd backend
go generate ./ent            # schema 变更后
go generate ./cmd/server     # 动了 wire.go
```

不要手改 `wire_gen.go`，不要抄上游生成的 `backend/ent/group*.go`。

8. **bcrypt / `$`。** 写 SQL 或脚本用文件，不要在 shell 里直接贴哈希。

9. **「适配 SQLite」不等于「把服务关掉」。** `skipSQLiteBackgroundJobs`（`internal/service/wire.go`）会跳过整个后台服务的 `Start()`。判断依据必须是**这个服务的 SQL 是不是 PG 专属**，而不是「我们在跑 SQLite」。关错的代价是静默的：服务不启动 → 不报错 → 功能悄悄失效，直到某天现象离根因十万八千里（见第 7 节）。

   遇到上游服务在 SQLite 上报错，正确顺序是：**先看错在哪一条 SQL** → 能改方言就改方言 → 确实是 PG 专属特性（`COPY`、分区、`plpgsql`、advisory lock）才考虑跳过。跳过时在代码里写清楚**跳过的是哪条 SQL**，不要只写「SQLite 不支持」。

   目前仍被跳过的：`UsageCleanupService`、`AccountExpiryService`、`ScheduledTestRunnerService`。其中账号过期自动暂停被关意味着 `auto_pause_on_expired` 不生效——这几个都还没逐条核实过，动它们之前先验证各自的 SQL。

10. **`EnsureSQLiteAuxTables` 的 DDL 必须和迁移文件逐列一致。** 两边都是 `CREATE TABLE IF NOT EXISTS`，**谁先跑谁定型**，而 aux 表通常先跑。一旦列类型不一致，现网库以 aux 表为准，迁移文件里的定义完全是死的，看代码会被误导。

    移植上游新表时：加迁移的同时检查 `sqlite_aux_tables.go` 有没有同名表；两边都要建就让 DDL 完全一致。**改 aux 表的列类型只对新装库生效**，已有库不会重建——需要为存量库单独写迁移，或让读取端兼容两种形态。

11. **读 SQLite 的时间列要类型无关。** `modernc.org/sqlite` 对 `TEXT` 列返回 `string`，对 `DATETIME` 列才返回 `time.Time`。直接 `rows.Scan(&t)`（`*time.Time`）扫一个历史上被建成 `TEXT` 的列，会每次都报 `unsupported Scan, storing driver.Value type string into type *time.Time`。存量库的列类型不由你的代码决定，所以**读取端要兼容两种**（参考 `scanSchedulerOutboxTime`）。这条是 CLAUDE.md「时间戳用 DATETIME」的延伸：写入端约定管不住已经建错的存量库。

12. **去重（dedup）语义不能默认消费者活着。** `INSERT ... ON CONFLICT (dedup_key) DO NOTHING` 的隐含前提是「冲突的那行马上会被消费掉」。消费者一旦停摆，滞留行的 `dedup_key` 就永久占位，**后续同 key 的新事件在入队处被静默吞掉**——没有报错、没有日志，只有功能不生效。

    本仓库改成了**先删后插**：待处理的重复仍然合并成一条，已消费/滞留的旧行不再挡路，正确性不再依赖水位状态。移植上游任何 outbox / 事件表时按同样标准审一遍。

---

## 5. PG → SQLite 转换速查

移植上游 SQL（迁移、手写 raw SQL）时对照本节。**下表全部在本仓库实际运行时验证过**：`modernc.org/sqlite` 内置 SQLite **3.51.2**，比多数「SQLite 不支持 X」的网上说法新很多。

自动转换器：`backend/scripts/pg_sql_to_sqlite.py`。它只处理常见模式，复杂逻辑仍要人工过一遍。

### 5.1 不用改 —— 常被误判成「SQLite 不支持」

改这些等于白费功夫，还容易在改写中引入 bug：

| 构造 | 说明 |
|------|------|
| `$1` / `$2` 占位符 | **SQLite 原生参数语法**，不是 PG 残留。全仓大量使用，不要改成 `?` |
| `RETURNING` | 3.35+ |
| `ON CONFLICT (a,b) DO UPDATE SET x = excluded.x` | 标准 UPSERT |
| `ON CONFLICT (a,b) WHERE cond` | 部分索引推断，支持。**前提是那个部分唯一索引真的存在**，否则运行时报错 |
| `OVER (PARTITION BY ...)` 窗口函数 | 支持。别和 PG 的**表分区** `PARTITION BY RANGE` 混为一谈——后者才要改写 |
| `IS DISTINCT FROM` | 3.39+。硬约束第 3 条里曾把它列为需改写，已过时 |
| `string_agg(x, ',')` | 3.44+。`group_concat` 也可用 |
| JSON `->` / `->>` | 3.38+ |
| `json_extract` / `json_set` / `json_patch` | 支持 |
| `TRUE` / `FALSE` 字面量 | 3.23+ |
| 递归 CTE `WITH RECURSIVE` | 支持 |
| 单对象 `CREATE TABLE/INDEX IF NOT EXISTS` | 支持（`ALTER TABLE ADD COLUMN IF NOT EXISTS` 不支持，见下） |
| `NOW()` / `GREATEST` / `LEAST` / `TO_CHAR` / `HOST` | **本仓库自己补的兼容函数**，见 `sqlite_pg_compat.go`。原样保留即可，但注意 `NOW()` 返回的是 RFC3339Nano **字符串**（带 `Z`），和 `CURRENT_TIMESTAMP` 的格式不同；`TO_CHAR` 只实现了 4 种格式串 |

### 5.2 必须改写

| PG | SQLite | 备注 |
|----|--------|------|
| `ILIKE` | `LIKE` | LIKE 对 **ASCII 默认不区分大小写**，见 5.3 |
| `x::type` | `CAST(x AS type)` | |
| `DATE_TRUNC('day', ts)` | `date(ts)` / `strftime('%Y-%m-%d', ts)` | |
| `EXTRACT(YEAR FROM ts)` | `strftime('%Y', ts)` | 返回字符串，需要数字要 `CAST` |
| `ts + INTERVAL '7 days'` | `datetime(ts, '+7 days')` | |
| `AT TIME ZONE` / `TIMESTAMPTZ` | `DATETIME` + 应用层处理 | |
| `DISTINCT ON (a)` | `ROW_NUMBER() OVER (PARTITION BY a ORDER BY ...)` 外层过滤 `rn=1` | |
| `FOR UPDATE` / `SKIP LOCKED` | 删掉 | SQLite 单写者，靠事务本身串行 |
| `pg_advisory_lock()` | 应用层 mutex | 参考 `TryAcquireCleanupLock` |
| `= ANY(ARRAY[...])` | 展开成 `IN (?,?,...)` | 用 `sqlInt64In` 生成占位符 |
| `array_agg` / `unnest` / `ARRAY[...]` | `group_concat` / JSON 数组 | |
| `COPY t FROM STDIN` | 事务内批量 `INSERT` | |
| `information_schema.tables` | `sqlite_master` | |
| `SERIAL` / `BIGSERIAL` | `INTEGER PRIMARY KEY AUTOINCREMENT` | |
| `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` | 直接 `ADD COLUMN`，**吞掉 duplicate column 错误** | SQLite 无此语法 |
| `CREATE INDEX CONCURRENTLY` | 去掉 `CONCURRENTLY` | |
| `COMMENT ON ...` | 删掉 | |
| `DO $$ ... $$` / plpgsql / 触发器函数 | 改写成普通 SQL，或 `SELECT 1` no-op | |
| `CREATE EXTENSION` / `trgm` / `tsvector` | `SELECT 1` no-op，或换 `LIKE` / FTS5 | |
| `gen_random_uuid()` | 应用层生成 | |
| `PARTITION BY RANGE` 表分区 | 单表 + 索引 | |

### 5.3 最危险的一类：语法通过，语义不同

这类**转换器和 `TestProductionSQLUsesSQLiteDialect` 都抓不到**，只有在真实数据上跑才会暴露：

- **`jsonb_set` 之类不会报「函数不存在」。** SQLite 3.45+ 已有 `jsonb_*` 函数，但路径语法是 `$.k` 而不是 PG 的 `{k}`。报错会是 `bad JSON path`，甚至在某些写法下悄悄返回别的结果。**看到 `jsonb_` 一律改成 `json_` 并改写路径**。
- **`LIKE` 的大小写规则是分裂的**：ASCII 不区分大小写（`'xy'` 匹配 `'Xy'`），非 ASCII **区分**（`'мир'` 匹配不到 `'Мир'`，`'äbc'` 匹配不到 `'ÄBC'`）。从 PG 的 `ILIKE` 改过来时，含非 ASCII 的搜索行为会变。
- **列类型是「建表时谁先跑谁说了算」**，不是代码说了算。声明 `DATETIME` 但库里实际是 `TEXT` 时，写入不报错、读取才炸。见硬约束 10、11。
- **SQLite 是动态类型**：往 `INTEGER` 列写字符串不会报错。PG 上靠列类型兜底的假设在这里不成立。
- **布尔存成 0/1**，`SELECT ... WHERE flag` 行为和 PG 的 `boolean` 有差异。
- **`datetime('now')` 是 UTC**，而 Go 写入的 `time.Time` 可能带本地时区。同一张表混进两种格式，排查时会得出完全相反的时间线结论（第 7 节踩过）。

### 5.4 一个有用的诊断信号

`AUTOINCREMENT` 主键在 `ON CONFLICT ... DO NOTHING` 命中冲突时**仍然消耗 id**：

```
INSERT k='a'                    -> id=1
INSERT k='a' ON CONFLICT DO NOTHING  -> 无行，但 id=2 被吃掉
INSERT k='b'                    -> id=3
```

所以**表里 id 跳号 = 有多少次插入被 `DO NOTHING` 静默吞掉**。第 7 节那次事故里，`scheduler_outbox` 的 id 空洞正是被吞事件的计数。查「事件为什么没落库」时，先看 id 连不连续。

### 5.5 验证手段

```bash
# 静态：扫 internal/ + cmd/ 所有字符串字面量里的 PG 专属语法（24 类模式）
go test ./internal/repository/ -run TestProductionSQLUsesSQLiteDialect -count=1
```

这个测试能拦 5.2 的大部分，但**拦不住 5.3**。改完 SQL 一定要在真实（或复制的）库上跑一遍，别只靠单测里的临时建表——临时表的列类型是你自己写的，正好绕开了最容易出问题的那一类。

---

## 6. 日常节奏

```text
偶尔     git fetch upstream
         看 releases / git log upstream/main

有用的修  cherry-pick 到 sync 分支 → 测 → 合 main
大功能    先看 PR 文件列表，按 PORTING 文档移植
整仓 merge  不要做
```

每合完一轮，更新 [PORTING-0.1.176.md](./PORTING-0.1.176.md) 的状态列，或新开 `PORTING-<tag>.md`。

---

## 7. 案例：一次由 SQLite 适配引发的调度事故（2026-08-16）

修复提交 `c35a482`。放在这里是因为**根因全部出在 fork 的 SQLite 适配上**，移植上游时最容易再次种下同类问题。

### 现象

后台把一个健康账号重新启用后，**开启永远不生效**，请求一直报：

```
no available OpenAI accounts supporting model: gpt-5.6-sol (pool=2, eligibility_empty)
```

数据库里该账号 `schedulable=1`、模型映射齐全、分组正确、配额/限流/过期全干净——**看库完全正常**。重启进程后恢复，过一阵又复发。

### 五层叠加的根因

| # | 根因 | 为什么难查 |
|---|------|-----------|
| 1 | `skipSQLiteBackgroundJobs` 把 `SchedulerSnapshotService.Start()` 整个跳过，outbox poller 和 300 秒全量重建在 SQLite 上从不运行 | 服务没启动就不会报错，**日志里一条调度相关的行都没有** |
| 2 | 即使启动，poller 首轮就死：`scheduler_outbox.created_at` 是 `TEXT` 列（aux 表先建，迁移 036 的 `DATETIME` 被 `IF NOT EXISTS` 空转），`Scan(*time.Time)` 每秒失败 | 被第 1 层完全掩盖，只有强行打开 Start 后才暴露 |
| 3 | poller 从未消费，4650 行滞留、`dedup_key` 永不释放，`ON CONFLICT DO NOTHING` 把每次开关事件**在入队处吞掉** | 审计日志有 4 次开关操作，outbox 表里零条事件——两边对不上才发现 |
| 4 | 开机水位跳过历史行，这些毒行永远不会被清理 | 重启只是让桶重建一次，毒池仍在，所以「重启能好一阵」 |
| 5 | `SetSchedulable` 只在**关闭**时直写账号快照，开启靠（已被吞掉的）outbox | 关闭立刻生效、开启不生效，现象极不对称 |

关键教训：**1 掩盖 2，2 制造 3，3 制造 4**。只修任何一层都不够，而且修了第 1 层才能看见第 2 层。查这类问题不要满足于「找到一个能解释现象的原因」。

### 排查手法（下次直接照做）

线上是单机 SQLite，可以把**真实数据搬到本地**复现，比在生产加日志安全得多：

```bash
# 1. 远端做一致性快照（不要直接拷 .db，WAL 会撕裂）
ssh <vps> 'python3 -c "
import sqlite3
src = sqlite3.connect(\"file:/path/data/sub2api.db?mode=ro\", uri=True)
dst = sqlite3.connect(\"/tmp/repro.db\"); src.backup(dst); dst.close()"'
scp <vps>:/tmp/repro.db /tmp/lab/data/sub2api.db && ssh <vps> 'rm -f /tmp/repro.db'

# 2. 本地起同一份代码，指向副本，log.level: debug、端口改掉、redis.enabled: false
cd backend && go build -o /tmp/lab/server ./cmd/server
DATA_DIR=/tmp/lab/data /tmp/lab/server

# 3. 改本地库的 users.password_hash 换个已知密码 → 正常登录拿 admin JWT
#    （不要手工铸 token，会被 session 绑定校验拦下）
# 4. 用 admin API 按审计日志的顺序重放操作，再打网关请求
```

配套技巧：

- **`audit_logs` 表能还原精确操作序列**（谁、何时、`request_body` 里的原始参数）。`accounts.updated_at` 只告诉你「变过」，不告诉你「变成什么」，别拿它推断。
- **`api_keys.key` 是明文存的**，可以直接取来在本地打真实网关请求。
- 判定资格的关卡散在多处，与其读代码猜，不如**临时在候选循环里逐门打印拒绝原因**（本次就是这么定位的，验证完删掉，别提交）。
- 注意 `scheduler_outbox.created_at` 是 UTC 且无时区后缀，而 `accounts.updated_at` 带 `+08:00`——**比时间线前先统一时区**，否则会得出完全相反的结论。
- `pkill -f <路径片段>` 会连自己的 shell 一起杀掉，用 `for p in $(pgrep -x server); do kill $p; done`。

### 回归测试落点

`internal/repository/scheduler_outbox_enqueue_replace_test.go`、`internal/service/scheduler_snapshot_initial_purge_test.go`。动 outbox / 快照传播链路时先跑这两个。

