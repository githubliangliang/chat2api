# 上游合并指南

本 fork 基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)，但 **git 历史已重写**（最早一条提交就是 SQLite 改造，和上游没有共同祖先）。因此 **不要** `git merge upstream/main`，按功能 cherry-pick / 手工移植。

当前对照与待移植清单见 [PORTING-0.1.176.md](./PORTING-0.1.176.md)。

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
3. **上游新迁移先转 SQLite。** `backend/scripts/pg_sql_to_sqlite.py`。PG 专属（分区、`trgm`、`plpgsql`、`COMMENT ON`、`JSONB`、`IF NOT EXISTS` 多列、`IS DISTINCT FROM`）改写成 SQLite，或继续 `SELECT 1` no-op。
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

---

## 5. 日常节奏

```text
偶尔     git fetch upstream
         看 releases / git log upstream/main

有用的修  cherry-pick 到 sync 分支 → 测 → 合 main
大功能    先看 PR 文件列表，按 PORTING 文档移植
整仓 merge  不要做
```

每合完一轮，更新 [PORTING-0.1.176.md](./PORTING-0.1.176.md) 的状态列，或新开 `PORTING-<tag>.md`。
