# 移植清单：上游 v0.1.179

对照日期：2026-08-21。本文所有「本仓库现状」结论均在 `665a959` 上逐条 grep / 实测核实，行号以该提交为准。
合完一项就把状态改成「已合」。**第 4 节 P0 五条已于 2026-08-21 全部合入**（记录见 4.6），其余待合。

通用流程见 [README.md](./README.md)，上一轮清单见 [PORTING-0.1.176.md](./PORTING-0.1.176.md)（标题写 0.1.177，P0–P3 已全合）。

移植前先读 README [第 4 节「硬约束」](./README.md#4-硬约束)，尤其 9–12 条；写 SQL 对照 [第 5 节转换速查](./README.md#5-pg--sqlite-转换速查)。

---

## 1. 版本对照

| 点 | 状态 |
|---|---|
| 上游最新正式版 | [`v0.1.179`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.179)（2026-08-20 07:06 UTC） |
| 上一轮对照基线 | `v0.1.177`（2026-08-15） |
| `v0.1.178` → `v0.1.179` | 78 commits / 212 files |
| `v0.1.177` → `v0.1.179` | 185 commits / 300 files |
| `v0.1.179` → `main` | ahead 32 commits / 57 files（一整串 Grok 4.6 默认模型迁移，见第 11 节） |
| 本仓库版本号 | `backend/cmd/server/VERSION` = `1.1.2`（自有编号，不要改成 0.1.179） |
| 本仓库迁移号 | 已用到 `224`；上游 226/227/228 在这边要顺延成 **225 起** |

对比链接：

- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.178...v0.1.179>
- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.177...v0.1.179>

**结论：78 个 commit 里适合做进来的约 10 条，其中 4 条是「本仓库确实存在同一个 bug」的直接补丁。**

---

## 2. 基线缺口：0.1.178 的供应商底座没进来

这是判断 0.1.179 大量条目「N/A」的唯一依据，先摆在前面。

| 底座 | 上游 | 本仓库 | 核实方式 |
|---|---|---|---|
| Kimi / 智谱 GLM / DeepSeek 平台 | 0.1.178 新增 | **没有** | `internal/domain/constants.go:20-27` 只有 anthropic / openai / gemini / antigravity / grok / composite |
| 渠道监控配额模式 | 0.1.178 新增 | **没有** | `channel_monitor_quota_fetcher.go`、`cn_provider_balance_service.go`、`cn_provider_quota_service.go`、`domain/channel_monitor_quota.go` 全部不存在 |
| 账号 `api_protocol` 字段 | 0.1.179 新增 adaptive | **没有** | 全仓 grep `api_protocol` / `APIProtocol` 零命中 |
| Composite 分组 | 上游有 | **有** | `PlatformComposite`、`composite_platform.go`、`composite_route_resolver.go` 齐全 |

0.1.178 里已经进来的：按上游服务层级计费（`00a818f` / `6eedaa4`）、部分用量与流恢复修复（`d622ec7`）。CN 供应商与配额模式两块是整体缺失。

---

## 3. 建议顺序

```text
① P0 四条 + 清 Sora            已合（2026-08-21）
② WS 两条（82cbe6aff → b94e484e2）
③ Grok 两组（内联图片 → tool_search；后者必须排在 ② 之后）
④ OpenAI 容量过载两条
⑤ 内置价卡补充（先核官方价，别抄倍率结构）
⑥ 按需：composite-codex / 代理探测目标 / input_tokens 预检
```

第 9 节的「长上下文计费门控 AND → OR」是**破坏性变更**，独立决策，不排在流程里。

---

## 4. P0 — 已核实本仓库有同一缺陷，补丁小且 patch site 逐字对得上

### 4.1 grok-4.6 的 `xhigh` 被降级成 `high`

上游 `892787723`（PR #5815），2 文件 +29-13。状态：**已合**（2026-08-21）

本仓库现状：`internal/service/openai_gateway_grok.go:625` 就是 `case "xhigh", "extrahigh", "max", "ultra": return "high", true`，与上游改前逐字一致。fork 已合过 grok-4.6 目录（见 0.1.176 清单 P0），这条是补票。

| 文件 | 动作 | 改什么 |
|---|---|---|
| `backend/internal/service/openai_gateway_grok.go` | 改 | `normalizeGrokReasoningEffortValue(raw)` → `(raw, model)`，三个调用点传 `upstreamModel`；`xhigh`/`extrahigh` 分支加 `grokSupportsXHighReasoningEffort(model)`；新增该 helper（只认 `grok-4.6` / `grok-4.6-latest`）。`max`/`ultra` 仍降 `high` |
| `backend/internal/service/openai_gateway_grok_test.go` | 改 | 用例表加 `upstreamModel` 列 |
| `backend/internal/handler/gateway_handler.go` | 可选 | `grokModelSupportsConfigurableReasoning`（:1256）吐的 `/models` 选项仍只有 low/medium/high。上游这一版没动，要不要给 4.6 加 xhigh 选项自行决定 |

### 4.2 Chat 非流式缓冲读取失败未触发故障转移

上游 `b228b93e9`（PR #5801），3 文件 +209-2。状态：**已合**（2026-08-21）

上游响应体读取失败（`unexpected EOF` / HTTP/2 stream reset）时直接返回原始错误，退化成通用 502，不 failover。

本仓库现状：patch site 完全对得上 —— `openai_gateway_chat_completions.go:416` 就是 `return nil, err`，`openai_gateway_messages.go:779` 就是 `return nil, usage, acc, ev.err`。依赖的 `shouldClassifyOpenAIUpstreamStreamReadError` / `newOpenAIUpstreamStreamReadError` / `OpenAIUpstreamStreamReadErrorDetails` / `newOpenAIStreamFailoverError` / `OpenAIUpstreamStreamReadErrorCode` / `OpenAIUpstreamHTTP2StreamErrorCode` 六个 helper **本仓库全有**，只差新增的 `openAICompatBufferedReadError` 类型。零冲突。

| 文件 | 动作 | 改什么 |
|---|---|---|
| `backend/internal/service/openai_gateway_messages.go` | 改 | 新增 `openAICompatBufferedReadError{cause}`（含 `Error()`/`Unwrap()`）；`readOpenAICompatBufferedTerminal` 的 `ev.err` 返回处包一层；Anthropic 侧解包后返回原 cause（保持既有行为） |
| `backend/internal/service/openai_gateway_chat_completions.go` | 改 | `import "bufio"`；`return nil, err` → `newOpenAICompatBufferedReadFailoverError(...)`；新增该方法（`bufio.ErrTooLong` 不重放，分类失败回落原错误，成功则造 `UpstreamFailoverError` 并保留稳定错误码） |
| `backend/internal/service/openai_gateway_compat_buffered_read_failover_test.go` | 新建 | 上游 152 行原样可用 |

### 4.3 本地 404 model_not_found 不再计入 SLA

上游 `6b0ec50f2`（PR #5876），5 文件 +167-13。状态：**已合**（2026-08-21）

本仓库现状：已有 `classifyNoAccountError` / `noAccountErrorClassification.ModelNotFound`，十余处 handler 用 `if !cls.ModelNotFound` 决定要不要写 ops 错误。上游这条是把它标成业务限制并清掉上游归因残留。三个目标文件都在。

| 文件 | 动作 |
|---|---|
| `backend/internal/handler/no_account_error.go` | 改（+5-1） |
| `backend/internal/handler/ops_error_logger.go` | 改（+23-5） |
| `backend/internal/service/ops_upstream_context.go` | 改（+20-7） |
| `backend/internal/handler/no_account_error_test.go` / `ops_error_logger_test.go` | 改 |

### 4.4 用量统计筛选一致性（从上游 perf 提交里只拆这一半）

上游 `a9514a68d` 的一部分。状态：**已合**（2026-08-21；性能那一半见第 6 节，不合）

**本仓库确实有这个 bug**：`usage_log_repo_stats.go:666 GetStatsWithFilters` 把 `filters.UpstreamModelMismatch` 加进了总计 SQL，但

- `:824 getEndpointStatsByColumnWithFilters(...)`
- `:892 getEndpointPathStatsWithFilters(...)`

两个函数签名里**根本没有 `upstreamModelMismatch` 参数**。于是在管理端按「上游模型不一致」筛选时，总计变了、端点/上游端点/路径三个分项不变，页面自相矛盾。

修法与 GROUPING SETS 无关：把参数透传下去，复用已有的 `upstreamModelMismatchCondition()`（`usage_log_repo_trend.go` 里已经这么用了）。约 30 行。

### 4.5 顺手：清 Sora 残留

上游 `7e45634df`（PR #5749），10 文件 +0-372，纯删除。状态：**已合**（2026-08-21）

本仓库残留确认存在：`backend/internal/handler/dto/settings.go:402 SoraClientEnabled`、`frontend/src/i18n/locales/{zh,en}/admin/{overview,settings}.ts` 里的 `soraStorageQuota` / `soraClient` / `soraS3` 三组文案。上游还删了 `deploy/config.example.yaml` 130 行和三个 README 段落，本仓库对应位置要各自核一遍再删。

### 4.6 合入记录（2026-08-21）

五条一起落在工作区，`19 files changed, +287 -395`，另新增两个测试文件（253 行）。相对上游的偏离都在下面列清楚了。

| 项 | 相对上游的偏离 |
|---|---|
| 4.1 | 上游改测试时把 `max camel` 用例删了（`max`→`high` 的 camelCase 路径失去覆盖）。这边**保留**该用例并把模型改成 `grok-4.6`，顺带证明 `max`/`ultra` 即使在 4.6 上也仍降 `high`；另补一条 `xhigh` 在 4.5 上仍降 `high` 的 chat 断言 |
| 4.2 | 无。两个生产文件按上游原样改，测试文件取上游 152 行原文 |
| 4.3 | 上游只在主中间件路径挂 `suppressOpsUpstreamAttributionForLocalModelConfiguration`，这边照做。本仓库另有一处 `applyOpsUpstreamFieldsFromContext` 调用（`ops_error_logger.go` 的 in-band SSE 恢复路径，status 200）——那条路径的上游归因是真的，不该清；它的分类仍走 `classifyOpsErrorLog`，所以 SLA 口径已被覆盖 |
| 4.4 | **不是移植，是自己修**。上游靠 GROUPING SETS 重写顺带解决，这边只给两个 helper 加了一个尾参 `upstreamModelMismatch *bool`（与 `usage_log_repo_trend.go` 既有写法一致），三个 `GetStatsWithFilters` 调用点透传 `filters.UpstreamModelMismatch`，两个导出包装器传 `nil` 保持原语义。新建 `usage_log_repo_stats_mismatch_filter_test.go`（sqlmock，101 行）：正向断言四条 SQL 都带 `IS TRUE`/`IS FALSE`；反向用记录式 matcher 断言不传筛选时四条 SQL 都不含该列（Go 的 RE2 没有负向断言，写不了 `(?!...)`） |
| 4.5 | 三个 Sora 迁移（`045` / `047` / `090_drop_sora.sql`）**没动**——已应用文件改不得（硬约束第 2 条）。`backend/internal/web/dist` 里还有旧 i18n 字符串，但它被 `.gitignore` 忽略，构建时重新生成，不手改。前端 4 个 locale 文件删除行数与上游一一对上（settings 各 -98、overview 各 -2） |

自测：

```bash
cd backend
go build ./...
go test -tags=unit ./internal/service/ ./internal/handler/... ./internal/repository/ ./internal/server/... -count=1
go test          ./internal/service/ ./internal/handler/... ./internal/repository/ ./internal/server/... -count=1
go vet           ./internal/service/ ./internal/handler/... ./internal/repository/
cd ../frontend && pnpm run typecheck && pnpm run lint:check
cd .. && make test-frontend-critical
```

全绿。前端全量 `pnpm run test:run` 有 1 个文件 5 条失败（`src/composables/__tests__/useRoutePrefetch.spec.ts`），
**与本次无关**：`git stash` 掉 4 个 locale 改动后在 `25402b2` 上复现同样的 5 条失败。

---

## 5. P1 — 值得做，改动面中等

### 5.1 Responses WebSocket http_bridge 两条

状态：**待合**。两条动同一个文件，必须一起做、按顺序。

| 上游 commit | 内容 | 规模 |
|---|---|---|
| `82cbe6aff`（#5845） | 后续轮次遇上游 429 无法切账号 | 9 文件 +461-20 |
| `b94e484e2`（#5822） | 多轮会话丢失首轮客户端工具映射 + item id 不被上游接受 | 5 文件 +286-6 |

**本仓库确实有 429 这个 bug**：`internal/service/openai_ws_http_bridge.go` 的 254 / 257 / 422 行都是 `turn == 1 && shouldFailover`，第二轮起 failover 分支根本进不去。

涉及文件本仓库全有：`openai_ws_http_bridge.go`、`openai_ws_forwarder.go`、`openai_ws_forwarder_ingress.go`、`openai_ws_forwarder_payload.go`、`openai_gateway_service.go`、`handler/openai_gateway_handler.go`、`pkg/apicompat/responses_client_tools.go`、`openai_gateway_grok_tool_protocol.go`。

对实际用 Codex WS 的场景这两条价值最高。

### 5.2 Grok 内联图片 / tool_search 两组

状态：**待合**。**必须排在 5.1 之后**——`9ede0f716` 动 `openai_ws_http_bridge.go` +52-13，先做会撞。

| 上游 commit | 内容 | 规模 | 备注 |
|---|---|---|---|
| `99a8b8470` + `b0cdea303`（#5844） | 内联图片与客户端 `view_image` 工具冲突，模型只回行动预告不识别图片 | 5 文件 +442-0 | 纯新增无删除。新建 `openai_gateway_grok_chat_image_tools.go`。本仓库只在 `openai_gateway_grok_cache.go:505` 有一句 view_image 注释 |
| `9ede0f716` + `5b2089c5a`（#5881 / #5868） | `tool_search_output` 未按标准 `function_call_output` 下发；搜索发现结果未提升为可调用工具 | 10 文件 | 需新建 `pkg/apicompat/responses_tool_search_discoveries.go`（263 行，本仓库无）。本仓库 `responses_client_tools.go:172` 已有 `case "tool_search_output"` 分支，是改不是加 |

### 5.3 OpenAI 容量过载仅以文本消息返回时未被识别

上游 `c3063e01a` + `539064798`（PR #5676），19 文件 +625-90。状态：**待合**

目标文件本仓库全有，含 `openai_capacity_shed_test.go`、`openai_compact_sse_keepalive.go`、`openai_account_runtime_block_fastpath.go`。纯 OpenAI 侧稳定性，无迁移、无接口变更。

### 5.4 Responses→Chat 桥接 encrypted-only reasoning 回注

上游 `612436a5a`（#5729）+ `401dd43b4`，17 文件 +665-12。状态：**待合，优先级低**

⚠️ **动了 `internal/repository/gateway_cache.go`（+42）和 `internal/service/gateway_service.go`（+14），并给 `internal/testutil/stubs.go` 加接口方法** → 触发 DEV_GUIDE 坑 6（改 interface 要改所有 stub/mock），上游自己就顺带改了 8 个测试文件各 +7 行。缓存走 Redis 接口，本 fork 的 miniredis 能跑。

上游举的例子是 DeepSeek thinking 持续 400（本 fork 没有 DeepSeek 平台）。对本仓库受益的是走 Responses→Chat 回落的 upstream 透传账号。**不用这条路径就先别合。**

### 5.5 内置价卡补充

状态：**待合**。只抄价卡，不抄倍率。

本仓库现状是别名链：

```
billing_service.go:265  claude-opus-4.8 = claude-opus-4.7 = claude-opus-4.6 = claude-opus-4.5
billing_service.go:266  claude-opus-5   = claude-opus-4.8
billing_service.go:303  gpt-5.5         = gpt-5.4
billing_service.go:304  gpt-5.5-pro     = gpt-5.4
```

上游 0.1.179 给 opus-4.8 / opus-5 补了 Fast 价，给 gpt-5.5 / gpt-5.5-pro 换成各自官方独立价。

⚠️ 这块价卡改动混在 `5b2a386ed`（渠道倍率特性）里，**不要整个 diff 抄过来**：那个提交同时给 `ModelPricing` 加了 `FastMultiplier *float64` / `FlexMultiplier *float64`，属于第 8 节判为「不合」的渠道倍率特性。只取 `fallbackPrices` 的数值，且先去官方价页核一遍。

---

## 6. P2 — 要重写方言，收益不值

### `a9514a68d` perf(usage) 单次扫描聚合 —— **不合**

上游用 `GROUPING SETS` 把四次扫描合成一次。**在本仓库的 SQLite 上直接语法报错**，2026-08-21 实测（把下面这段临时塞进 `internal/repository/` 跑 `go test`）：

```
sqlite_version=3.51.2
FAIL  select a, b, sum(n) from t group by grouping sets ((a),(a,b),())  -> near "sets": syntax error (1)
FAIL  select a, b, sum(n) from t group by rollup(a,b)                    -> no such function: rollup (1)
OK    select a, sum(case when b='p' then n else 0 end) from t group by a -> 2 rows
```

要移植必须改写成 `SUM(CASE WHEN …)` 条件聚合或 UNION ALL。而**本仓库已经把这四条查询用 `errgroup` 并行跑了**（`usage_log_repo_stats.go:785-790`，上游是串行），上游宣称的「20 秒级降到秒级」前提是千万级用量表，个人单机 SQLite 远到不了。

→ **只取第 4.4 节的筛选一致性，性能部分跳过。**

迁移 226（用量日志表达式索引，`CREATE INDEX CONCURRENTLY`）若要做：本仓库的 `*_notx.sql` 通道已经会在 SQLite 上丢掉 `CONCURRENTLY`（`migrations_runner.go:409`、校验规则见 :591-618），单独建成 `225_add_usage_log_effective_model_indexes_notx.sql` 即可，别跟上游的 GROUPING SETS 捆在一起。

---

## 7. 看需求决定

| 上游 commit | 内容 | 规模 | 判断 |
|---|---|---|---|
| `58e147fba`（#5816） | Composite 分组支持 Codex 端点（含 Alpha Search 与 Live） | 12 文件 +216-28 | **用 composite 组跑 Codex CLI 就值得。** 本仓库有 composite，`routes/gateway.go:361` 的 `codexDirect` 已挂 `compositeTarget`。上游改的是 ① `modelsHandler` 带 `client_version` 时 Composite 也走 `CodexModels`；② `compositeJSONRequestModel` 额外认 `session.model` 路径（Live/realtime 用）；③ multipart 认 `session` 字段。**patch 里 `countTokensHandler` 那一跳带 Kimi/Zhipu/Deepseek 常量，要剔掉** |
| `b0464a986` + `ec5a34593` + `1ab325678` + `d5484866f`（#5834） | 代理连通性探测目标可配置（有序列表 + 按目标解析） | 4 文件 +100-5 | 小而实用。本仓库 `internal/repository/proxy_probe_service.go:49` 还是硬编码 `probeURLs`。国内网络下能换探测目标。要同步 `deploy/config.example.yaml`，并考虑要不要进 `deploy/config.personal.sqlite.yaml` |
| `bfac49fef`（#5810） | `POST /v1/responses/input_tokens` 预检 | 10 文件 +479-17 | 纯新端点，Codex CLI 会调。本仓库 `handler/` 与 `service/` 两侧的 `openai_gateway_count_tokens.go` 都在。风险低但非必需 |
| `1f2a87adb`（#5875） | 补全管理端平台筛选 + 抽共享 `frontend/src/constants/platforms.ts` | 11 文件 +148-62 | **思路借鉴，不整包合。** 上游目录含三个本 fork 没有的平台；本 fork 另有 hidden menu / simple mode 过滤。可以自己抽一个只含 6 平台的目录 |
| `63839f193`（#5838） | 管理端用户角色选择器样式 | 前端小改 | 可选。本仓库有 `frontend/src/views/admin/UsersView.vue` |

---

## 8. 不合 / N/A

| 上游条目 | commit | 原因 |
|---|---|---|
| 账号自适应 API 协议（Kimi/GLM/DeepSeek adaptive） | `85051616f`、`b3092145d`（#5842） | 无 CN 平台，无 `api_protocol` 字段。见第 2 节 |
| 国产供应商账号请求头覆写 | `1b30a2d74`（#5847） | 同上 |
| 国产供应商连接测试按 `api_protocol` 路由 | `ac6208de1`（#5782） | 同上 |
| Composite 支持 Kimi/GLM/DeepSeek + 迁移 227 | `b171bb0e4`、`aa673062e`、`499a8ee42`、`4d3b300a2`（#5817） | 同上。迁移 227 一并跳过 |
| 渠道监控配额语义五连 | `1128df259`、`c41ae19e5`、`e2dfb3b8c`、`c9effc456`、`2c250bfd7`（#5780） | 无 0.1.178 的配额模式底座，见第 2 节 |
| 监控 quota 占位模型本地化、CN 配额标签重叠 | `2c250bfd7`、`994fbfedd` | 同上 |
| 渠道定价服务层级倍率 + 上下文区间倍率 + 迁移 228 | `fce90ecf8`、`5b2a386ed`、`26be82cc8`、`d4d2c746c`、`d536795e9`（#5851） | 个人单机用不上「按渠道配倍率」。且要新建 SQLite 迁移、动 `channel_repo_pricing.go`（本 fork 手写 SQLite SQL，不能整文件覆盖）。**若日后要做第 9 节的 Anthropic Fast，须带上 `5b2a386ed` 里把 `usePriorityServiceTierPricing` 扩到 `tier == "fast"` 那一小块** |
| 原生 Anthropic 识别 Fast mode（`speed: "fast"`）并计费 | `7dae055f2` | 本仓库 `ParsedRequest` **没有 `Speed` 字段**、`ForwardResult` **没有 `ServiceTier`**（grep 零命中）；只在 OpenAI-compat 路径认 `anthropic-beta: fast-mode-2026-02-01` → priority（`openai_gateway_messages.go:124`）。要合得先补三处基础字段 + 用量落库 + 价卡 fast 价。**只有真在用原生 Anthropic 账号且开 Fast 才划算**，否则只是在「没跑 fast 却多收钱」的方向上加代码 |
| 管理端用量 GROUPING SETS 聚合 + 迁移 226 | `a9514a68d` | SQLite 不支持，见第 6 节（筛选一致性那一半已拆到 4.4） |
| `85cb732cd` README star history | — | 上游 README 专属 |

---

## 9. 决策点：长上下文计费门控 AND → OR（破坏性变更）

上游 0.1.179 把「分组开关 AND 账号开关」改成「任一启用即生效」。由于分组开关默认开、账号开关默认关，**存量部署里 OpenAI 超过 272k 上下文的请求会开始按 2× 输入 / 1.5× 输出计费**。上游给的退路是把相关分组的 `long_context_pricing_enabled` 置 false。

本仓库现状（`internal/service/billing_service.go:1102-1105`）就是 AND：

```go
applyLongCtx := len(resolved.Intervals) == 0 && resolved.longContextPricingEnabled
if input.LongContextBillingEnabled != nil {
    applyLongCtx = applyLongCtx && *input.LongContextBillingEnabled   // ← 改 OR 就是这一行
}
```

账号侧的门由 `openai_gateway_usage.go:490 openAILongContextBillingGate` 提供（非 OpenAI 账号返回 nil，保留 Grok ≥200k 官方 2×，见 0.1.176 清单 P0）；分组侧由 `model_pricing_resolver.go:72` 提供。

取舍：

- **对外分发** → 跟 OR，账单口径和 OpenAI 实际扣费对齐，否则长上下文请求一直少收；
- **纯自用记账** → 保持 AND 也行，代价是用量统计低估真实成本。

状态：**未决**。改之前先确认这台是不是对外发 key。

---

## 10. 不要动

| 项 | 原因 |
|---|---|
| `git merge upstream/main` / 整 PR cherry-pick | 历史无共同祖先 |
| 上游迁移 226 / 227 / 228 原文件 | 226 是 PG `CONCURRENTLY` + GROUPING SETS 配套；227 是 CN 平台；228 是渠道倍率。且本地 224 之后要顺延编号 |
| `5b2a386ed` 里的 `ModelPricing.FastMultiplier` / `FlexMultiplier` | 渠道倍率特性的一部分，别跟着价卡一起抄进来 |
| 上游 `frontend/src/constants/platforms.ts` 原文件 | 含三个本 fork 没有的平台 |
| 改已应用的 `migrations/*.sql` | checksum 不可变 |
| `wire_gen.go` 手改 | 动了 wire 就 `go generate ./cmd/server` |
| 在 `skipSQLiteBackgroundJobs` 里新增服务来「绕过」上游 SQL | README 硬约束第 9 条 |

---

## 11. v0.1.179 之后的 `main`（32 commits，尚未发版）

`v0.1.179..main` ahead 32 / 57 files，几乎全是 Grok：默认模型迁移到 4.6、官方计费目录校正、Realtime 预握手复用与切号、限流冷却、媒体超时重试、错误分类与容量重试。

本仓库 `DefaultTextModel` 目前仍是 `grok-4.5`（0.1.176 清单里明确要求保持）。**等上游打成 tag 再整体评估这一串**，不要现在零散摘——默认模型迁移和计费目录是一套。

---

## 12. 核查命令

整仓 fetch 在这边容易超时，用 API 逐条看：

```bash
# 发布说明
curl -sS -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/Wei-Shaw/sub2api/releases/tags/v0.1.179 | python3 -c 'import json,sys;print(json.load(sys.stdin)["body"])'

# 两个 tag 之间的提交
curl -sS "https://api.github.com/repos/Wei-Shaw/sub2api/compare/v0.1.178...v0.1.179"

# 单个提交的文件与 patch
curl -sS "https://api.github.com/repos/Wei-Shaw/sub2api/commits/b228b93e9"

# 单文件原文
curl -fsS "https://raw.githubusercontent.com/Wei-Shaw/sub2api/v0.1.179/backend/internal/service/openai_gateway_grok.go"
```

未认证 API 每小时 60 次，逐条拉 patch 时留意配额。`github.com/<owner>/<repo>/commit/<sha>.patch` 在这边不稳定，优先走 API。

验证某条 SQL 在本仓库的 SQLite 上到底行不行（第 6 节就是这么测的）：把用例写成 `package repository` 的临时 `_test.go` 丢进 `backend/internal/repository/`，`go test ./internal/repository/ -run … -v`，跑完删掉。别用系统 `sqlite3` CLI——版本和 `modernc.org/sqlite` 内置的 3.51.2 不一定一致。

自测重点见 [README.md 第 3.4 节](./README.md#34-自测再合回-main)：登录 / `user_allowed_groups`、`usage_billing_dedup` 写入、调度冷却与账号 failover。
