# sub2api_new 重构说明

按根目录 `new.txt` 要求，在 `sub2api/` 基础上重构到 `sub2api_new/`：

1. **支持 SQLite 与 PostgreSQL，Redis 非必须**
2. **菜单可隐藏**（含独立「菜单管理」页面）

---

## 1. 数据库：PostgreSQL | SQLite

### 配置

```yaml
database:
  driver: "postgres"          # 或 "sqlite"
  path: "./data/sub2api.db"   # 仅 sqlite 使用
  # ... 其余为 postgres 连接参数
```

环境变量（auto-setup）：

| 变量 | 说明 |
|------|------|
| `DATABASE_DRIVER` | `postgres`（默认）或 `sqlite` |
| `DATABASE_PATH` | SQLite 文件路径，默认 `./data/sub2api.db` |

### 行为

| 驱动 | Schema 来源 | 说明 |
|------|-------------|------|
| `postgres` | `backend/migrations/*.sql` | 注意：当前迁移文件已转为 **SQLite 方言** |
| `sqlite` | `backend/migrations/*.sql` | 纯 Go 驱动 `modernc.org/sqlite`；启动时跑同一套 SQL 迁移 |

相关代码：

- `backend/internal/config/config.go` — `DatabaseConfig.Driver/Path`、`NormalizedDriver()` / `IsSQLite()`
- `backend/internal/repository/ent.go` — 双驱动打开与 schema 准备
- `backend/internal/repository/migrations_runner.go` — 迁移执行（SQLite 跳过 advisory lock、按语句拆分执行）
- `backend/internal/repository/db_pool.go` — SQLite 连接池收敛
- `backend/internal/setup/*` — 安装向导 / Web 安装 / auto-setup 支持 sqlite
- `backend/scripts/pg_sql_to_sqlite.py` — PG→SQLite 迁移转换脚本

### 注意

- `backend/migrations/*.sql` 已整体转换为 **SQLite 语法**（`INTEGER PRIMARY KEY AUTOINCREMENT`、`TEXT` 时间戳、`json_extract` 等）。
- PG 专属能力（分区表、pg_trgm、plpgsql 触发器/函数、`GRANT`、部分 legacy backfill）在对应迁移中降级为 `SELECT 1` no-op。
- 若需继续用 PostgreSQL 生产库，应另建方言目录或从 git 历史恢复 PG 版 SQL。
- 生产高并发、多实例仍建议 **PostgreSQL**（需配套 PG 方言迁移）。

---

## 2. Redis：可选

### 配置

```yaml
redis:
  enabled: true   # false = 不连接外部 Redis
  host: "localhost"
  # ...
```

环境变量：`REDIS_ENABLED=true|false`

### 行为

| 模式 | 行为 |
|------|------|
| `enabled=true` | 连接外部 Redis（原行为） |
| `enabled=false` | 进程内启动 **miniredis** 嵌入式 Redis，API 兼容，**仅单机** |

相关代码：

- `backend/internal/config/config.go` — `RedisConfig.Enabled` / `IsEnabled()`
- `backend/internal/repository/redis.go` — 嵌入式 fallback
- `backend/internal/setup/*` — 安装时 Redis 可跳过

### 注意

- 嵌入式模式不支持多实例共享缓存/分布式锁。
- 多节点部署必须启用外部 Redis。

---

## 3. 菜单可隐藏 + 菜单管理页

### 3.1 设置项

| 项 | 说明 |
|----|------|
| Key | `hidden_menu_keys` |
| 值 | JSON 字符串数组，元素为侧边栏 **path** |
| 示例 | `["/usage", "/redeem", "/admin/ops"]` |

自定义菜单仍使用原有 `custom_menu_items`。

### 3.2 管理页面（新增）

| 项 | 值 |
|----|-----|
| 路由 | `/admin/menu` |
| 名称 | `AdminMenuManagement` |
| 视图 | `frontend/src/views/admin/MenuManagementView.vue` |
| 侧栏入口 | 管理端侧栏「菜单管理」（系统设置上方） |
| 设置页入口 | 系统设置中的入口卡片 → 跳转 `/admin/menu` |

页面能力：

1. **内置菜单** Tab  
   - 用户端 / 管理端分组  
   - 单项开关 + 全部显示/隐藏  
   - 写入 `hidden_menu_keys`
2. **自定义菜单** Tab  
   - 增删改、排序、可见角色  
   - 写入 `custom_menu_items`
3. **保存后立即生效**  
   - 同步 `adminSettingsStore` + `cachedPublicSettings`  
   - 侧栏 `finalizeNav` 自动重算

### 3.3 侧栏生效链路（已核对）

