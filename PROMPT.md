# ven-blog 项目介绍提示词

> 用法：把下面「提示词正文」整段粘贴给 AI 代理作为开场 briefing。
> 提示词假设代理的工作区就是本仓根目录。

---

## 提示词正文

你在为 **ven-blog** 项目工作——一个基于 VenHybird 混合渲染框架的博客系统：Go(Fiber) 网关 + Node SSR worker + MySQL，带完整 AI Agent 能力。项目**已完整可用**，不是骨架：文章/动态/评论审核/留言板/订阅通知/访问统计/合规开关都已实现并在线运行。你的任务是维护与扩展业务，不是重写。

### 先读这几篇文档（就在仓里）

1. `README.md`——项目总览、架构、目录结构、关键配置、Agent 能力
2. `AGENTS.md`——分层纪律、检查命令、测试风格、环境坑、git 工作流红线（**必须遵守**）
3. `docs/usage.md`——怎么启动、环境变量总表、Agent 能力细节、运维排查
4. `docs/agent-design/`——Agent 能力的设计文档（MCP 网关/审核 worker 等的实现依据）

### 启动方式（两个进程 + MySQL）

```bash
# 前置：MySQL 8（库 ven_blog 自动创建），.env.local 填 BLOG_MYSQL_DSN 与 BLOG_AUTHOR_PASSWORD
cd frame/node && npm install && npm run build && node dist/main.js   # :3000，先起
cd frame/go && go run .                                               # :8080，后起（依赖 Node 拉路由表）
```

### 业务边界：你能动哪里

- **能动**：`frame/go/build/`（DDD 四层业务：domain / application / infrastructure / interfaces + register.go 组装根）和 `src/`（React 页面，文件路径即路由，`[id]` → `:id`）
- **不能动**：`frame/go/internal/`、`frame/go/hybrid/`、`frame/node/` 的框架代码——这是上游框架仓的代码（业务仓与其保持同步）。**觉得框架缺能力时不要私改框架**，把需求写成 Brief（参考 `docs/framework-requests.md` 的格式）交给框架仓处理
- 分层依赖纪律：`interfaces → application → domain`，`infrastructure` 实现 `domain` 的仓储接口，由 `register.go` 组装注入；**失效声明（InvalidatePage/DataChange）只在接口层调用**

### 关键约定（都踩过坑）

- 失效语义：`DataChange(pattern, ...params)` 只用于静态页（ISR）；**动态页（Page 声明）只能用 `InvalidatePage("/具体路径")`**——带查询串或动态参数的动态页不能用 DataChange（会报 static page not declared）
- 页面缓存 1 分钟 TTL：数据变更后必须补失效声明，否则线上显示旧数据（v1.2.6 已补齐 /search、/users/:name，新页面照此办理）
- 敏感信息：SMTP 密码/LLM key 存库自动加密（BLOG_SECRET_KEY）；日志里**永远不要打印邮件正文、密码、令牌明文**
- 内部令牌 `VEN_INTERNAL_TOKEN` 生产强制（空/development-token 拒绝启动）；Go↔Node 两侧一致

### Agent 能力（已是功能，不是要你重写）

- `/api/mcp`：JSON-RPC 网关（14 个 action：post.*/moment.*/comment.*/author.*），Bearer `ven_` key 鉴权，只认 key 不认 cookie
- 自动审核 worker：LLM 判定评论/留言（明显违规驳回、不确定保持待审、失败回滚重审），摘要邮件通知
- 手动复核：`/admin/comments`、`/admin/guestbook` 后台；`/review-comments` skill

### 工程纪律

- 提交前必跑：`cd frame/go && go build ./... && go vet ./... && go test ./...`；`cd frame/node && npm run typecheck && npm test`
- git 工作流（红线见 AGENTS.md）：一单元一 issue → 分支 `<type>/issue-<N>-<slug>` → conventional commits 中文描述带 `(#N)` → 检查通过 → PR → squash 合并；**不直推 master、不 force-push**
- 测试风格：Go 用 fake 仓储 + httptest；Node 只测纯逻辑（不碰 DOM/React）；本机跑不了 `-race`（DLL 问题），并发正确性靠设计
- 版本号、CHANGELOG 按语义化版本维护；封版走 issue + PR

### 建议的第一步

先跑起来（部署工具 `tools/deploy` 或手动两终端）→ 用面板生成一个 API Key → 调 `/api/mcp` 发一篇测试文章验证链路 → 再开始你的任务。
