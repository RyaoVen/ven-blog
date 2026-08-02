# Agent 能力设计总览（agent-design）

博客项目接入 AI agent 能力的完整设计。5 个单元各自独立成文（可独立验收、独立 PR），本文档是索引与跨单元契约。

## 背景

用户（唯一作者）希望通过 agent 完成：发布文章/动态、自动审核评论（有问题发邮件、没问题自动放行）、修改个人主页。整体架构分三层：**/api/mcp 网关（遥控器）→ agent 侧 skill（操作手册）→ 后端审核 worker（自动巡航）**，密钥机制是共同地基。

## 单元索引

| 单元 | 文档 | 内容 | 依赖 |
|---|---|---|---|
| U1 密钥管理 | [unit-1-key-management.md](unit-1-key-management.md) | `api_keys` 表 + 面板生成/列表/吊销（明文仅一次）+ `apikeyapp.AuthenticateKey` | 无 |
| U2 /api/mcp 网关 | [unit-2-mcp-gateway.md](unit-2-mcp-gateway.md) | Bearer key 校验 + JSON-RPC 分发 + 14 个 action 契约 | U1（AuthenticateKey） |
| U3 UGC 审核语义 | [unit-3-ugc-moderation.md](unit-3-ugc-moderation.md) | 评论三态 + 留言板纳入审核 + 面板管理 + 行为变更 | 无 |
| U4 审核 worker | [unit-4-moderator-worker.md](unit-4-moderator-worker.md) | LLM 判定 + ticker + 摘要邮件 + 失败安全 | U3（rejected 状态） |
| U5 agent skill | [unit-5-agent-skills.md](unit-5-agent-skills.md) | publish-post / review-comments / edit-author-page 三个 SKILL.md 草稿 | U2（action 在线） |

## 跨单元关键契约（实现时对齐这些，别各写各的）

- **鉴权**：`Authorization: Bearer ven_xxx`；`apikeyapp.Service.AuthenticateKey(rawKey) (userID int64, err error)`（U1 定义，U2 经 `KeyAuthenticator` 接口调用）；`/api/mcp` 只认 key 不认 cookie，401 不带 `X-Ven-Login-Path`。
- **migration 编号**：`008_api_keys.sql`（U1）、`009_ugc_status.sql`（U3），注意合并顺序按此编号。
- **审核状态**：comment 与 guestbook 同构三态 `pending/approved/rejected` + `rejected_reason`（≤200，仅 reject 非空）；公开查询只返 approved；评论公开查询已满足（无需改），**guestbook 公开查询必须改**（现为全量，本次唯一动现有公开行为处）。
- **失效声明**（只在接口层调用）：post 宿主 → `declarePostsChanged`；moment 宿主 → `DataChange("/moments")`；作者页 → `InvalidatePage("/author/"+name)`；approve/recover 才需要读者侧失效，reject 不需要。
- **settings 键**：新增 `ugc_ai_moderation`（U4，未设置视为 on）；审核主开关沿用 `comment_moderation`。
- **配置 env**：`BLOG_LLM_BASE_URL`（默认 DeepSeek）/`BLOG_LLM_API_KEY`/`BLOG_LLM_MODEL`、`BLOG_MODERATOR_INTERVAL`、`BLOG_MODERATOR_BATCH`、`BLOG_API_KEY`（agent 侧存 key）。
- **邮件**：复用 `mailer.Mailer.Send`；收件人 `settingsapp.AuthorEmail()`；面板人工操作不主动发邮件，驳回/不确定邮件由 U4 worker 与 U2 action 触发。

## 实施顺序

U1 → U2 → U3 → U4 → U5。U1/U3 无相互依赖可并行；U2 依赖 U1；U4 依赖 U3；U5 最后（依赖 U2 在线）。

## 开放问题（实现时逐项处理）

1. **author.update 的 quotes**：`settingsapp.Content` 含 Quotes 但现有 admin PUT `/api/admin/author/content` 未暴露保存——U2 的 `author.update` 直接调 `settingsapp.SetQuotes` 即可绕过，不扩 admin 接口。
2. **多实例重复判定**（U4）：单实例部署无碍；多实例时可用"审核前先取锁/复核状态"缓解，v1 不引入分布式锁。
3. **审核最坏耗时**（U4）：pending 量大 + 重试时单轮可能较长，ticker 防重叠 + 每轮上限已缓解。
4. **MomentCommentCounts 计数偏差**（U3）：评论计数未按 approved 过滤，建议顺手修（独立小 commit，不进本单元验收）。
5. **面板路径**：面板源码在仓库根 `src/admin/`（非 frame/node/），已按此写设计。
