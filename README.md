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
1C1G 精简说明另见 [deploy/START_NATIVE.md](./deploy/START_NATIVE.md)。

> **注意**：当前 `backend/migrations/*.sql` 为 **SQLite 方言**。高并发 / 多实例生产环境仍建议上游 + PostgreSQL；本仓库优先单机 SQLite 部署。

---

## 部署

### 方式一：1C1G 无 Docker（推荐个人服务器）

目标：一个人用、内存尽量小。**不要在 1G 机器上编译**，在本机编好二进制再上传。

方案：单二进制 + SQLite + 无外部 Redis（不跑 PostgreSQL / Redis 进程）。

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

default:
  admin_email: "你的邮箱"
  admin_password: "强密码"
```

模板已默认的关键项：

| 项 | 值 | 作用 |
|----|-----|------|
| `run_mode` | `simple` | 弱化 SaaS/计费 |
| `database.driver` | `sqlite` | 不跑 PG |
| `database.path` | `/opt/sub2api/data/sub2api.db` | 库文件 |
| `redis.enabled` | `false` | 嵌入式 Redis，单机 |
| `ops.enabled` | `false` | 关运维采集，省 CPU |
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

#### 6. 1C1G 建议开 Swap

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

#### 7. 备份 / 升级

- **备份**：整目录 `/opt/sub2api/data`（含 `sub2api.db`、`config.yaml`）
- **升级**：本机重新 `go build` → 上传覆盖 `/opt/sub2api/sub2api` → `sudo systemctl restart sub2api`

#### 不建议在 1G 机上做的事

- 本机 `pnpm build` + `go build`（容易 OOM）
- 装 PostgreSQL + Redis 全栈
- 开 `ops`、批量生图队列、高并发连接池

#### 校验清单

1. 登录成功  
2. API Key / 账号 / 分组能开  
3. 调一次 API，用量能落库  
4. 菜单管理 `/admin/menu` 能开  
5. `journalctl -u sub2api` 无疯狂刷屏 SQL 错误  

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
| `DATABASE_DRIVER` | `sqlite`（本 compose 已写死） |
| `DATABASE_PATH` | 默认 `/app/data/sub2api.db` |
| `REDIS_ENABLED` | `false` = 嵌入式 Redis |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | 首次自动创建管理员 |
| `JWT_SECRET` | 会话密钥，重建容器建议固定 |
| `RUN_MODE` | `simple` 可跳过计费等 SaaS 能力（生产需配合确认项） |

完整示例见：`deploy/.env.sqlite.example`。

---

### 方式三：Docker Compose（PostgreSQL + Redis，接近上游）

与上游一致，文档见：

- https://github.com/Wei-Shaw/sub2api  

本仓库也可使用 `deploy/docker-compose.yml` / `docker-compose.local.yml`（需自备 `.env`）：

```bash
cd deploy
cp .env.example .env   # 按上游要求填写 POSTGRES_PASSWORD 等
docker compose -f docker-compose.local.yml up -d
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
