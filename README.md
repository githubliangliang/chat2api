<div align="center">

<img src="assets/logo.svg" alt="Sub2API Logo" width="128" />

# Sub2API（本仓库 fork）

[![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![SQLite](https://img.shields.io/badge/SQLite-supported-003B57.svg)](https://www.sqlite.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

基于上游 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 的个人/单机向改动版。

上游项目地址：**https://github.com/Wei-Shaw/sub2api**

</div>

## 本仓库改动

相对上游，主要变化：

| 能力 | 说明 |
|------|------|
| **SQLite** | 可不依赖 PostgreSQL；迁移 SQL 已转为 SQLite 方言；启动时执行 `backend/migrations/*.sql` |
| **Redis 可选** | `REDIS_ENABLED=false` 时使用进程内嵌入式 Redis（仅单机） |
| **菜单可隐藏** | 设置项 `hidden_menu_keys` + 管理页「菜单管理」 |
| **用量落库** | 补齐 `usage_logs` 幂等索引与 `usage_billing_dedup` 表，避免请求成功但不写 usage |
| **支付页清理** | 已移除对下线接口 `/admin/payment/providers` 的前端请求 |

更细的重构说明见 [REFACTOR.md](./REFACTOR.md)。

> **注意**：当前 `backend/migrations/*.sql` 为 **SQLite 方言**。高并发 / 多实例生产环境仍建议上游 + PostgreSQL；本仓库优先单机 SQLite 部署。

---

## 快速部署（推荐：SQLite 单容器）

无需 PostgreSQL / 外部 Redis。

### 前置条件

- Docker 20.10+
- Docker Compose v2+

### 步骤

```bash
git clone <本仓库地址>
cd sub2api_new/deploy

# 首次
cp .env.sqlite.example .env.sqlite
# 建议修改 ADMIN_PASSWORD、JWT_SECRET（openssl rand -hex 32）

# 构建并启动
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build

# 日志
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite logs -f
```

### 访问

- 地址：`http://localhost:8080`（端口见 `.env.sqlite` 的 `SERVER_PORT`）
- 默认管理员：见 `.env.sqlite` 中 `ADMIN_EMAIL` / `ADMIN_PASSWORD`

### 数据位置

| 路径 | 内容 |
|------|------|
| `deploy/data/sub2api.db` | SQLite 数据库 |
| `deploy/data/config.yaml` | 运行配置 |
| `deploy/data/logs/` | 日志 |

### 常用命令

```bash
# 停止
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite down

# 重建（代码变更后）
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build

# 状态
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite ps
```

### 环境变量（节选）

| 变量 | 说明 |
|------|------|
| `DATABASE_DRIVER` | `sqlite`（本 compose 已写死） |
| `DATABASE_PATH` | 默认 `/app/data/sub2api.db` |
| `REDIS_ENABLED` | `false` = 嵌入式 Redis |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | 首次自动创建管理员 |
| `JWT_SECRET` | 会话密钥，重建容器建议固定 |
| `RUN_MODE` | `simple` 可跳过计费等 SaaS 能力（生产需配合确认项） |

完整示例见：`deploy/.env.sqlite.example`。

---

## 其他部署方式

### Docker Compose（PostgreSQL + Redis，接近上游）

与上游一致，文档见：

- https://github.com/Wei-Shaw/sub2api  

本仓库也可使用 `deploy/docker-compose.yml` / `docker-compose.local.yml`（需自备 `.env`）：

```bash
cd deploy
cp .env.example .env   # 按上游要求填写 POSTGRES_PASSWORD 等
docker compose -f docker-compose.local.yml up -d
```

### 从源码构建

```bash
# 前端
cd frontend && pnpm install && pnpm run build

# 后端（嵌入前端）
cd ../backend
VERSION="$(./scripts/resolve-version.sh 2>/dev/null || echo dev)"
go build -tags embed -ldflags="-X main.Version=${VERSION}" -o sub2api ./cmd/server

# 配置示例
cp ../deploy/config.example.yaml ./config.yaml
# 或 SQLite 个人配置：../deploy/config.personal.sqlite.yaml

./sub2api
```

SQLite 配置示例：

```yaml
database:
  driver: "sqlite"
  path: "./data/sub2api.db"

redis:
  enabled: false   # 嵌入式 Redis
```

---

## 技术栈（本仓库）

| 组件 | 技术 |
|------|------|
| 后端 | Go、Gin、Ent、`modernc.org/sqlite` |
| 前端 | Vue 3、Vite、TailwindCSS |
| 数据库 | **SQLite（推荐单机）** / PostgreSQL（上游主路径） |
| 缓存 | Redis 可选；可关外置、用进程内实现 |

---

## 上游与许可证

- 上游 Sub2API：https://github.com/Wei-Shaw/sub2api  
- 许可证：与上游一致，见 [LICENSE](./LICENSE)（LGPLv3）

本仓库在上游基础上做个人部署向改动；功能与合规风险说明请以**上游 README** 为准。
