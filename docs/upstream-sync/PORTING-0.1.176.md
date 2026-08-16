# 移植清单：上游 v0.1.176

对照日期：2026-08-15（当日复核：上游已发 [`v0.1.177`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.177)，见第 9 节）。2026-08-16 逐项复核：全部「已合」项经代码与测试核实属实。  
合完一项就把状态改成「已合」。P0–P3 与第 8 节小 bugfix 全部已合。

通用流程见 [README.md](./README.md)。

---

## 1. 版本对照

| 点 | 状态 |
|----|------|
| 改造时所称基线 | 上游 [`v0.1.173`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.173)（2026-08-09，`29009f0`） |
| 中间跳过 | 没有 `v0.1.174` |
| 上游最新正式版 | [`v0.1.177`](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.177)（2026-08-15 13:40 UTC，Latest）；`v0.1.176` 已不是最新 |
| `main` 比 `v0.1.173` | 124 commits / 244 files |
| `v0.1.176` 比 `v0.1.173` | 107 commits |
| `v0.1.177` 比 `v0.1.176` | 18 commits（即原第 9 节所列 main 提交 + VERSION bump） |
| `main` 比 `v0.1.177` | 1 commit（2026-08-16 实测 ahead_by=1：`baeac1f3` 仅 sync VERSION，打 tag 后 16 秒才落 main、不在 tag 内；无代码可合） |
| 本仓库版本号 | `backend/cmd/server/VERSION` = `1.1.0`（自有编号，不要改成 0.1.176） |

对比链接：

- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.173...v0.1.176>
- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.175...v0.1.176>
- <https://github.com/Wei-Shaw/sub2api/compare/v0.1.176...v0.1.177>
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
| **Codex OAuth 设备指纹收敛**（`off/device/session/full`） | **已合**（2026-08-15，按 `main` 的 opt-in / 默认 off） | 已合：`openai_codex_fingerprint.go` + Forward / 透传两条路径 + 三个账号弹窗 |
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
2026-08-16 全量复核补充（在 `216fc41` 上验证同样失败）：
`TestOpenAIGatewayService_OAuthPassthrough_{CodexTuiIdentityUnified,OfficialIdentityUnified,StreamKeepsToolNameAndBodyNormalized}`。
`internal/repository` 此前整包编译失败（`usage_log_session_id_unit_test.go` 引用已被 `25646e6` 删除的
`usageLogInsertArgTypes`），2026-08-16 已修复，并连带清完被它掩盖的 9 个旧测试失败——逐个甄别均为
「测试仍锚定 PG 行为、生产已按设计 SQLite 化」，无真实回归：proxy 身份 SELECT 去掉 `FOR NO KEY UPDATE`、
探测失效改 `json_remove`/`json_type`（事务与回滚语义仍由测试验证）；monitor 模板 headers 合并改
`json_patch`/`json_set`（行数守卫与回滚消息不变）；DB 池对 SQLite 单写者恒 clamp（≤0 或 >8 → 4，idle 封顶到 open）；
OAuth 候选页 `LIMIT ?`；审核计数 `NOT $3`。生产侧仅动 `error_translate.go`：typed-nil 守卫 +
`SQLState()=="23505"` 接口判定（不引 lib/pq，过 sqlite 方言审计）。该包现全绿。

---

## 7. P3 — 原生 `POST /x_search`

本仓库已有 tool 形态的 `x_search` 计数/注入，没有独立端点。

状态：已合（2026-08-15）

| 文件 | 动作 | 改什么 |
|------|------|--------|
| `backend/internal/handler/openai_x_search.go` | 新建 | `XSearch` 只 set `grok_x_search_endpoint` 再调 `WebSearch`；请求体 + `buildGrokXSearchResponsesBody`（`x_search` tool 与 `include: x_search_call.action.sources` 字面量在此 builder 里） |
| `backend/internal/handler/openai_x_search_test.go` | 新建 | 工具字段、`input` 别名、运行时默认模型 |
| `backend/internal/handler/gateway_web_search.go` | 改 | 读该 flag 分流 `doGrokNativeXSearch`（调用上行 builder）；模型改用运行时默认；`x_search_call` 也抽 sources；用量模型标签改为动态 `grok-`+label（/x_search 落 `grok-x-search`；搜索按分组 search_price_per_1k 计费，与标签解耦） |
| `backend/internal/server/routes/gateway.go` | 改 | 分组路由 + 根路由各加 `POST /x_search`，仅 Grok |
| `backend/internal/server/routes/prompt_audit_route_coverage_test.go` | 改 | `/x_search` 归到 `gateway_web_search.go`（审计在那里做） |
| `backend/internal/pkg/apicompat/types.go` | 改 | `ResponsesTool` / `ChatTool` 加 x_search 过滤字段 |
| `backend/internal/pkg/apicompat/chatcompletions_to_responses.go` | 改 | Chat → Responses 保留 x_search |
| `backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go` | 改 | Responses → Chat 保留 x_search + tool_choice |
| `backend/internal/pkg/apicompat/chatcompletions_x_search_test.go` | 新建 | Chat→Responses 与 Responses→Chat 两个单向用例 + tool_choice 字符串形式 |
| `backend/internal/handler/openai_gateway_handler.go` | 核过 | 本次未改；composite grok `/v1/messages` 修复无回退（仍走 `isOpenAIResponsesCompatibleGatewayPlatform` 分支） |
| `backend/internal/service/openai_gateway_grok*.go` | 跳过 | 核对 #5571 后确认这三个文件没有 x_search 改动（文档原先的猜测不成立） |

