# 本地启动指南（sub2api_new）

## 登录页

**未修改登录页。** 使用默认登录界面与原版相同。

---

## 不用 Docker？直接部署

见 **[START_NATIVE.md](./START_NATIVE.md)**（SQLite + 无外部 Redis + systemd，适合 1C1G）。

---

## 推荐：SQLite 单容器（最快验证重构）

无需 PostgreSQL / Redis，一条命令构建并启动。

```bash
cd /media/liang/工作/work_space/idea/github_code/change/sub2api_new/deploy

# 1. 环境变量
cp .env.sqlite.example .env.sqlite
# 建议改 ADMIN_PASSWORD、JWT_SECRET

# 2. 数据目录
mkdir -p data

# 3. 构建并启动（首次会较久：前端 + Go 编译）
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build

# 4. 看日志
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite logs -f
```

打开：http://localhost:8080

默认账号（见 `.env.sqlite`）：

| 项 | 默认值 |
|----|--------|
| 邮箱 | `admin@sub2api.local` |
| 密码 | `admin123456` |

登录后侧栏 → **菜单管理**（`/admin/menu`）验证隐藏菜单。

### 常用命令

```bash
# 停止
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite down

# 重建（代码改了之后）
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build

# 清空数据重来（会删库）
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite down
rm -rf data/*
docker compose -f docker-compose.sqlite.yml --env-file .env.sqlite up -d --build
```

---

## 完整栈：PostgreSQL + Redis + 本地构建

```bash
cd /media/liang/工作/work_space/idea/github_code/change/sub2api_new/deploy

cp .env.example .env
# 必填：POSTGRES_PASSWORD、ADMIN_PASSWORD、JWT_SECRET

mkdir -p data postgres_data redis_data

docker compose -f docker-compose.build.yml --env-file .env up -d --build
docker compose -f docker-compose.build.yml --env-file .env logs -f sub2api
```

访问：http://localhost:8080

---

## 文件说明

| 文件 | 用途 |
|------|------|
| `docker-compose.sqlite.yml` | 单机 SQLite + 嵌入式 Redis，本地 build |
| `.env.sqlite.example` | SQLite 模式环境变量模板 |
| `docker-compose.build.yml` | PG + Redis + 本地 build |
| `.env.example` | 完整栈环境变量（官方同款扩展） |
| `../Dockerfile` | 多阶段构建：前端 embed + Go 二进制 |

---

## 故障排查

1. **构建失败 / go mod 超时**  
   在 `.env.sqlite` 或 shell 里设置：
   ```bash
   export GOPROXY=https://goproxy.cn,direct
   export GOSUMDB=sum.golang.google.cn
   ```

2. **健康检查一直 unhealthy**  
   ```bash
   docker logs sub2api-new-sqlite
   ```
   常见原因：首次 AUTO_SETUP 建库较慢（可等 start_period），或 ADMIN/JWT 配置异常。

3. **登录后没有「菜单管理」**  
   确认用的是本地镜像 `sub2api-new:local`（不是 `weishaw/sub2api:latest`），且账号是 **管理员**。

4. **改完代码不生效**  
   必须 `--build` 重新构建镜像；仅 `up -d` 不会重编。
