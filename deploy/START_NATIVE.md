# 个人自用 · 少占内存（推荐）

目标：**一个人用、内存尽量小、能登录、能管账号/Key、能转发 API**。

方案：**单二进制 + SQLite + 无外部 Redis**（不要 PostgreSQL / Redis 进程）。

---

## 你现在怎么跑（本机已有容器时）

```bash
# 地址（不要和 8080 官方镜像混了）
http://127.0.0.1:8081/login

账号: admin@sub2api.local
密码: admin123456
```

重启/重建：

```bash
cd sub2api/deploy
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite logs -f
```

---

## 精简配置（已默认）

模板：`config.personal.sqlite.yaml`

| 项 | 值 | 作用 |
|----|-----|------|
| `run_mode` | `standard` | 完整分组/计费语义（推荐）；菜单用 `/admin/menu` 隐藏，详见根 README |
| `default.user_balance` | `100000` | standard 会校验余额，留 0 则调 API 报 403 |
| `database.driver` | `sqlite` | 不跑 PG |
| `redis.enabled` | `false` | 嵌入式 Redis，单机 |
| `ops.enabled` | `true` | 开错误/系统日志 + 每天 03:00 清理；聚合与采集缓存仍关 |
| `batch_image` | 关闭 | 关生图队列 |
| 连接池 | 2 | 少占连接 |

后台还会在 SQLite 下**自动跳过**一批 PG 专用轮询任务（outbox / 定时测试 / 清理等），减少日志刷屏和 CPU。

---

## 已关闭的页面入口

支付、邀请、兑换、公告管理、运维页、admin 仪表盘等（见 `REMOVED_PAGES.md`）。

个人常用保留：登录、API Key、账号、分组、渠道、用量、设置、菜单管理、网关转发。

---

## 能用 / 别指望

| ✅ 个人够用 | ⚠️ SQLite 仍弱 |
|------------|----------------|
| 登录、改密 | 部分高级统计 SQL |
| 账号 / 分组 / 渠道 | 复杂批量 ANY()/COPY |
| API Key 增删 | 部分定时调度语义 |
| 改余额、用户列表 | 完整多实例一致性 |
| 设置、菜单管理 | 与官方 PG 全功能 1:1 |

若你发现「管理后台每个高级按钮都要稳」，请改用 PostgreSQL（内存会高一些）。

---

## 非 Docker 安装（1C1G 服务器）

1. **在电脑上编译**（不要在 1G 机器上 build）：

```bash
cd sub2api
cd frontend && pnpm install && pnpm build && cd ..
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed -ldflags="-s -w" -o ../sub2api ./cmd/server
```

2. **上传** `sub2api` 到服务器 `/opt/sub2api/sub2api`

3. **配置**：

```bash
sudo mkdir -p /opt/sub2api/data
sudo cp deploy/config.personal.sqlite.yaml /opt/sub2api/data/config.yaml
# 改 jwt.secret、admin 密码
sudo cp deploy/sub2api-sqlite.service /etc/systemd/system/sub2api.service
sudo systemctl enable --now sub2api
```

4. 浏览器：`http://IP:8080`

---

## 一键发布（推荐，全程免密）

本地改完代码后，一条命令完成：构建 → 打包 → 上传 → 重启 → 健康检查（失败自动回滚）。

```bash
cp deploy/deploy.env.example deploy/deploy.env   # 填 REMOTE_HOST / REMOTE_USER / REMOTE_DIR
deploy/deploy-remote.sh --setup-key              # 装公钥，只需输入一次服务器密码
deploy/deploy-remote.sh                          # 之后每次发布都不用输密码
```

常用参数：

| 参数 | 作用 |
|------|------|
| `--skip-frontend` | 只改了后端，复用已有 `backend/internal/web/dist` |
| `--skip-build` | 直接发布现有 `./sub2api` |
| `--package-only` | 只本地打包不连服务器，产物在 `dist/` |
| `--no-restart` | 只换二进制不重启 |
| `--dry-run` / `-y` | 预演 / 跳过确认 |

行为约定：

- 远端**已有** `data/config.yaml` 与 systemd unit 时不会被覆盖；要覆盖得显式加 `--force-config` / `--force-unit`，且旧文件自动备份。
- 换二进制前先在服务器上跑一次 `sub2api -version` 自检，架构不对直接中止，不动线上服务。
- 旧二进制存为 `sub2api.prev`，历史版本留在 `releases/`（默认保留 3 个）。
- 重启后轮询 `127.0.0.1:$APP_PORT/health`，超时（默认 60s）自动回滚并打印 `journalctl` 最后 40 行。
- 免密要两件事：SSH 免密（密钥，`--setup-key` 一次搞定）+ 远端免密 sudo（或直接用 root）。

等价的 Makefile 入口：`make deploy ARGS="--skip-frontend -y"`、`make deploy-package`。

---

## 备份

整目录 `/opt/sub2api/data`（含 `sub2api.db`、`config.yaml`）。

---

## 校验清单

1. 打开登录页，登录成功  
2. 侧栏无支付/邀请/运维页  
3. 账号列表能打开  
4. 用户改余额成功  
5. 菜单管理 `/admin/menu` 能开  
6. 日志里不应再疯狂刷 outbox SQL 错误（SQLite 下已停这些 worker）  