补记（2026-08-16 核查）：`/web_search` 与 `/x_search` 共用 `grokStandaloneSearchRequest` 绑定，普通 `/web_search` 会解析并静默忽略 `allowed_x_handles` / `from_date` 等 x_search 专有字段，无害。

---

## 8. 顺手可合的小 bugfix

| PR | 文件 | 说明 | 状态 |
|----|------|------|------|
| [#5540](https://github.com/Wei-Shaw/sub2api/pull/5540) | `channel_service.go` + test | 定价冲突检测改用 `normalizeChannelPricingModelName`，和缓存键同一套归一化；映射侧保持 ToLower | 已合 |
| [#5543](https://github.com/Wei-Shaw/sub2api/pull/5543) | `admin_group.go`、`admin_service.go`、`channel_service.go`、`wire.go`、`api_contract_test.go`（1 行） + test | 分组改平台后失效渠道缓存；`ChannelCacheInvalidator` 窄接口，`wire_gen.go` 用 generate | 已合 |
| [#5504](https://github.com/Wei-Shaw/sub2api/pull/5504) | `openai_apikey_responses_probe.go` + 新 test | 探测判据不成立（`incomplete/max_output_tokens`、`failed`）时保持 unknown；落标不支持时 warn | 已合 |

---

## 9. v0.1.177（2026-08-15 已发版，原「main 尚未发版」一节）

<https://github.com/Wei-Shaw/sub2api/compare/v0.1.176...v0.1.177>（18 commits；`main` 截至 2026-08-15 与 tag 一致，没有更新的提交）

| 主题 | 上游提交 / PR | 状态与建议 |
|------|---------------|------------|
| Grok 长上下文只跟分组开关，不被 OpenAI 账号开关否决 | `fd82dfd5`（#5573） | **已合**（随 P2 一起带） |
| 未知 Grok 文本兜底排除 image/video/audio | `e29b93a1`（#5573） | **已合**（随 P0 一起带） |
| relay `x-codex-turn-state`、防跨账号 echo | `8219dcfc` + 测试补丁 `4d9fedee`（#5668） | **已合**（2026-08-15）：`openai_codex_turn_state.go` + 响应侧 relay（流式 / 非流式 / SSE→JSON / 透传写头）+ 出站守卫。测试取 `8219dcfc` 版本并剔掉属于下一项的 beta 头用例（该三个用例已随 `ea48805` 补回） |
| session-level beta features + 探测 native compaction v2 | `8ae6d8f6`（#5668） | **已合**（2026-08-15）：探测改打流式 `/responses` + `compaction_trigger`，2xx 无 compaction item 记为不支持；`x-codex-beta-features` 在 HTTP / 透传 / WS 三处按会话级补注（探测路径另有第四处窄补注 `ensureOpenAIRemoteCompactionV2BetaFeature`，且 OAuth 探测套用指纹收敛头，为本 fork 相对上游的增量）；`compactProbeSessionID` 改 SHA-256 确定性派生（UUIDv4 形状，账号级稳定） |
| Codex fingerprint 改成 **opt-in，默认 off**，并覆盖透传 | `fce41e31`（#5668） | **已合**（2026-08-15）：`openai_codex_fingerprint.go` 取 v0.1.177 最终版；Forward / 透传各自解析、各路径内 body 与出站头共享同一份收敛 ID；只合指纹，同提交里的 turn-state 与 beta 头留给下一项 |
| 保留 remote compaction v2、native/legacy 分流 | `9662cff2` / `a8b9ea22`（#5641） | **大部分已有，暂缓**。本地 `normalizeOpenAIResponsesCompactRequest` 已让 native v2 保持裸 `/responses` 直通（9662cff2 的核心）；缺的只有 `openai_compaction_context.go`（渠道限制按 forward model 判定的窄修复）。合 `8ae6d8f6` 时若冲突再顺手带 |
| 分组用量日聚合 + 时区 | `cb7b0379` 等（#5649） | **不合**：本 fork 刚藏过用量筛选/图表，且聚合 SQL 面向 PG |
| 账号页自动刷新偏好改为模块初始化时恢复 | `e215c98c`（#5573） | 可跳过：28 行前端小改，本 fork AccountsView 已简化，收益低 |
| CI 里 Go **1.26.6** | 见 #5649 相关 CI 修复 | 低优先级 chore。本地 go.mod + 3 个 workflow 文件里 4 处断言（backend-ci ×2、security-scan、release）仍写死 `1.26.5`，升则一起改 |

推荐合入顺序：① fingerprint（已合）→ ② turn-state relay + echo 守卫（已合）→ ③ compact 探测 v2 + beta 头（已合）。三者都在 #5668 内，文件互不重叠但语义相关。

---

## 10. 不要动

| 项 | 原因 |
|----|------|
| 整 PR cherry-pick / `merge upstream/main` | 历史无共同祖先；ent 生成物、PG SQL 会炸 |
| 上游 `backend/ent/group*.go`、`mutation.go` | 用 `go generate` |
| 上游 `221_group_model_pricing.sql` 原文件 | PG；本地 221 已被占用 |
| `AccountsView.vue` 大改、用量格刷新 | 和本 fork 简化过的用量 UI 冲突 |
| 备份 leader 锁、分卷上传 | 单机 SQLite 用不上 |
| Codex fingerprint `session` 默认 | `v0.1.177` 已改 opt-in / off |
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