```
保存菜单配置
  → PUT /admin/settings { hidden_menu_keys, custom_menu_items }
  → adminSettingsStore.hiddenMenuKeys / customMenuItems
  → appStore.cachedPublicSettings.hidden_menu_keys
  → AppSidebar.finalizeNav()
       1) featureFlags
       2) simple 模式
       3) hidden_menu_keys（public ∪ admin store）
       4) 去掉无子项的 expand-only 分组
  → 用户端 / 管理端 / 管理员「我的账户」侧栏同步隐藏
```

覆盖范围：

| 侧栏区域 | computed | 是否走 `finalizeNav` |
|----------|----------|----------------------|
| 普通用户菜单 | `userNavItems` | ✅ |
| 管理员「我的账户」 | `personalNavItems` | ✅ |
| 管理员系统菜单 | `adminNavItems` | ✅（含 `/admin/menu`、`/admin/settings`） |

入口检查：

| 检查项 | 状态 |
|--------|------|
| 侧栏有「菜单管理」 | ✅ `path: /admin/menu` + `MenuIcon` |
| 路由已注册 | ✅ `router/index.ts` |
| 内置目录含 `/admin/menu` | ✅ `constants/menuCatalog.ts` |
| 隐藏后侧栏过滤 | ✅ `hidden.has(item.path)` |
| 子菜单全隐藏时父级消失 | ✅ 空 children 丢弃 |
| 中英文案 | ✅ `nav.menuManagement` + `admin.menuManagement.*` |

相关代码：

- 后端：`SettingKeyHiddenMenuKeys`，public/admin settings 透出与更新
- 前端：`views/admin/MenuManagementView.vue`
- 前端：`constants/menuCatalog.ts`
- 前端：`components/layout/AppSidebar.vue`（`finalizeNav`）
- 前端：`stores/adminSettings.ts`（`hiddenMenuKeys`）
- 前端：`i18n/locales/{zh,en}/admin/menuManagement.ts`

### 3.4 与 feature flag 的关系

- 既有 `featureFlags`（支付、渠道监控等）继续生效
- `hidden_menu_keys` 在此之上提供**任意内置菜单 path** 的显式隐藏
- 过滤顺序：`featureFlag` → `simple mode` → `hidden_menu_keys`

---

## 快速本地试跑（Docker，推荐）

详细步骤见：[`deploy/START_LOCAL.md`](./deploy/START_LOCAL.md)

### SQLite 单容器（最快）

```bash
cd deploy
cp .env.sqlite.example .env.sqlite
mkdir -p data
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build
```

- 访问：http://localhost:8080  
- 默认管理员：`admin@sub2api.local` / `admin123456`（见 `.env.sqlite`）

### PostgreSQL + Redis + 本地构建

```bash
cd deploy
cp .env.example .env   # 设置 POSTGRES_PASSWORD / ADMIN_PASSWORD / JWT_SECRET
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.build.yml --env-file .env up -d --build
```

### 相关 compose 文件

| 文件 | 说明 |
|------|------|
| `deploy/docker-compose.sqlite.yml` | 本地 build，SQLite + 嵌入式 Redis |
| `deploy/docker-compose.build.yml` | 本地 build，PostgreSQL + Redis |
| `deploy/.env.sqlite.example` | SQLite 模式环境变量模板 |
| `deploy/START_LOCAL.md` | 启动说明与排错 |

菜单验证步骤：

1. 管理员登录 → 侧栏应出现 **菜单管理**
2. 打开 `/admin/menu` → 关闭例如「使用记录」(`/usage`) → 保存
3. 侧栏「使用记录」应消失；刷新后仍保持隐藏
4. 系统设置页卡片可跳转到菜单管理

---

## 变更范围摘要

- 复制上游 `sub2api` → `sub2api_new`（不含 `.git`）
- 配置 / 仓储 / 安装向导：双数据库 + 可选 Redis
- 设置系统 + 独立菜单管理页 + 侧栏过滤
- `deploy/config.example.yaml`、`deploy/.env.example` 文档同步

## 关键文件索引

```
sub2api_new/
├── REFACTOR.md                          # 本文档
├── deploy/config.example.yaml           # driver/path、redis.enabled
├── deploy/.env.example                  # DATABASE_DRIVER、REDIS_ENABLED
├── backend/internal/config/config.go
├── backend/internal/repository/ent.go
├── backend/internal/repository/redis.go
├── backend/internal/setup/{setup,cli,handler}.go
├── backend/internal/service/domain_constants.go   # SettingKeyHiddenMenuKeys
├── frontend/src/constants/menuCatalog.ts
├── frontend/src/views/admin/MenuManagementView.vue
├── frontend/src/components/layout/AppSidebar.vue
├── frontend/src/router/index.ts
└── frontend/src/i18n/locales/{zh,en}/admin/menuManagement.ts
```
