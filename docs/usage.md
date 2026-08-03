# ven-blog 使用文档

本仓库是 **VenHybird** 框架 + **ven-blog** 博客业务的完整实现：Go(Fiber) 网关 + Node SSR worker 混合渲染，业务代码在 `frame/go/build/`（DDD 四层）。框架概念见 [README.md](../README.md)，本文档聚焦**怎么跑起来、怎么用 agent 能力、怎么运维**。

## 项目构成

| 组件 | 位置 | 端口 | 说明 |
| --- | --- | --- | --- |
| Node SSR worker | `frame/node/` | 3000（`VEN_NODE_PORT` 可配） | 页面路由唯一真相源（`src/**/page.tsx`）、SSR 渲染 |
| Go 网关 | `frame/go/` | 8080（`VEN_LISTEN_ADDR` 可配） | 唯一公网入口、业务 API、鉴权、缓存/ISR |
| MySQL | 外部 | 3306 | 业务库 `ven_blog`（启动自动建库建表+迁移） |
| 部署工具 | `tools/deploy/` | — | 跨平台（Windows/Linux）TUI + 子命令，配置/构建/进程管理 |

启动依赖：**Node 必须先起**（Go 启动时拉取 `/pages` 路由表），Go 启动时连 MySQL 并自动执行迁移。

## 前置条件

- Go 1.25+、Node 22+（npm）、MySQL 8（本地或可达）
- `.env.local`（仓库根，gitignore）：`cp env.local.example .env.local` 后填 `BLOG_MYSQL_DSN`
- 首次启动用户表为空时自动种子 author 账号（`BLOG_AUTHOR_NAME` 默认 `author`；`BLOG_AUTHOR_PASSWORD` **必配**，未配置拒绝启动）；种子 reader 密码 `BLOG_READER_PASSWORD` 可配（默认 `reader123`）

## 快速部署

### 推荐：部署工具（Windows/Linux 通用）

```bash
cd tools/deploy && go build -o deploy .     # 或 go run .
./deploy check          # 环境检测（go/node/npm/MySQL/端口/配置）
./deploy config         # 配置向导（生成/更新 .env.local）
./deploy build          # Node（npm ci + tsc）+ Go（bin/ven_hybird）
./deploy start          # Node 先起→等 /pages 就绪→Go 后起→等 /api/site 就绪
./deploy status         # 进程 + 端口 + MySQL 状态
./deploy logs [-n N]    # tail 日志（logs/node.log、logs/go.log）
./deploy stop           # 强杀停止（按 PID 文件）
```

无参运行 `./deploy` 进 TUI 菜单（状态面板 + 检测/配置/构建/启动/停止/重启/日志）。子进程已 detach（关终端不杀），`stop` 按 PID 强杀仍有效。端口从配置读取：Node `VEN_NODE_PORT`（默认 3000）、Go `VEN_LISTEN_ADDR`（默认 8080）。详见 [tools/deploy/README.md](../tools/deploy/README.md)。

### 手动方式

```bash
# 终端一
cd frame/node && npm install && npm run build && node dist/main.js
# 终端二（等 Node 起来后）
cd frame/go && go run .
```

## 环境变量总表（.env.local）

| 变量 | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `BLOG_MYSQL_DSN` | ✅ | — | `root:密码@tcp(127.0.0.1:3306)/ven_blog?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci` |
| `BLOG_AUTHOR_NAME` / `BLOG_AUTHOR_PASSWORD` | 首次启动必配（密码） | author / — | 种子 author（仅首次；未配密码拒绝启动） |
| `BLOG_READER_PASSWORD` | 可选 | reader123 | 种子 reader 密码（仅首次） |
| `VEN_INTERNAL_TOKEN` | 生产必改 | development-token | Go↔Node 内部调用令牌，两侧一致 |
| `BLOG_SITE_URL` | 可选 | http://127.0.0.1:8080 | RSS/邮件链接拼接 |
| `VEN_NODE_PORT` | 可选 | 3000 | Node worker 端口（Node 与部署工具同读） |
| `VEN_LISTEN_ADDR` | 可选 | :8080 | Go 网关监听地址 |
| `VEN_COOKIE_SECURE` | 可选 | true | 鉴权 cookie 带 Secure 标志（仅 HTTPS 发送）；本地 http 开发设 `false`，否则登录不上 |
| `BLOG_LLM_BASE_URL` / `BLOG_LLM_API_KEY` / `BLOG_LLM_MODEL` | 可选 | DeepSeek / — / deepseek-chat | 自动审核 LLM（OpenAI 兼容；**配了 API_KEY 才启动 worker**） |
| `BLOG_MODERATOR_INTERVAL` | 可选 | 5m | 审核轮询间隔 |
| `BLOG_MODERATOR_BATCH` | 可选 | 20 | 每类宿主（评论/留言板）每轮处理上限 |

