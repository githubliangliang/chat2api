# 移植清单：上游 v0.1.176

对照日期：2026-08-15。  
合完一项就把状态改成「已合」。P0 已合，P1–P3 未动。

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

状态：未合

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/internal/pkg/xai/subscription_tier.go` | **新建** | 上游 279 行：`MapJWTSubscriptionTier`、`SubscriptionTierFromJWT`、`CanonicalGrokPlan`、4.5 window 推断 Heavy |
| `backend/internal/pkg/xai/subscription_tier_test.go` | **新建** | 上游测试原样 |
| `backend/internal/pkg/xai/quota.go` | 改 | 快照字段 `PlanFrom45Responses` / `PlanFrom45ResponsesAt` |
| `backend/internal/service/grok_oauth_service.go` | 改 | `applyGrokTokenClaims` 写入 JWT `tier`；刷新后覆盖失效订阅 |
| `backend/internal/service/grok_oauth_service_test.go` | 改 | 上游约 +80 |
| `backend/internal/service/grok_quota_fetcher.go` | 改 | 刷新用 JWT + 4.5 window 覆盖档位 |
| `backend/internal/service/grok_quota_fetcher_test.go` | 改 | 上游约 +105 |
| `backend/internal/service/account_test_service.go` | 小改 | 测连通性时带档位 |
| `frontend/src/components/common/PlatformTypeBadge.vue` | 可选 | SuperGrok / Heavy 颜色 |
| `frontend/src/components/account/AccountUsageCell.vue` | **慎** | 按实时档位展示，可能和简化过的用量格冲突。P1 可只做后端 |
| `frontend/src/utils/accountUsageRefresh.ts` | **慎** | 增量刷新比 Grok 快照；后置 |
| `frontend/src/views/admin/AccountsView.vue` | **先别动** | +100 行徽章/自动刷新，和本 fork 简化冲突 |

自测：JWT `tier=0/1/5/6` → free / supergrok / heavy / lite；刷新后旧档位被覆盖。

---

## 6. P2 — 分组逐模型定价

解析链改成 **Group → Channel → 内置**。本仓库还是 Channel → 内置。

状态：未合

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/migrations/223_group_model_pricing.sql` | **新建** | 不要用上游文件名 `221_*`。SQLite 方言 |
| `backend/ent/schema/group.go` | 改 | 在 audio 价字段后加：`long_context_pricing_enabled` 默认 **true**；`model_pricing` JSON optional |
| `backend/ent/*` | 生成 | `go generate ./ent`，不要抄上游 ent |
| `backend/internal/service/group.go` | 改 | struct 加两个字段 |
| `backend/internal/repository/group_repo.go` | **手改 SQL** | SELECT/INSERT/UPDATE 加列 |
| `backend/internal/handler/admin/group_handler.go` | 改 | 创建/更新/响应带上 |
| `backend/internal/handler/dto/types.go` + `mappers.go` | 改 | API 字段 |
| `backend/internal/service/admin_group.go` | 改 | 读写 + 校验 |
| `backend/internal/service/model_pricing_resolver.go` | 改 | 先 group，再 channel。关长上下文时 token 模型只取最低档 |
| `backend/internal/service/model_pricing_resolver_test.go` | 改 | 上游约 +63 |
| `backend/internal/service/billing_service.go` | 再改 | 分组开关；**非 OpenAI 不要被 OpenAI 账号开关否决**（#5573） |
| `backend/internal/repository/api_key_repo.go` | 改 | available groups 查出新字段 |
| `backend/internal/server/api_contract_test.go` | 改 | 契约字段 |
| `frontend/src/types/index.ts` | 改 | Group 类型 |
| `frontend/src/views/admin/GroupsView.vue` | 改 | 逐模型定价表 + 长上下文开关（上游 +122） |
| `frontend/src/i18n/locales/{zh,en}/admin/channels.ts` | 改 | 文案 |
| `frontend/src/components/admin/channel/PricingEntryCard.vue` | 看 | 可能只是复用卡片 |

SQLite 迁移草稿（新建 `223`，不要改已有文件）：

```sql
ALTER TABLE groups ADD COLUMN long_context_pricing_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE groups ADD COLUMN model_pricing TEXT;
UPDATE groups SET long_context_pricing_enabled = 1
 WHERE long_context_pricing_enabled IS NULL OR long_context_pricing_enabled = 0;
```

默认必须是开。上游修过一次默认 `false` 导致存量分组丢 ≥200k 阶梯。

禁止照抄上游 PG：

```sql
-- 不要用
ADD COLUMN IF NOT EXISTS ... JSONB
COMMENT ON COLUMN ...
IS DISTINCT FROM
```

---

## 7. P3 — 原生 `POST /x_search`

本仓库已有 tool 形态的 `x_search` 计数/注入，没有独立端点。

状态：未合

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/internal/handler/openai_x_search.go` | **新建** | `XSearch` 只 set `grok_x_search_endpoint` 再调 `WebSearch` |
| `backend/internal/handler/openai_x_search_test.go` | **新建** | |
| `backend/internal/handler/gateway_web_search.go` | 改 | 读该 flag，走 `x_search` tool + `include: x_search_call.action.sources`；接受 `allowed_x_handles` 等 |
| `backend/internal/server/routes/gateway.go` | 改 | 分组路由 + 根路由各加 `POST /x_search`，仅 Grok（和现有 `/web_search` 并列） |
| `backend/internal/pkg/apicompat/types.go` | 改 | 保留 x_search 过滤字段 |
| `backend/internal/pkg/apicompat/chatcompletions_to_responses.go` | 改 | Chat↔Responses 往返 |
| `backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go` | 改 | sources 抽取 |
| `backend/internal/pkg/apicompat/chatcompletions_x_search_test.go` | **新建** | |
| `backend/internal/service/openai_gateway_grok.go` / `_cache.go` / `_chat_bridge.go` | 小改 | 保留 tool_choice / sources |
| `backend/internal/handler/openai_gateway_handler.go` | 看 | 已有 composite grok `/v1/messages` 修复，合路由时别回退 |

---

## 8. 顺手可合的小 bugfix

| PR | 文件 | 说明 | 状态 |
|----|------|------|------|
| [#5540](https://github.com/Wei-Shaw/sub2api/pull/5540) | `channel_service.go` + test | 定价冲突检测和 cache key 归一化对齐。本地已有 `normalizeChannelPricingModelName`，先 `git show` 再决定 | 未核 |
| [#5543](https://github.com/Wei-Shaw/sub2api/pull/5543) | `admin_group.go`、`admin_service.go`、`channel_service.go`、`wire.go` | 分组改平台后失效渠道缓存。本地 `admin_group` 只见 auth cache，渠道定价缓存大概率还没失效 | 未合 |
| [#5504](https://github.com/Wei-Shaw/sub2api/pull/5504) | `openai_apikey_responses_probe.go` + 新 test | 探测未跑完不要标「不支持 Responses」。本地 probe 还很瘦 | 未合 |

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
