<div align="center">

<img src="assets/logo.svg" alt="Sub2API Logo" width="128" />

# Sub2API（本仓库 fork）

[![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![SQLite](https://img.shields.io/badge/SQLite-only-003B57.svg)](https://www.sqlite.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

基于上游 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 的个人/单机向改动版。

上游项目地址：**https://github.com/Wei-Shaw/sub2api**

</div>

## 本仓库改动

相对上游，主要变化：

| 能力 | 说明 |
|------|------|
| **SQLite only** | 运行时只打开 SQLite（`modernc.org/sqlite`）。`database.driver` / `DATABASE_DRIVER` 即使写成 `postgres` 也会被忽略 |
| **安装向导** | Web / CLI / `AUTO_SETUP` 只收集 SQLite 文件路径，不再要 host/port/user/ssl |
| **备份** | 管理页备份用 `VACUUM INTO` 打 `*.db.gz`；镜像不再带 `pg_dump` / `psql` |
| **Redis 可选** | `REDIS_ENABLED=false` 时使用进程内嵌入式 Redis（仅单机）；Web 向导第 2 步有「使用外部 Redis」开关，默认关闭 |
| **菜单可隐藏** | 设置项 `hidden_menu_keys` + 管理页「菜单管理」（`/admin/menu`），侧边栏入口可逐项隐藏 |
| **用量落库** | 补齐 `usage_logs` 幂等索引与 `usage_billing_dedup` 表，避免请求成功但不写 usage |
| **支付页清理** | 已移除对下线接口 `/admin/payment/providers` 的前端请求 |

延伸文档：

| 想看 | 去哪 |
|------|------|
| 重构细节 | [REFACTOR.md](./REFACTOR.md) |
| 1C1G 精简说明 | [deploy/START_NATIVE.md](./deploy/START_NATIVE.md) |
| 开发约定与踩坑 | [DEV_GUIDE.md](./DEV_GUIDE.md) |
| 已下线页面 | [REMOVED_PAGES.md](./REMOVED_PAGES.md) |
| 合上游（**不要**整仓 merge） | [docs/upstream-sync/README.md](./docs/upstream-sync/README.md) |

> **注意**：本 fork **不能**连 PostgreSQL。`backend/migrations/*.sql` 是 SQLite 方言；已应用的 `SELECT 1` no-op 迁移不要删（checksum 不可变）。高并发 / 多实例请回[上游仓库](https://github.com/Wei-Shaw/sub2api)。

---

## 选哪条路

| 场景 | 走 |
|------|-----|
| 一台 1C1G VPS 自己用 | **方式一**：单二进制 + SQLite + 嵌入式 Redis |
| 本机 / NAS 上已有 Docker | **方式二**：SQLite 单容器 |
| 多实例、高并发、团队用 | 回[上游仓库](https://github.com/Wei-Shaw/sub2api)（PostgreSQL + 外部 Redis） |

两条路都要先定 `run_mode`，**推荐 `standard`**，见下一节。

---

## run_mode：推荐 standard

合法值只有 `standard` 和 `simple`，写别的会被 `config.NormalizeRunMode` 归一成 `standard`。程序默认值、`deploy/config.example.yaml`、`deploy/.env.sqlite.example` 都是 `standard`。

### 两种模式的实际差别

| 行为 | `standard`（推荐） | `simple` |
|------|-------------------|----------|
| 余额 / 订阅校验 | 生效：余额 ≤ 0 或低于 `billing.minimum_balance_reserve`（默认 `0.000001`）→ `403 INSUFFICIENT_BALANCE` | 整块跳过 |
| 用量与扣费 | 落 `usage_logs` + 按 token 成本实扣余额 | 只落 `usage_logs`（成本照算、写进日志，但不动余额，日志打 `[SIMPLE MODE] Usage recorded (not billed)`） |
| 分组调度 | 按分组隔离：Key 绑了分组 → 只调该分组的账号；Key 没绑分组 → 只调**不属于任何分组**的账号 | 忽略分组，按平台捞所有可调度账号 |
| 启动副作用 | 无 | 自动补 `anthropic-default` / `openai-default` / `gemini-default` / `grok-default` / `antigravity-default-1,2` 分组；把并发仍是 5 的管理员一次性提到 30 |
| 侧边栏 | 全量 | 隐藏 用户、分组、渠道管理、订阅、审计日志，以及用户侧的 用量统计、批量生图、可用渠道 |
| 直接输 URL | 都能进 | `/admin/groups`、`/admin/subscriptions`、`/admin/redeem`、`/subscriptions`、`/redeem` 被重定向回首页（`/admin/users` 只是不在侧边栏，仍可直达） |
| `/v1/sub2api/billing` | 可用 | 404 |

### 为什么个人自用也建议 standard

- **分组是这套系统里唯一的账号隔离 / 定向手段**（分组模型映射、分组限额、Key 绑分组）。`simple` 下这些配置能填、能存、UI 上看着生效，但**不参与调度**——账号一多，排查最费时间的就是这个。
- 用量统计、审计日志、用户管理这些"看清楚到底发生了什么"的页面，`simple` 下从侧边栏消失。
- 嫌菜单多是 UI 问题：本 fork 有 `hidden_menu_keys` + `/admin/menu` 可以逐项隐藏，比整体降级成 `simple` 精准得多。
- `simple` 真正省的只有"不用管余额"这一件事，而那就是下面一步配置。

### standard 必做一步：给自己余额

standard 会校验余额，而管理员创建时余额取 `default.user_balance`（上游默认 `0`）。不给余额就是首次调 API 直接 `403 INSUFFICIENT_BALANCE`。

- **首次启动前**：在 `config.yaml` 把 `default.user_balance` 设成够用的数（`deploy/config.personal.sqlite.yaml` 已给 `100000`），管理员按这个值创建。
  - 它同时是**新注册用户**的初始余额。要开放注册就改回 `0`，再用下面那条单独给自己充。
- **已经启动过、或 Docker `AUTO_SETUP` 装的**（这条路的管理员余额固定是 `0`）：登录后 管理页 → **用户管理**（`/admin/users`）→ 给自己加余额（接口 `POST /admin/users/:id/balance`）。

余额按 token 成本实扣，扣到 0 会重新 403，再加即可；`/admin/usage` 能看到花了多少。

### 什么时候才选 simple

只想"能转发就行"、完全不看账、也不用分组隔离账号的时候。切回 `standard` 前留意两件事：`simple` 期间的余额从没被动过（用量日志里有算出来的成本，但没有扣费流水），切过去以后才开始真扣；`simple` 启动时自动建的默认分组会留在库里（这些分组是空的，Key 不绑分组仍走"无分组账号"那条路，不影响 standard）。

---

## 部署

### 方式一：1C1G 无 Docker（推荐个人服务器）

目标：一个人用、内存尽量小。**不要在 1G 机器上编译**，在本机编好二进制再上传。

方案：单二进制 + SQLite + 无外部 Redis（不跑 PostgreSQL / Redis 进程）。

本机需要：Go `1.26.5`（见 `backend/go.mod`）、Node 20、**pnpm**（不要用 npm）。

#### 1. 本机编译

```bash
git clone <本仓库地址>
cd sub2api

# 前端
cd frontend
pnpm install
pnpm run build
cd ..

# 后端（嵌入前端，strip 体积）
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags embed -ldflags="-s -w" \
  -o ../sub2api ./cmd/server
```

得到仓库根目录的 `sub2api` 单文件。

> ARM 服务器把 `GOARCH=amd64` 改成 `arm64`。

#### 2. 上传到服务器

```bash
# 本机
scp sub2api root@你的服务器:/tmp/sub2api

# 服务器
sudo useradd -r -s /usr/sbin/nologin sub2api 2>/dev/null || true
sudo mkdir -p /opt/sub2api/data
sudo mv /tmp/sub2api /opt/sub2api/sub2api
sudo chmod +x /opt/sub2api/sub2api
sudo chown -R sub2api:sub2api /opt/sub2api
```

#### 3. 配置

```bash
# 把仓库里的模板拷上去
sudo cp deploy/config.personal.sqlite.yaml /opt/sub2api/data/config.yaml
sudo nano /opt/sub2api/data/config.yaml
```

**必改：**

```yaml
jwt:
  secret: "用 openssl rand -hex 32 生成"

totp:
  encryption_key: "用 openssl rand -hex 32 生成（必须 64 位 hex）"

default:
  admin_email: "你的邮箱"
  admin_password: "强密码"
  user_balance: 100000    # 模板已给这个值；standard 模式必须 > 0，否则第一次调 API 就是 403
```

> `totp.encryption_key` 留空时程序**每次启动都会随机生成一个**：重启后所有已绑定的 2FA 立即失效，支付 resume token 也不可用。部署前就填好，之后别再改（换掉 = 作废所有已有绑定）。格式必须是 32 字节 hex，写错服务会直接启动失败。
> 用 `deploy/deploy-remote.sh` 首次部署时，该密钥会和 `jwt.secret` 一起自动随机生成，不用手填。

模板已默认的关键项：

| 项 | 值 | 作用 |
|----|-----|------|
| `run_mode` | `standard` | 完整分组 / 计费语义；不想看到的菜单用 `/admin/menu` 逐项隐藏 |
| `default.user_balance` | `100000` | 管理员（以及新注册用户）初始余额，standard 下必须 > 0 |
| `default.user_concurrency` | `10` | 管理员 / 新用户并发；不够在用户管理里改 |
| `database.driver` | `sqlite` | 兼容字段，运行时一律按 sqlite |
| `database.path` | `/opt/sub2api/data/sub2api.db` | 库文件 |
| `redis.enabled` | `false` | 嵌入式 Redis，单机 |
| `ops.enabled` | `true` | 开错误/系统日志与每天 03:00 定时清理；聚合 `aggregation`、采集缓存仍关，省 CPU |
| 连接池 | 2 | 省内存 |

配置模板：`deploy/config.personal.sqlite.yaml`。

#### 4. systemd 开机自启

```bash
sudo cp deploy/sub2api-sqlite.service /etc/systemd/system/sub2api.service
sudo systemctl daemon-reload
sudo systemctl enable --now sub2api
sudo systemctl status sub2api
sudo journalctl -u sub2api -f
```

单元文件：`deploy/sub2api-sqlite.service`。

- 工作目录：`/opt/sub2api`
- 数据：`DATA_DIR=/opt/sub2api/data`
- 监听：`0.0.0.0:8080`
- 可选内存上限：在 unit 里取消注释 `# MemoryMax=768M`

#### 5. 访问

浏览器：`http://服务器IP:8080`  
用配置里的管理员邮箱/密码登录。

防火墙放行 8080（或前面挂 Nginx 反代 80/443）。

#### 6. 备份 / 升级

- **停机拷贝**：整目录 `/opt/sub2api/data`（`sub2api.db`、`sub2api.db-wal`、`sub2api.db-shm`、`config.yaml`）
  - `config.yaml` 里有 `jwt.secret` 和 `totp.encryption_key`，丢了等于所有会话失效 + 已绑定的 2FA 全部作废，必须一起备份
- **管理页备份**（配好 S3 后）：`VACUUM INTO` 打一致性快照，对象名 `*.db.gz`；恢复会替换线上 `.db` 并删掉 `-wal`/`-shm`
- **升级**：本机重新 `go build` → 上传覆盖 `/opt/sub2api/sub2api` → `sudo systemctl restart sub2api`

#### 一键发布（可选）

```bash
cp deploy/deploy.env.example deploy/deploy.env   # 填 VPS 信息
deploy/deploy-remote.sh --setup-key              # 一次性装 SSH 公钥，只输一次密码
deploy/deploy-remote.sh                          # 之后每次：构建 → 上传 → 重启 → 健康检查，失败自动回滚
```

只换二进制 + 重启，远端已有的 `config.yaml` 与 systemd unit 不会被覆盖。

#### 不建议在 1G 机上做的事

- 在服务器上跑 `pnpm build` + `go build`（容易 OOM）
- 装 PostgreSQL + Redis 全栈
- 开 `ops.aggregation` / `ops.use_preaggregated_tables` / 采集缓存、批量生图队列、高并发连接池
  - 模板里这些都是关的；`ops.enabled: true` 只开错误/系统日志与每天 03:00 清理，开销很小

#### 校验清单

1. 登录成功  
2. API Key / 账号 / 分组能开  
3. 调一次 API 成功，且没有 `403 INSUFFICIENT_BALANCE`（报了就去补余额）  
4. 用量能落库（`/admin/usage` 有记录、有花费）  
5. 菜单管理 `/admin/menu` 能开  
6. `journalctl -u sub2api` 无疯狂刷屏 SQL 错误  

---

### 方式二：Docker Compose（SQLite 单容器）

无需 PostgreSQL / 外部 Redis。

#### 前置条件

- Docker 20.10+
- Docker Compose v2+

#### 步骤

```bash
git clone <本仓库地址>
cd sub2api/deploy

# 首次
cp .env.sqlite.example .env.sqlite
# 建议修改 ADMIN_PASSWORD、JWT_SECRET（openssl rand -hex 32）

# 构建并启动
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build

# 日志
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite logs -f
```

#### 访问

- 地址：`http://localhost:8080`（端口见 `.env.sqlite` 的 `SERVER_PORT`）
- 默认管理员：见 `.env.sqlite` 中 `ADMIN_EMAIL` / `ADMIN_PASSWORD`

> `AUTO_SETUP` 创建的管理员余额固定是 `0`（并发 5）。`RUN_MODE=standard`（默认）下第一次调 API 会 `403 INSUFFICIENT_BALANCE`：登录后到 `/admin/users` 给自己加余额、按需调并发。

#### 数据位置

| 路径 | 内容 |
|------|------|
| `deploy/data/sub2api.db` | SQLite 数据库 |
| `deploy/data/config.yaml` | 运行配置 |
| `deploy/data/logs/` | 日志 |

#### 常用命令

```bash
# 停止
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite down

# 重建（代码变更后）
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build

# 状态
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite ps
```

#### 环境变量（节选）

| 变量 | 说明 |
|------|------|
| `RUN_MODE` | `standard`（默认，推荐）/ `simple`；差别见上文 [run_mode](#run_mode推荐-standard) 一节 |
| `DATABASE_PATH` | 默认 `/app/data/sub2api.db`（`DATABASE_DRIVER` 会被忽略） |
| `REDIS_ENABLED` | `false` = 嵌入式 Redis |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | 首次 `AUTO_SETUP` 创建管理员 |
| `JWT_SECRET` | 会话密钥，重建容器建议固定 |
| `TOTP_ENCRYPTION_KEY` | 用 2FA 就必须固定，否则重启作废所有绑定 |
| `SERVER_PORT` / `BIND_HOST` | 宿主机映射，默认 `8080`；本机被占时可改 `8081` |

完整示例见：`deploy/.env.sqlite.example`。

`deploy/docker-compose.yml` / `docker-compose.local.yml` 等是上游遗留文件，本 fork 进程仍只开 SQLite，个人部署不要用它们。

---

## 常见问题

| 现象 | 原因 / 处理 |
|------|------------|
| 调 API 报 `403 INSUFFICIENT_BALANCE` | standard 下余额不足。`/admin/users` 给自己加余额；或首次启动前就设好 `default.user_balance` |
| 一直调度不到账号 / 报没有可用账号 | standard 按分组隔离：Key 绑了分组只用该分组的账号，Key 没绑分组只用**没进任何分组**的账号。对齐 Key 与账号的分组绑定 |
| 改了 `config.yaml` 的 `admin_password`，登录还是旧密码 | `default.admin_*` 只在库里 0 个用户时用来创建管理员，之后不做任何同步。改密走 UI 个人资料页，或直接改 `users` 表 |
| 重启后所有 2FA 失效 | `totp.encryption_key` 没固定，程序每次启动随机生成。填死一个 32 字节 hex 并随 `config.yaml` 一起备份 |
| 启动直接退出，日志说 `default.admin_password must be changed from the example value` | 模板里的 `CHANGE_ME_STRONG_PASSWORD` 没改。改成真密码（8–128 字符）再启动 |
| 启动报迁移 checksum 不一致 | 改过已经应用的迁移文件。加新迁移文件解决；个人库可以直接删 `*.db` 重来 |
| 登录 503 | 老库缺辅助表（如 `user_allowed_groups`）。看日志缺哪张表，`EnsureSQLiteAuxTables` 的 DDL 必须与迁移列对列一致 |

---

## 技术栈（本仓库）

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.26.5、Gin、Ent、`modernc.org/sqlite` |
| 前端 | Vue 3、Vite、TailwindCSS（pnpm） |
| 数据库 | **SQLite only**（`backend/migrations/*.sql` 为 SQLite 方言） |
| 缓存 | Redis 可选；关掉外置就用进程内嵌入式实现（仅单机） |
| 分发 | 单二进制（`-tags embed` 内嵌前端）/ Docker Compose |

---

## 上游与许可证

- 上游 Sub2API：https://github.com/Wei-Shaw/sub2api  
- 许可证：与上游一致，见 [LICENSE](./LICENSE)（LGPLv3）

本仓库在上游基础上做个人部署向改动；功能与合规风险说明请以**上游 README** 为准。