框架级 `VEN_*`（ISR/Redis/缓存等）见 [README.md](../README.md) 配置表。

## Agent 能力

三层架构：**面板密钥（身份）→ /api/mcp（统一入口）→ skill（工作流）**，另有后端**自动审核 worker（自主运行）**。

### 1. API Key（程序化身份）

`/admin/settings` →「API 访问密钥」区块：

- **生成**：明文仅展示一次（关窗后不可再查，只能吊销重发）；支持多个 key 按用途分（如 `zcode-agent`、`moderator-worker`）
- **列表**：脱敏展示（前缀/名称/创建时间/最后使用），吊销即时生效
- 存储：库内仅 sha256，明文永不落库

### 2. /api/mcp（agent 统一入口）

`POST /api/mcp`，JSON-RPC 风格，**只认 key 不认 cookie**：

```bash
curl -X POST http://127.0.0.1:8080/api/mcp \
  -H "Authorization: Bearer <ven_xxx>" -H "Content-Type: application/json" \
  -d '{"action":"post.create","payload":{"title":"你好","category":"技术","content":"..."}}'
```

- 请求：`{"action": "...", "payload": {...}}`；成功 `{"ok":true,"data":{...}}`；失败 `{"error":{"code":"...","message":"..."}}`
- 错误码：`invalid_key`(401) / `bad_request`(400) / `validation`(400) / `not_found`(404) / `forbidden`(403) / `internal`(500)
- 请求体上限 1MB；401 不带 `X-Ven-Login-Path`（那是 web 登录重定向用的）

| action | 说明 |
| --- | --- |
| `post.create / update / delete / list` | 文章 CRUD（payload 对齐 admin 字段：title/category/content/summary/coverUrl/tags） |
| `moment.create / delete / list` | 动态 |
| `comment.list_pending / list / approve / reject / recover` | 评论审核（reject 需 `reason` ≤200；recover 从 rejected 恢复） |
| `author.get / update` | 个人主页（**部分更新**：只传改动字段，未提及字段保持原值；显式传空数组=清空；github 需走后台设置页） |

### 3. Skill 工作流（ZCode 用户级）

配置 `BLOG_API_KEY`（面板生成的 key）后可用：

- `/publish-post`：发文章/动态（确认标题分类→创建→验证→汇报）
- `/review-comments`：复核评论队列（worker 的兜底通道，驳回/恢复/通过）
- `/edit-author-page`：改个人页（简介/技能/友链/项目/引用，部分更新）

### 4. 自动审核 worker（评论+留言板）

- 启动条件：`BLOG_LLM_API_KEY` 非空 && settings 开关 `ugc_ai_moderation`（默认开）
- 语义：明显违规→驳回（记原因）；正常→放行；**不确定→保持待审**（交人工）
- **失败安全**：LLM 超时/出错重试 1 次仍失败→保持待审，绝不误杀
- **邮件节流**：每轮一封摘要（驳回明细+需人工复核明细+面板链接），已报告条目按 `moderator_reported`（settings 键）去重不重复刷屏；SMTP 未配置时降级日志
- 人工兜底：`/admin/comments`（三态列表+驳回原因+恢复）、`/admin/guestbook`（同款）

## 运维与排查

| 场景 | 做法 |
| --- | --- |
| 看日志 | `deploy logs` 或 `logs/node.log` / `logs/go.log` |
| 停止/重启 | `deploy stop` / `deploy restart`（按 PID 强杀，detach 进程同样有效） |
| Go 启动后立即退出 | 查 `logs/go.log`——常见 `BLOG_MYSQL_DSN` 错误、MySQL 未启动 |
| 端口被占 | `deploy check` 看占用位；`deploy start` 会先报冲突并提示 `stop` |
| npm ci 报文件占用 | Windows 上多为残留 node 进程或杀毒软件，先清 node 进程重试 |
| 面板登录 401 | 会话 24h TTL（`VEN_SESSION_TTL`），重新登录即可 |
| 评论不显示 | 先审后发开关 `comment_moderation` 开时新评论为待审，需 approve 或等 AI 判定 |
