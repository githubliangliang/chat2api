# 移植清单：上游 v0.1.176

对照日期：2026-08-15。  
合完一项就把状态改成「已合」。P0–P3 与第 8 节小 bugfix 全部已合。

通用流程见 [README.md](./README.md)。

---

## 1. 版本对照

| 点 | 状态 |
|----|------|
| 改造时所称基线 | 上游 [`v0.1.173`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.173)（2026-08-09，`29009f0`） |
| 中间跳过 | 没有 `v0.1.174` |
| 上游最新正式版 | [`v0.1.176`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.176)（2026-08-13，Latest） |
| `main` 比 `v0.1.173` | 124 commits / 244 files |
| `v0.1.176` 比 `v0.1.173` | 107 commits |
| `main` 比 `v0.1.176` | 再超 17 commits（截至 2026-08-15） |
| 本仓库版本号 | `backend/cmd/server/VERSION` = `1.1.0`（自有编号，不要改成 0.1.176） |

对比链接：

- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.173...v0.1.176>
- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.175...v0.1.176>
- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.176...main>
- 大功能 PR：[#5571](https://github.com/Wei-Shaw/sub2api/pull/5571)（86 文件）、[#5573](https://github.com/Wei-Shaw/sub2api/pull/5573)

### 本仓库不是纯 v0.1.173

最早一条提交是 2026-08-12 22:57 的整仓 squash（`feat: SQLite 个人部署支持与用量落库修复`），时间已经晚于 [`v0.1.175`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.175)。

**不要从 v0.1.173 整段 cherry-pick 到现在。**

---

## 2. v0.1.175：大部分已有

发布说明：<https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.175>

| 上游改动 | 本仓库 | 建议 |
|----------|--------|------|
| 按上游响应模型计费 | 已有 `billing_model_source` / `upstream_response_model` | 跳过 |
| HTML 403 误下线 | 已有 `openai_403_html_body_skips_account_penalty` | 跳过 |
| 空 `response.completed` 流 failover | 已有 `openai_gateway_responses_empty_completed_test.go` | 跳过 |
| 原生 Responses 400→502、TTFT、OAuth 图片 failover | 大概率已有 | 不要整包合 |
| remote compaction v2、`compaction_trigger` | 已有 | 跳过整包；`main` 后续若有补丁再对 |
| `x-codex-turn-state` 透传 | 已有 | 跳过整包；看 `main` 的「跨账号 echo」 |
| Composite 分组图片开关、API Key 校验 | 需逐项看 | 低优先级 |
| **Codex OAuth 设备指纹收敛**（`off/device/session/full`） | **没有**（搜不到 `codex_fingerprint_mode`） | 值得合，但用 `main` 的 **opt-in / 默认 off**，不要合 175 的默认 session |
| 大文件备份分卷 | 这边是 SQLite `VACUUM INTO` | 跳过 |
| simple mode 显示安全审计菜单 | 和 hidden menu / simple mode 会撞 | 慎合 |
| 风控 fail-closed | 上游自己 revert 了 | **不要合** |

---

## 3. 建议顺序

```text
① P0  Grok 4.6 目录 + 价卡 + reasoning 白名单     无迁移，风险最低
② P1  JWT 订阅档位                                 无迁移
③     #5543 渠道缓存失效 + #5504 probe              小
④ P2  分组逐模型定价（新 223 迁移 + generate）      最大
⑤ P3  POST /x_search                               独立
```

上游 `DefaultTextModel` 仍是 `grok-4.5`。4.6 是目录 / 别名 / 价卡，**不要**把默认模型改成 4.6。

本仓库迁移号已用到 `222`。上游 `221_group_model_pricing.sql` 在这边编号为 **`223`**（本地 `221` 已是 affiliate）。

`#5571` 带了生成的 `backend/ent/*`。这边只改 `backend/ent/schema/group.go`，然后：

```bash
cd backend && go generate ./ent
```

`group_repo.go` 的 SQL 是本 fork 手写 SQLite，不能整文件覆盖。

---

## 4. P0 — Grok 4.6 能跑、有价

目标：请求 `grok-4.6` 不 404，有官方价，未登记文本模型不记 0。

状态：已合（2026-08-15）

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/internal/pkg/xai/models.go` | 改 | `defaultModels` 加 `{ID:"grok-4.6", ...}`；`grokTextResponsesModelAliases` 加 `grok-4.6` / `grok-4.6-latest` → `"grok-4.6"`。`DefaultTextModel` **保持 `grok-4.5`** |
| `backend/internal/pkg/xai/models_test.go` | 改 | 补 4.6 别名断言（上游约 +9 行） |
| `backend/internal/service/billing_service.go` | 改 | `fallbackPrices["grok-4.6"]`：input `$2/M`、cached `$0.50/M`、output `$6/M`，**200k 长上下文倍率**。`getFallbackPricing` 增加 `grok-4.6` / `grok-4.6-latest`。未登记 Grok 文本回退 `grok-4.5` 价卡，**排除 image/video/audio**（#5573） |
| `backend/internal/service/billing_service_test.go` | 改 | 跟价卡和媒体排除 |
| `backend/internal/service/openai_gateway_usage.go` | 改 | `openAILongContextBillingGate`：非 OpenAI 账号返回 nil，Grok ≥200k 官方 2× 不被账号开关否决（#5573） |
| `backend/internal/service/openai_gateway_grok.go` | 改 | `grokSupportsReasoningEffort` 加上 `grok-4.6` / `grok-4.6-latest` |
| `backend/internal/handler/gateway_handler.go` | 改 | `grokModelSupportsConfigurableReasoning` 同样加 4.6 |
| `backend/internal/handler/grok_audio.go` | 跳过 | 仍硬编码 `grok-4.5`（上游默认也是 4.5） |
| `backend/internal/service/grok_model_quota_block.go` | 跳过 | 已按模型名通用封禁，不必为 4.6 单开 |
| `frontend/src/composables/useModelWhitelist.ts` | 改 | 白名单 + 预设加 4.6 |
| `frontend/src/constants/channel.ts` | 跳过 | 本仓库没有 Grok 模型清单；分组定价是 P2 |

自测：

```bash
cd backend
go test -tags=unit ./internal/pkg/xai/ ./internal/service/ -count=1 \
  -run 'Grok|Billing|Usage|QuotaBlock|4\.6'
```

手工打一条 `grok-4.6`，确认 usage 不是 0。

---

## 5. P1 — JWT 订阅档位

本仓库 `applyGrokTokenClaims` 只解 `email` / `sub` / `team_id`，不解 `tier`。

状态：已合（2026-08-15，仅后端）

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/internal/pkg/xai/subscription_tier.go` | 新建 | `MapJWTSubscriptionTier`、`SubscriptionTierFromJWT`、`CanonicalGrokPlan`、4.5 window 推断 Heavy |
| `backend/internal/pkg/xai/subscription_tier_test.go` | 新建 | 上游测试原样 |
| `backend/internal/pkg/xai/quota.go` | 改 | 快照字段 `Model` / `PlanFrom45Responses` / `PlanFrom45ResponsesAt` |
| `backend/internal/service/grok_oauth_service.go` | 改 | access token 解 JWT `tier`；刷新后覆盖失效订阅；ID token 不解档位 |
| `backend/internal/service/grok_oauth_service_test.go` | 改 | SSO 抽出档位 + 刷新覆盖 / 保留 |
| `backend/internal/service/grok_quota_fetcher.go` | 改 | 实时 JWT 优先；否则 CanonicalGrokPlan（含 4.5 window） |
| `backend/internal/service/grok_quota_fetcher_test.go` | 改 | JWT / SuperGrokPro / 4.5 Heavy 窗口 |
| `backend/internal/service/openai_gateway_grok.go` | 改 | 写入快照时 stamp 4.5 窗口信号 |
| `backend/internal/service/grok_quota_service.go` | 改 | 主动探测 stamp 探测模型 |
| `backend/internal/service/account_test_service.go` | 小改 | Responses / web_search 探测带模型进快照 |
| `frontend/...` | 跳过 | 徽章/用量格和本 fork 简化 UI 冲突，P1 只做后端 |

自测：JWT `tier=0/1/5/6` → free / supergrok / heavy / lite；刷新后旧档位被覆盖。

---

## 6. P2 — 分组逐模型定价

解析链改成 **Group → Channel → 内置**。本仓库还是 Channel → 内置。

状态：已合（2026-08-15）

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/migrations/223_group_model_pricing.sql` | 新建 | SQLite 方言；`long_context_pricing_enabled INTEGER NOT NULL DEFAULT 1`、`model_pricing TEXT` |
| `backend/ent/schema/group.go` | 改 | audio 价字段后加两列；`model_pricing` 用 `json.RawMessage` |
| `backend/ent/*` | 生成 | `GOPROXY=https://goproxy.cn,direct go generate ./ent` |
| `backend/internal/service/group.go` | 改 | struct 加 `LongContextPricingEnabled` / `ModelPricing` |
| `backend/internal/service/channel.go` | 改 | `ChannelModelPricing` / `PricingInterval` 补 JSON tag（分组定价按 JSON 落库） |
| `backend/internal/repository/group_repo.go` | 改 | 创建/更新 marshal `model_pricing` |
| `backend/internal/repository/api_key_repo.go` | 改 | `groupEntityToService` unmarshal，失败降级为 nil + warn |
| `backend/internal/handler/admin/group_handler.go` | 改 | 创建用值类型、更新用指针；透传两字段 |
| `backend/internal/handler/dto/types.go` + `mappers.go` | 改 | `Group.long_context_pricing_enabled`；`model_pricing` 仅 AdminGroup |
| `backend/internal/service/admin_service.go` / `admin_group.go` | 改 | 输入字段 + `normalizeGroupModelPricing`（复用 `validatePricingEntries`） |
| `backend/internal/service/model_pricing_resolver.go` | 改 | Group → Channel → 内置；关长上下文时 token 模型只取最低档 |
| `backend/internal/service/billing_service.go` | 改 | `CostInput.Group`；`applyLongCtx` 加 `resolved.longContextPricingEnabled` |
| `backend/internal/service/gateway_usage_billing.go` / `openai_gateway_usage.go` | 改 | resolve 带 Group，接受 `PricingSourceGroup` |
| `backend/internal/service/model_pricing_resolver_test.go` | 改 | 分组优先级、长上下文开关、渠道阶梯塌陷、nil group |
| `backend/internal/server/api_contract_test.go` | 改 | 契约补 `long_context_pricing_enabled` |
| `frontend/src/types/index.ts` | 改 | Group / AdminGroup / 创建更新请求 |
| `frontend/src/views/admin/GroupsView.vue` | 改 | 复用 `PricingEntryCard` 的逐模型定价表 + 长上下文开关 |
| `frontend/src/i18n/locales/{zh,en}/admin/overview.ts` | 改 | `admin.groups.modelPricing.*` 文案 |

自测：

```bash
cd backend
go test -tags=unit ./internal/service/ -count=1 -run 'TestResolve_|TestApplyTokenOverrides|TestGetIntervalPricing'
go test -tags=unit ./internal/server/ -count=1 -run 'TestAPIContracts/GET_/api/v1/groups/available'
cd ../frontend && pnpm run typecheck && pnpm run lint:check
```

已知预存在失败（HEAD 上同样红，与本次无关）：`migrations` 包全部（断言 PG 语法）、
`TestAPIContracts/GET_/api/v1/admin/settings*`、OpenAI OAuth originator 系列。

---

## 7. P3 — 原生 `POST /x_search`

本仓库已有 tool 形态的 `x_search` 计数/注入，没有独立端点。

状态：已合（2026-08-15）

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/internal/handler/openai_x_search.go` | 新建 | `XSearch` 只 set `grok_x_search_endpoint` 再调 `WebSearch`；请求体 + `buildGrokXSearchResponsesBody` |
| `backend/internal/handler/openai_x_search_test.go` | 新建 | 工具字段、`input` 别名、运行时默认模型 |
| `backend/internal/handler/gateway_web_search.go` | 改 | 读该 flag，走 `x_search` tool + `include: x_search_call.action.sources`；模型改用运行时默认；`x_search_call` 也抽 sources |
| `backend/internal/server/routes/gateway.go` | 改 | 分组路由 + 根路由各加 `POST /x_search`，仅 Grok |
| `backend/internal/server/routes/prompt_audit_route_coverage_test.go` | 改 | `/x_search` 归到 `gateway_web_search.go`（审计在那里做） |
| `backend/internal/pkg/apicompat/types.go` | 改 | `ResponsesTool` / `ChatTool` 加 x_search 过滤字段 |
| `backend/internal/pkg/apicompat/chatcompletions_to_responses.go` | 改 | Chat → Responses 保留 x_search |
| `backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go` | 改 | Responses → Chat 保留 x_search + tool_choice |
| `backend/internal/pkg/apicompat/chatcompletions_x_search_test.go` | 新建 | 往返 + tool_choice 字符串形式 |
| `backend/internal/service/openai_gateway_grok*.go` | 跳过 | 核对 #5571 后确认这三个文件没有 x_search 改动（文档原先的猜测不成立） |

---

## 8. 顺手可合的小 bugfix

| PR | 文件 | 说明 | 状态 |
|----|------|------|------|
| [#5540](https://github.com/Wei-Shaw/sub2api/pull/5540) | `channel_service.go` + test | 定价冲突检测改用 `normalizeChannelPricingModelName`，和缓存键同一套归一化；映射侧保持 ToLower | 已合 |
| [#5543](https://github.com/Wei-Shaw/sub2api/pull/5543) | `admin_group.go`、`admin_service.go`、`channel_service.go`、`wire.go` + test | 分组改平台后失效渠道缓存；`ChannelCacheInvalidator` 窄接口，`wire_gen.go` 用 generate | 已合 |
| [#5504](https://github.com/Wei-Shaw/sub2api/pull/5504) | `openai_apikey_responses_probe.go` + 新 test | 探测判据不成立（`incomplete/max_output_tokens`、`failed`）时保持 unknown；落标不支持时 warn | 已合 |

---

## 9. v0.1.176 之后的 main（尚未发版）

<https://github.com/Wei-Shaw/sub2api/compare/v0.1.176...main>（17 commits）

| 主题 | 建议 |
|------|------|
| Grok 长上下文只跟分组开关，不被 OpenAI 账号开关否决 | 合 P2 时一起带（#5573 已含） |
| 未知 Grok 文本兜底排除 image/video/audio | 合 P0 时一起带（#5573 已含） |
| Codex fingerprint 改成 **opt-in，默认 off**，并覆盖透传 | 合指纹时以这个为准 |
| 保留 remote compaction v2、native/legacy 分流 | 已有类似实现，先 `git show` |
| relay `x-codex-turn-state`、防跨账号 echo | 已有透传，重点看「跨账号 echo」 |
| 分组用量日聚合 + 时区 | 刚藏过用量筛选/图表，别把大统计 UI 合回来 |
| CI 里 Go **1.26.6** | 这边 CI 写死 `1.26.5`，升则 `go.mod` + workflow 一起改 |

---

## 10. 不要动

| 项 | 原因 |
|----|------|
| 整 PR cherry-pick / `merge upstream/main` | 历史无共同祖先；ent 生成物、PG SQL 会炸 |
| 上游 `backend/ent/group*.go`、`mutation.go` | 用 `go generate` |
| 上游 `221_group_model_pricing.sql` 原文件 | PG；本地 221 已被占用 |
| `AccountsView.vue` 大改、用量格刷新 | 和本 fork 简化过的用量 UI 冲突 |
| 备份 leader 锁、分卷上传 | 单机 SQLite 用不上 |
| Codex fingerprint `session` 默认 | `main` 已改 opt-in / off |
| 改已应用的 `migrations/*.sql` | checksum 不可变 |
| `wire_gen.go` 手改 | 动了 wire 就 `go generate ./cmd/server` |

---

## 11. 拉上游文件的办法

整仓 `git fetch` / `git clone` 在这边经常超时。对照时可以：

```bash
# 只看 PR 改了哪些文件
curl -sS "https://api.github.com/repos/Wei-Shaw/sub2api/pulls/5571/files?per_page=100"

# 单文件
curl -fsS "https://raw.githubusercontent.com/Wei-Shaw/sub2api/v0.1.176/backend/internal/pkg/xai/models.go"
```

临时对照目录（不进本仓库）：`/tmp/u176/`。
