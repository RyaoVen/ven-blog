# ven-blog

基于 **VenHybird** 混合渲染框架的博客系统：Go(Fiber) 网关 + Node SSR worker + MySQL，内置 AI Agent 能力（文章/动态发布、评论自动审核、个人主页编辑）。

- **首屏 SSR 直出**，SPA 接管后续导航；文章/动态静态页 ISR 物化，数据变更事件总线失效 + SSE 推送
- **业务与框架严格分层**：业务代码只在 `frame/go/build/`（DDD 四层），框架层（`frame/go/internal` + `hybrid` + `frame/node`）与上游框架仓保持一致，不私改
- **Agent 能力开箱即用**：`/api/mcp` JSON-RPC 网关 + API Key 身份 + 自动审核 worker + 订阅通知

## 功能一览

- 文章（分类/标签/置顶/展示柜）与动态（置顶/点赞/评论）
- 评论审核：AI 自动审核（明显违规驳回、不确定保持待审）+ 后台人工兜底 + 摘要邮件
- 留言板、友链、技能、个人主页（读者页 + 作者页）
- 邮箱注册登录（验证码）、@提及邮件通知、新文章订阅通知
- 访问统计（仪表盘 PV 折线、文章点击/点赞/收藏）
- 合规开关：评论总开关、注册登录入口开关（作者后台保留）
- MCP Agent 入口：`post.*` / `moment.*` / `comment.*` / `author.*` 共 14 个 action

## 技术栈与架构

| 组件 | 位置 | 端口 | 说明 |
| --- | --- | --- | --- |
| Node SSR worker | `frame/node/` | 3000 | 页面路由唯一真相源（`src/**/page.tsx`）、SSR 渲染 |
| Go 网关 | `frame/go/` | 8080 | 唯一公网入口、业务 API、鉴权、缓存/ISR/SSE |
| MySQL | 外部 | 3306 | 业务库 `ven_blog`（启动自动建库建表 + 迁移） |
| 部署工具 | `tools/deploy/` | — | 跨平台 TUI + 子命令（配置/构建/进程管理） |

```text
Browser ──▶ Go :8080 ──▶ Node :3000（SSR 渲染 /pages 路由表）
                 │
                 └── MySQL（业务数据）
```

## 快速开始

前置：Go 1.25+、Node 22+、MySQL 8。启动依赖 **Node 先起**（Go 启动时拉取 `/pages` 路由表）。

```bash
# 终端一
cd frame/node && npm install && npm run build && node dist/main.js
# 终端二（等 Node 起来后）
cd frame/go && go run .
```

环境变量：`cp env.local.example .env.local` 后填 `BLOG_MYSQL_DSN`；首次启动种子 author 的 `BLOG_AUTHOR_PASSWORD` **必配**（未配置拒绝启动）。完整说明见 [docs/usage.md](docs/usage.md)。

### 一键部署（推荐）

```bash
cd tools/deploy && go build -o deploy .
./deploy config   # 配置向导（含 VEN_INTERNAL_TOKEN / BLOG_SECRET_KEY 自动生成）
./deploy build && ./deploy start
./deploy status / logs / stop / restart
```

## 目录结构

```text
frame/go/
  build/                  # 业务层（DDD 四层，唯一业务改动区）
    domain/               #   领域层：实体/值对象/仓储接口
    application/          #   应用层：用例服务
    infrastructure/       #   基础设施：MySQL 仓储（embed 迁移）、SMTP、LLM、加密
    interfaces/           #   接口层：页面/API/MCP/认证注册
    register.go           #   组装根（composition root）
  hybrid/                 # 框架胶水层（与上游同步，不私改）
  internal/               # 框架实现（与上游同步，不私改）
frame/node/               # Node SSR worker（框架侧）
src/**/page.tsx           # 页面路由（Node 是唯一真相源）
tools/deploy/             # 跨平台部署工具（独立 module）
docs/
  usage.md                # 使用文档：启动/环境变量/Agent 能力/运维
  agent-design/           # AI Agent 能力设计文档（unit-1 ~ unit-5）
  framework-requests.md   # 业务仓 → 框架仓的需求 Brief（协作记录）
```

## 关键配置（节选）

| 变量 | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `BLOG_MYSQL_DSN` | ✅ | — | MySQL DSN（自动建库建表） |
| `BLOG_AUTHOR_NAME` / `BLOG_AUTHOR_PASSWORD` | 首次必配（密码） | author / — | 种子 author |
| `VEN_INTERNAL_TOKEN` | 生产必改 | development-token | Go↔Node 内部令牌（两侧一致，生产拒绝默认值） |
| `BLOG_SECRET_KEY` | 可选 | — | 敏感配置加密密钥（32 字节 hex；配置后 smtp_pass/llm_api_key 存库加密） |
| `BLOG_LLM_BASE_URL` / `BLOG_LLM_API_KEY` / `BLOG_LLM_MODEL` | 可选 | DeepSeek / — / deepseek-chat | 自动审核 LLM（配了 API_KEY 才启动 worker） |
| `BLOG_SITE_URL` | 可选 | http://127.0.0.1:8080 | RSS/邮件链接拼接（设置页可覆盖） |

完整变量表见 [docs/usage.md](docs/usage.md#环境变量总表envlocal)。

## Agent 能力

三层架构：**面板 API Key（身份）→ `/api/mcp`（统一入口）→ skill（工作流）**，另有后端自动审核 worker。

```bash
curl -X POST http://127.0.0.1:8080/api/mcp \
  -H "Authorization: Bearer <ven_xxx>" -H "Content-Type: application/json" \
  -d '{"action":"post.create","payload":{"title":"你好","category":"技术","content":"..."}}'
```

- 14 个 action：`post.create/update/delete/list`、`moment.create/delete/list`、`comment.list_pending/list/approve/reject/recover`、`author.get/update`
- API Key 在 `/admin/settings` 生成（明文仅展示一次，库内仅 sha256）
- 常用 skill：`/publish-post`（发文章/动态）、`/review-comments`（审核复核）、`/edit-author-page`（改个人主页）

详见 [docs/usage.md](docs/usage.md#agent-能力)。

## 安全与合规

- 内部令牌强制校验、Node 熔断、页面缓存 stale-while-revalidate、登录/发码限速
- SMTP 密码与 LLM API Key 加密存储（BLOG_SECRET_KEY）、降级日志不含正文
- 评论总开关、注册登录入口开关（适配国内备案合规）
- 安全与失效语义变更记录见 [CHANGELOG.md](CHANGELOG.md)

## 开发约定

- 提交前检查：`cd frame/go && go build ./... && go vet ./... && go test ./...`；`cd frame/node && npm run typecheck && npm test`
- Git 工作流红线（issue → 分支 → PR）与 DDD 分层纪律见 [AGENTS.md](AGENTS.md)
