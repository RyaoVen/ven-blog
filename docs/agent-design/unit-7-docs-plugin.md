# Unit 7 — Docs 插件设计（草案 v0.1）

> 状态：**已实施（2026-09-16，issue #6~#12，PR #6~#12；review 修复见 issue #13）**
> 归属：ven-blog 业务仓库，**首个运行在 Unit 6 插件内核上的插件**（`build/plugins/docs/`）
> 上游：Unit 6（插件系统治理）、框架需求 7（catch-all 路由）
> 用户已拍板决策：树形目录 / hook 四件套（webhook、MCP 增量、导入导出、SSE）/ admin 完整编辑器
> 日期：2026-09-15

---

## 1. 定位

与 posts（博客文章）、moments（动态）并列的第三种内容形态：**长期维护的树形文档**（笔记、项目文档）。核心特点：

- 树形目录组织（侧边栏导航），path 为人类/agent 友好主标识；
- **MCP 为第一写入通道**（agent 原生协作）；
- 四形态 hook 协作：出站 webhook、MCP 增量拉取、Markdown 导入/导出、SSE 实时推送。

## 2. 数据模型（`docs` 表，嵌入式迁移自动建表）

统一节点表（文档与目录同表，目录节点可带 index 正文）：

```sql
CREATE TABLE IF NOT EXISTS docs (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  parent_id  BIGINT NULL,                -- NULL = 根级；自关联
  slug       VARCHAR(64)  NOT NULL,      -- 同级唯一（与 parent_id 组成唯一约束）
  path       VARCHAR(512) NOT NULL,      -- 物化全路径 'ai/attention'，UNIQUE
  kind       ENUM('doc','section') NOT NULL DEFAULT 'doc',
  title      VARCHAR(128) NOT NULL,
  summary    VARCHAR(200) NOT NULL DEFAULT '',
  content    MEDIUMTEXT,                 -- markdown
  tags       JSON NULL,                  -- ≤8 个，各 ≤24 字符（沿用 post 约束）
  sort_order INT NOT NULL DEFAULT 0,
  status     ENUM('draft','published') NOT NULL DEFAULT 'published',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uq_parent_slug (parent_id, slug),
  UNIQUE KEY uq_path (path),
  KEY idx_updated (updated_at)           -- since 增量拉取用
);
```

**领域规则**（`plugins/docs/domain.go`）：

- slug 合法字符：`[a-z0-9-]`（小写字母/数字/连字符），禁 `..`、首尾连字符；
- 删除 section 默认拒绝（存在子节点时返回 validation 错），`recursive=true` 级联删除子树；
- move/rename 级联更新子树 path（事务内：锁行序 = 按 path 前缀升序，防死锁）；
- doc.create 时父 section 不存在则**隐式自动补齐**（对 agent 友好：`doc.create path=ai/attention` 自动建 `ai` section）；
- path 总长 ≤512，树深软上限（Phase 1 定 8，防御性）。

## 3. Go 侧模块（插件内聚布局）

```
frame/go/build/plugins/docs/
├── plugin.go       # 唯一规范文件：New() 入口（纯构造）+ Plugin 实现（Meta/Register/Start/Stop）
├── domain.go       # 实体 + 规则 + 仓储接口（实现文件，命名自由）
├── app.go          # 用例服务（无 fiber 依赖）
├── repo.go         # MySQL 仓储 + 建表迁移（自建连接：默认复用 BLOG_MYSQL_DSN、可 BLOG_DOCS_MYSQL_DSN 覆盖；Stop 关池；不 import 主业务 persistence 包）
├── interfaces.go   # 页面注册 + /api/admin/docs CRUD + 失效声明
├── mcp.go          # doc.* actions（经 Runtime.MCP 注册）
└── dochook.go      # webhook 投递器（Start/Stop 生命周期）
```

> 入口约定遵循 unit-6 §5.3：`plugin.go` 之外的文件均为自由组织的实现细节，无规范性约束。

失效声明：所有写操作 → `DataChange(ChangeEvent{Pattern:"/docs/*"})` 全局失效（Phase 1 简化；精确到子树留待后续）+ SSE 联动（总线 flush 后自动）。

## 4. MCP 接口（`doc.*`，全部经 Unit 6 的 `Runtime.MCP.RegisterAction` 注册）

沿用 unit-2 契约：`POST /api/mcp`、`{action,payload}`、Bearer key、`{ok,data}` / `{error:{code,message}}`、错误码复用、ID 字符串化、写操作非幂等（重试前先 `doc.list` 查证——与 post 同纪律）。

| action | payload | 语义 | 分期 |
|---|---|---|---|
| `doc.create` | `{title, path, kind?, content?, summary?, tags?, order?, status?}` | 隐式补父；重名 path → validation | M1 |
| `doc.get` | `{path}` | 节点 + 直接子节点（MCP 全量含 draft） | M1 |
| `doc.list` | `{path?, recursive?, limit?, offset?}` | 列子树/平铺；limit 缺省 100 | M1 |
| `doc.tree` | `{}` | 完整导航树（页面侧只取 published，MCP 侧全量） | M1 |
| `doc.update` | `{path, title?, content?, summary?, tags?, order?, status?}` | **部分更新**（author.update 先例：指针判存在） | M1 |
| `doc.delete` | `{path, recursive?}` | 有子节点且未 recursive → validation | M1 |
| `doc.move` | `{path, newParent?, newSlug?, order?}` | 移动/改名/重排，级联子树 path | M4 |
| `doc.list` 扩展 | `{since}` | ISO8601，按 updated_at 过滤（**增量拉取**） | M4 |
| `doc.import` | `{files:[{path, content}], mode:"upsert"}` | frontmatter(title/tags/order/status)+md 解析；**1MB body 上限 → 分批** | M5 |
| `doc.export` | `{path?, recursive?}` | 按树导出 markdown 包（frontmatter 还原） | M5 |

## 5. Hook 四件套

| 形态 | 实现 | 分期 |
|---|---|---|
| **SSE 实时推送** | 框架既有能力：docs 失效事件入总线 → flush 后 `NotifyEvents` 推在线浏览器，**零新基础设施**（接线即可） | M2 |
| **MCP 增量拉取** | `doc.list since`（上表） | M4 |
| **出站 webhook** | settings 扩展 `plugin.docs.webhook_url/secret`（AES-GCM 加密，settingsapp 先例）；interfaces 层写操作成功后异步 POST `{event:"doc.created/updated/moved/deleted/imported", path, at}`（imported 为批量汇总事件，体见 §9 决议 2），头 `X-Ven-Signature: HMAC-SHA256(secret, body)`；简单退避重试 3 次，失败仅日志；投递器挂插件 Start/Stop 生命周期 | M4 |
| **Markdown 导入/导出** | `doc.import` / `doc.export`（上表）；服务本地笔记库首发（`~/Documents/MyNotes`、`~/Documents/Docs`） | M5 |

## 6. 前端（页面插件 = 源码级模块）

```
src/docs/page.tsx                    # 文档首页：目录树视图
src/docs/[...slug]/page.tsx          # 文档页（catch-all，依赖框架需求 7）
src/admin/docs/page.tsx              # 树形管理列表
src/admin/docs/new/page.tsx          # 新建（复用/适配 admin/editor.tsx）
src/admin/docs/[id]/edit/page.tsx    # 编辑
```

- 侧边栏目录树组件（同级按 sort_order、字典序）、正文复用 `lib/markdown.ts`、上一页/下一页；
- admin 编辑器适配点：parent 选择器、order、status；editor.tsx 若与 post 耦合过紧则**复制适配，不强行抽象**（AGENTS.md 惯例优先）；
- header 导航加 Docs 入口；
- 搜索页（存量 `src/search`）加 scope 切换（全站/博客/Docs）：`scope=all|blog|docs`，默认 all，经 unit-6 §5.4 SearchRegistry 聚合，不装 docs 插件时与现状 bit-exact；
- `/api/admin/docs` CRUD：cookie + author 角色（`a.Post/Put/Delete`）。

**路由风险（已源码确认）**：Node `matchRoute` 与 Go `Declaration.Match` 均要求段数相等，无 catch-all → 依赖框架需求 7；过渡兜底 = 固定深度三条声明 `/docs/:a`、`/docs/:a/:b`、`/docs/:a/:b/:c`（树深上限 3）。

## 7. 里程碑（GitCode，走 devflow 逐 issue 受理）

| 里程碑 | 内容 | 验收要点 |
|---|---|---|
| M1 | docs 数据层 + docapp + MCP 基础七 action（create/get/list/tree/update/delete + 建表） | MCP 冒烟全绿；失效声明生效；单测覆盖领域规则 |
| M2 | 前端 /docs（树导航/渲染/ISR/SSE） | 页面 200；改 doc → ISR 再生效 + SSE 推送 |
| M3 | admin 完整编辑器 + /api/admin/docs | 后台全流程可用 |
| M4 | doc.move + since 增量 + webhook 出站 | HMAC 签名验证通过 |
| M5 | doc.import/export + 首批内容导入 | 本地笔记选摘入库成树 |
| M6 | 搜索集成（scope：全站/博客/docs） | 三 scope 冒烟；不装 docs 插件 bit-exact |

## 8. 测试与验收

- 领域规则表驱动单测（slug 校验/级联删除/move 事务/隐式补齐）；
- MCP handler 测试参照 `mcp_test.go` 表驱动风格；
- 本地冒烟：Node(:3000)+Go(:8080)+MySQL 起服，curl 走 `~/.agents/skills/publish-post` 同款 mcp() 封装；
- 契约验收：**docs 插件完全经 Unit 6 内核注册**，register.go 主体与 mcp.go 内置区零改动（这是 unit-6 的验收条款，双向约束）。

## 9. 设计决议（2026-09-16 用户拍板）

1. **仓储 DB 连接：插件自实现，不走主业务**——插件自建 `database/sql` 连接与建表迁移，**不 import 主业务 `infrastructure/persistence` 包**；默认复用 `BLOG_MYSQL_DSN`（同库零新配置），允许 `BLOG_DOCS_MYSQL_DSN` 覆盖；配套纪律（已同步 unit-6 §6 治理规则）：连接池上限 MaxOpenConns ≤ 4、`Stop()` 必须关池（评审 P0）。已知代价：与主业务无跨域事务——docs 场景不存在此需求，接受。
2. **webhook 覆盖 `doc.imported` 批量事件**——import 逐文件发 `doc.created` / `doc.updated`（下游精确追踪），收尾另发一条汇总 `{event:"doc.imported", root, total, created, updated, failed, at}`（下游做"一次导入完成"触发）。
3. **搜索纳入，带 scope 可选项**——三档：全站（默认）/ 博客 / docs；实现经 unit-6 §5.4 SearchRegistry（内置 provider 名 `blog`、docs 插件注册 `docs`），不装 docs 插件时 `scope=all` 与现状 bit-exact；排期 M6。
