# Unit 2 设计：`/api/mcp` 网关（agent 统一入口）

- 状态：设计定稿（实现分两批：本单元全部；`comment.reject`/`comment.recover` 依赖 Unit 3）
- 前置依赖：Unit 1（API key 管理，`apikeyapp.Service.AuthenticateKey`，并行设计中）；Unit 3（评论 rejected 状态，仅影响两个 action 的实现）
- 与现有鉴权完全隔离：本入口**只认 `Authorization: Bearer ven_xxx`，不认 cookie**，不参与 hybrid 的 cookie 鉴权链（`hybrid/api.go:36-58`），401 不带 `X-Ven-Login-Path` 头。

---

## 1. 目标

外部 AI/脚本（agent）持 API key 通过**单一端点** `POST /api/mcp` 执行博客管理操作：

- 发布/编辑/删除/查询文章（post.*）
- 发布/删除/查询动态（moment.*）
- 审核/拒绝/恢复/查询评论（comment.*）
- 读取/更新作者个人页内容与资料（author.*）

关键约束（公共基线，本单元必须对齐）：

1. 端点用**原生 fiber 路由**注册，先例 `build/interfaces/auth.go:23-66`（`server := a.Server(); server.App().Post("/auth/login", ...)`，错误 `ctx.Status(code).JSON(fiber.Map{"error": "..."})`）。`/api/mcp` 写全路径直接注册在 fiber 上，**不走** `a.Post(...)`（那会经 hybrid cookie 鉴权，见 `hybrid/api.go:36-58`）。
2. 认证通过后 userID 注入 `ctx.Locals`，所有 action 视为 author 操作；发文归属用该 userID（对齐 `postapp.Service.Create(authorID, ...)` 语义，`build/application/postapp/service.go:110-123`）。
3. 写操作后的**失效声明必须复用同包辅助函数**（`declarePostsChanged` 等），否则缓存不失效。
4. 应用服务签名一律复用，不改（见 §3 契约表引用的真实签名）。

---

## 2. 涉及文件清单

| 文件 | 动作 | 说明 |
| --- | --- | --- |
| `frame/go/build/interfaces/mcp.go` | 新增 | 网关全部实现：协议、中间件、dispatch、action handlers |
| `frame/go/build/interfaces/mcp_test.go` | 新增 | 中间件/分发/错误映射测试（假仓储） |
| `frame/go/build/register.go` | 修改 | 末尾接线 `interfaces.RegisterMCP(...)`（现第 145 行 `RegisterAdmin` 之后） |
| `frame/go/build/application/apikeyapp/service.go` | 只读 | Unit 1 提供（当前不存在，见 §4 标注） |

依赖的既有实现（只读，契约对齐依据）：

- 原生 fiber 先例：`build/interfaces/auth.go:23-66`
- hybrid ApiCtx 模式与 DTO/错误映射：`build/interfaces/apis.go`（`postInput` 14-21、`toServiceInput` 24-33、`currentUserID` 36-42、`writePostError` 101-111、`declarePostsChanged` 116-122）、`build/interfaces/dto.go`（`toPostView` 27-53）、`build/interfaces/moments.go`（`toMomentView` 25-32、`writeMomentError` 110-120）、`build/interfaces/interactions.go`（`toCommentView` 29-48）
- 评论按宿主失效写法：`build/interfaces/interactions.go:145-161`（approve）与 164-188（delete）
- 作者页/首页失效：`build/interfaces/authorAdmin.go:61-84`、`build/interfaces/settings.go:173-204`（改用户名）、233-257（改资料）
- 服务签名：`build/application/postapp/service.go`、`momentapp/service.go`、`commentapp/service.go`、`settingsapp/service.go`、`userapp/service.go`
- 组装根：`build/register.go`（`authorFn` 54-59、`authorNameFn` 61-67、服务构造 70-85、接口注册 88-145）
- 测试先例：`frame/go/hybrid/api_test.go:25`（`app.Server().App().Test(req)` + `httptest.NewRequest`）、`page_test.go:35`（`setupTestApp`）、`page_test.go:445`（`loginAs`）

---

## 3. 协议与错误码

### 3.1 传输

- `POST /api/mcp`，`Content-Type: application/json`，`Authorization: Bearer ven_xxx`（缺失视为无效 key，见 §4）。
- 请求体：`{"action": "<action>", "payload": {...}}`。`payload` 缺省视为 `{}`。
- 注：命名 JSON-RPC 风格，但**不是 JSON-RPC 2.0**（无 `jsonrpc`/`id` 字段——单端点无并发关联语义，不引入无谓字段）；action 名带命名空间点号（`post.create`），未遵循 jsonrpc 方法名分隔规范，仅为可读性。

### 3.2 响应

成功（HTTP 200/201 统一 200，下同）：

```json
{
  "ok": true,
  "data": {
    "id": "42"
  }
}
```

失败（HTTP 状态码与错误码对应）：

```json
{
  "error": {
    "code": "not_found",
    "message": "post not found"
  }
}
```

### 3.3 错误码表（协议级，与 HTTP 状态一一对应）

| code | HTTP | 触发场景 |
| --- | --- | --- |
| `invalid_key` | 401 | 缺 key / key 格式错 / AuthenticateKey 失败（含吊销） |
| `bad_request` | 400 | 请求体非法 JSON、缺 `action`、`action` 非字符串、未知 action、`payload` 不是对象 |
| `validation` | 400 | 应用层 `*ValidationError`（各 app 包）或契约级字段校验失败 |
| `not_found` | 404 | `post.ErrNotFound` / `moment.ErrNotFound` / `comment.ErrNotFound` / `user.ErrNotFound` |
| `forbidden` | 403 | 预留（本单元首批 action 均为 author 自有操作，正常不触发；保留给 Unit 3+ 的权限分支） |
| `internal` | 500 | 其余未分类错误 |

映射统一由 `mcpError` 结构承载（见 §5.2），dispatcher 负责写 HTTP 状态与协议体。

### 3.4 完整响应样例

请求：

```bash
curl -X POST http://127.0.0.1:8080/api/mcp \
  -H "Authorization: Bearer ven_live_xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"action":"post.create","payload":{"title":"你好，Agent","category":"随笔","content":"# 正文","summary":"摘要","tags":["agent"]}}'
```

成功响应：

```json
{
  "ok": true,
  "data": {
    "id": "42"
  }
}
```

错误响应（key 无效）：

```json
{
  "error": {
    "code": "invalid_key",
    "message": "invalid api key"
  }
}
```

响应头约定：401 时**不设置** `X-Ven-Login-Path`（该头仅 web cookie 鉴权链使用：`hybrid/page.go:22-23` 定义、`hybrid/api.go:50-53` 设置，agent 场景无效且会误导客户端去重定向）。

---

## 4. 鉴权中间件

### 4.1 流程（`mcp.go` 内 `mcpAuth` handler）

```
Parse Authorization header
  ├─ 缺失 / 非 "Bearer <token>" 两段式 → 401 invalid_key
  ├─ token 无 "ven_" 前缀 → 401 invalid_key（格式预检，快速拒绝）
  └─ keys.AuthenticateKey(token)
       ├─ err != nil（未知/吊销/内部错） → 401 invalid_key（不区分原因，不泄露 key 是否存在）
       └─ ok → ctx.Locals(mcpUserIDKey, userID) → 进入 action 分发
```

要点：

- **身份注入**：定义类型化 key 常量避免与 fiber 内建/其他中间件冲突：

  ```go
  type mcpCtxKey string
  const mcpUserIDKey mcpCtxKey = "mcp.userID" // ctx.Locals(mcpUserIDKey, int64)
  ```

  handler 内经 `mcpUserID(ctx)` 取出（断言失败视为 internal，理论不可达）。
- **不读 cookie**：全程不调用 `a.Server().CookieAuth/CurrentUser`（`internal/httpserver/server.go:217-227`）。带合法 cookie 但无 key 的请求同样 401 invalid_key——本入口与面板会话零耦合。
- **401 语义**：HTTP 401 + `{"error":{"code":"invalid_key","message":"invalid api key"}}`，**不设置** `X-Ven-Login-Path`。
- **与 Unit 1 的接口解耦**：`RegisterMCP` 接受最小接口而非具体类型（Unit 1 尚在并行，具体类型未落地）：

  ```go
  // KeyAuthenticator 是 API key 校验的最小契约（实现由 Unit 1 提供：
  // apikeyapp.Service.AuthenticateKey(rawKey string) (userID int64, err error)）。
  // 若 Unit 1 设计文档已落盘（docs/agent-design/unit-1-key-management.md），以其为准；当前按本签名设计。
  type KeyAuthenticator interface {
      AuthenticateKey(rawKey string) (int64, error)
  }
  ```

  实现侧 `*apikeyapp.Service` 天然满足该接口；测试注入假实现（见 §9）。
- **注入失败一律 invalid_key**：包括仓储内部错误（如 DB 抖动）——避免把鉴权内部细节泄露给 agent，且 key 不可用时快速失败优于降级放行。

### 4.2 请求体大小预检（放同一中间件，见 §7.2）

---

## 5. action 分发与文件布局

### 5.1 文件布局取舍结论：**并入 `interfaces` 包**（`build/interfaces/mcp.go`），不建子包

理由（对齐任务约束"考虑失效辅助函数在同包私有、错误类型复用"）：

1. **失效辅助函数是包私有**：`declarePostsChanged`（`apis.go:116-122`）、`authorNameFn`/`usernameOf` 均由组装根/接口层注入或定义。若拆 `interfaces/mcp/` 子包，只能二选一：复制失效逻辑（冒缓存失效不一致风险——正是基线明令禁止的），或导出这些辅助函数（无谓扩大接口层 API 面）。
2. **视图/错误映射直接复用**：`toPostView/toMomentView/toCommentView`、`postInput.toServiceInput()`、`writePostError/writeMomentError`、`mustID`（`pages.go:92`）全部同包私有，零复制。
3. **代价可控**：`interfaces` 包现有 19 个文件，追加 `mcp.go` + `mcp_test.go` 不改变包结构；包名/依赖方向（interfaces → application → domain，见 `AGENTS.md`「业务分层」）不变。

### 5.2 结构设计

```go
// mcp.go —— /api/mcp 网关（agent 统一入口，只认 key 不认 cookie）

// mcpRequest / mcpError 协议结构（§3）
type mcpRequest struct {
    Action  string          `json:"action"`
    Payload json.RawMessage `json:"payload"` // 缺省 "{}"（json.RawMessage 为空时归一）
}

type mcpError struct {
    HTTPStatus int    // HTTP 状态码
    Code       string // 协议错误码
    Message    string
}

// action 处理函数签名：payload 已归一为非空 JSON；返回 data（成功）或 *mcpError
type mcpActionFunc func(h *mcpHandlers, payload json.RawMessage) (any, *mcpError)

// MCP 处理器：组装根注入全部依赖（§5.3）
type MCP struct {
    a            *hybrid.App
    keys         KeyAuthenticator
    posts        *postapp.Service
    moments      *momentapp.Service
    comments     *commentapp.Service
    settings     *settingsapp.Service
    users        *userapp.Service
    authorFn     func() (*user.User, error)
    authorNameFn func() string
}

// dispatch 表：action 名 → 处理函数（注册期构建，运行期只读，天然并发安全）
var mcpActions = map[string]mcpActionFunc{
    "post.create":        (*MCP).postCreate,
    "post.update":        (*MCP).postUpdate,
    "post.delete":        (*MCP).postDelete,
    "post.list":          (*MCP).postList,
    "moment.create":      (*MCP).momentCreate,
    "moment.delete":      (*MCP).momentDelete,
    "moment.list":        (*MCP).momentList,
    "comment.list_pending": (*MCP).commentListPending,
    "comment.approve":    (*MCP).commentApprove,
    "comment.reject":     (*MCP).commentReject,  // 实现依赖 Unit 3
    "comment.recover":    (*MCP).commentRecover, // 实现依赖 Unit 3
    "comment.list":       (*MCP).commentList,
    "author.get":         (*MCP).authorGet,
    "author.update":      (*MCP).authorUpdate,
}
```

主流程（`POST /api/mcp` handler）：

```
mcpAuth 中间件（§4）→
  解析 mcpRequest（非法 JSON / 缺 action / action 非字符串 → bad_request）
  h, ok := mcpActions[action]；!ok → bad_request（message: "unknown action: <action>"）
  data, merr := h(mcp, payload)
  merr != nil → ctx.Status(merr.HTTPStatus).JSON({"error":{code,message}})
  否则      → ctx.JSON({"ok":true,"data":data})
```

### 5.3 注册函数与接线

```go
// mcp.go
// RegisterMCP 注册 /api/mcp 网关（原生 fiber 路由，不走 hybrid cookie 鉴权链）。
func RegisterMCP(
    a *hybrid.App,
    keys KeyAuthenticator,
    posts *postapp.Service,
    moments *momentapp.Service,
    comments *commentapp.Service,
    settings *settingsapp.Service,
    users *userapp.Service,
    authorFn func() (*user.User, error),
    authorNameFn func() string,
) error {
    // 注：auth.go:28-29 先例 —— server := a.Server(); server.App().Post(path, ...)
    // /api 前缀对原生 fiber 路由不生效，必须写全路径（含 /api）
    m := &MCP{a: a, keys: keys, ...}
    a.Server().App().Post("/api/mcp", m.handle)
    return nil
}
```

`register.go` 接线位置：现第 145 行 `return interfaces.RegisterAdmin(...)` 之后（`RegisterAdmin` 目前是链尾；MCP 是纯 fiber 路由，与页面注册顺序无关，放链尾最稳）：

```go
// register.go（第 145 行后追加）
if err := interfaces.RegisterAdmin(a, posts, comments, interactions, moments, subscribe, users, settings); err != nil {
    return err
}
return interfaces.RegisterMCP(a, keys, posts, moments, comments, settings, users, authorFn, authorNameFn)
```

其中 `keys` 由 Unit 1 在 `register.go` 构造（本单元只在注册处占位并标注：`keys := apikeyapp.NewService(keyRepo)`——待 Unit 1 定稿替换）。`authorFn`/`authorNameFn` 复用现有 54-67 行闭包，**不新建**。

---

## 6. action 契约表（完整）

通用约定：

- 所有 ID 入参接受字符串（与面板 API 一致，`mustID` 归一；非法 ID 归一 0 → not_found）。
- 成功 `data` 内 ID 一律字符串（与 `toPostView`/`toMomentView`/`toCommentView` 既有契约一致，`dto.go:12-24`、`moments.go:16-22`、`interactions.go:17-26`）。
- 错误映射行标注 "复用" 的指直接复用现有 `writePostError`（`apis.go:101-111`）/`writeMomentError`（`moments.go:110-120`）的判定分支，但**响应体改协议格式**（`mcpError` 而非 `c.Error`）。
- 所有写 action 均为 author 语义：key 校验出的 userID 即发文归属。

### 6.1 post.create

| 项 | 内容 |
| --- | --- |
| payload | `{title, category, content, summary?, coverUrl?, tags?}` —— 字段对齐 `postInput`（`apis.go:14-21`） |
| 服务调用 | `posts.Create(userID, in.toServiceInput())`（`postapp/service.go:110-123`） |
| 成功 data | `{"id": "<p.ID>"}`（HTTP 201 → 协议统一 200 外层；沿用面板返回 id 语义 `apis.go:70`） |
| 错误映射 | `*postapp.ValidationError` → validation（message 直取）；其余 → internal（复用 `writePostError` 分支） |
| 失效声明 | `declarePostsChanged(a, p.ID)`（`apis.go:116-122`：`InvalidatePage("/posts")` + `InvalidatePage("/")` + `DataChange("/posts/:id", id)`） |

### 6.2 post.update

| 项 | 内容 |
| --- | --- |
| payload | `{id, title, category, content, summary?, coverUrl?, tags?}` |
| 服务调用 | `posts.Update(mustID(payload.id), in.toServiceInput())`（`postapp/service.go:126-139`；封面留空兜底首图由服务层保证） |
| 成功 data | `{"post": toPostView(p)}`（与面板 `apis.go:85` 一致） |
| 错误映射 | ValidationError → validation；`post.ErrNotFound` → not_found；其余 → internal |
| 失效声明 | `declarePostsChanged(a, id)`（Update 返回实体，用 `p.ID`） |

### 6.3 post.delete

| 项 | 内容 |
| --- | --- |
| payload | `{id}` |
| 服务调用 | `posts.Delete(mustID(payload.id))`（`postapp/service.go:142-144`） |
| 成功 data | `{"deleted": true}` |
| 错误映射 | `post.ErrNotFound` → not_found（仓储 `Delete` 语义）；其余 → internal |
| 失效声明 | `declarePostsChanged(a, id)`（对齐面板删除分支 `apis.go:90-97`） |

### 6.4 post.list

| 项 | 内容 |
| --- | --- |
| payload | `{limit?|category?, page?, pageSize?}` —— 二选一：只给 `limit` 走 `ListRecent(limit)`（limit<=0 全部）；给 `category`/`page`/`pageSize` 走 `posts.List(ListFilter{...})`（`postapp/service.go:57-71`，PageSize<=0 默认 10） |
| 服务调用 | `posts.ListRecent(limit)` 或 `posts.List(filter)` |
| 成功 data | 前者 `{"posts": toPostViews(list)}`；后者 `{"posts": toPostViews(paged.Posts), "total": paged.Total, "page": ..., "pageSize": ...}` |
| 错误映射 | internal |
| 失效声明 | 无（只读） |

### 6.5 moment.create

| 项 | 内容 |
| --- | --- |
| payload | `{content}`（对齐 `momentInput`，`moments.go:43-46`） |
| 服务调用 | `moments.Create(userID, content)`（`momentapp/service.go:28-38`；领域校验：非空、<=1000 字符 `domain/moment/moment.go:23-33`） |
| 成功 data | `{"id": "<m.ID>"}`（对齐 `moments.go:94`） |
| 错误映射 | `*momentapp.ValidationError` → validation；其余 → internal（复用 `writeMomentError` 分支） |
| 失效声明 | `_ = a.DataChange("/moments")`（`moments.go:93`） |

### 6.6 moment.delete

| 项 | 内容 |
| --- | --- |
| payload | `{id}` |
| 服务调用 | `moments.Delete(mustID(payload.id))`（`momentapp/service.go:41-43`） |
| 成功 data | `{"deleted": true}` |
| 错误映射 | `moment.ErrNotFound` → not_found；其余 → internal |
| 失效声明 | `_ = a.DataChange("/moments")`（`moments.go:104`） |

### 6.7 moment.list

| 项 | 内容 |
| --- | --- |
| payload | `{}` |
| 服务调用 | `moments.List()`（最多 50 条，`momentapp/service.go:22-25`） |
| 成功 data | `{"moments": toMomentViews(list)}`（对齐 `/moments` 页语义 `moments.go:53-77`） |
| 错误映射 | internal |
| 失效声明 | 无 |

### 6.8 comment.list_pending

| 项 | 内容 |
| --- | --- |
| payload | `{}` |
| 服务调用 | `comments.ListPending()`（`commentapp/service.go:47-49`；创建时间正序） |
| 成功 data | `{"comments": toCommentViews(list)}` |
| 错误映射 | internal |
| 失效声明 | 无 |

### 6.9 comment.approve

| 项 | 内容 |
| --- | --- |
| payload | `{id}` |
| 服务调用 | `comments.Approve(mustID(payload.id))`（`commentapp/service.go:35-44`，返回 `comment.Target{PostID, MomentID}`） |
| 成功 data | `{"id": "<id>", "status": "approved"}` |
| 错误映射 | `comment.ErrNotFound` → not_found；其余 → internal（对齐 `interactions.go:146-151`） |
| 失效声明 | **按宿主**（对齐 `interactions.go:153-157` approve 分支）：`target.MomentID > 0` → `_ = a.DataChange("/moments")`；否则 `declarePostsChanged(a, target.PostID)` |

### 6.10 comment.reject —— 契约定稿，实现依赖 Unit 3

| 项 | 内容 |
| --- | --- |
| payload | `{id}` |
| 服务调用 | `comments.Reject(mustID(payload.id))` —— **签名由 Unit 3 提供**（预期 `(comment.Target, error)`，内部 `SetStatus(id, comment.StatusRejected)`，`commentapp/service.go:40-43` 已有 `SetStatus` 先例；`StatusRejected = "rejected"` 常量由 Unit 3 加入 `domain/comment/comment.go:10-13`） |
| 成功 data | `{"id": "<id>", "status": "rejected"}` |
| 错误映射 | `comment.ErrNotFound` → not_found；对非 pending 评论拒绝（若 Unit 3 定义该规则）→ validation；其余 → internal |
| 失效声明 | **按宿主**（与 approve 同一分支函数——抽私有 `invalidateCommentHost(h, target)`，approve/reject/recover/delete 四路共用）：`MomentID > 0` → `DataChange("/moments")`；否则 `declarePostsChanged(a, PostID)`。理由：approved → rejected 是可见性变化，宿主页必须失效 |
| 实现标注 | 本单元**只落契约与 dispatch 接线**，handler 体标 `// TODO(unit3): commentapp.Reject` 并返回 internal 兜底；随 Unit 3 合并后启用 |

### 6.11 comment.recover —— 契约定稿，实现依赖 Unit 3

| 项 | 内容 |
| --- | --- |
| payload | `{id}` |
| 服务调用 | `comments.Recover(mustID(payload.id))` —— **签名由 Unit 3 提供**（预期 `(comment.Target, error)`：rejected → 回到 pending 重新进审核流；若 Unit 3 定为直接 approved 以本表为准改为契约修订点） |
| 成功 data | `{"id": "<id>", "status": "pending"}` |
| 错误映射 | `comment.ErrNotFound` → not_found；对非 rejected 评论恢复（若 Unit 3 定义该规则）→ validation；其余 → internal |
| 失效声明 | 同 6.10 按宿主分支（rejected → pending 不可见性不变，严格说无需失效；统一走宿主分支成本为零，避免两套逻辑） |
| 实现标注 | 同 6.10：契约定稿，实现挂 `// TODO(unit3)` |

### 6.12 comment.list

| 项 | 内容 |
| --- | --- |
| payload | `{limit?}`（缺省 100，对齐后台 `ListAll(100)` `admin.go:226`） |
| 服务调用 | `comments.ListAll(mustID(limit))`（`commentapp/service.go:52-54`；创建时间倒序，含用户名与所属文章标题） |
| 成功 data | `{"comments": toCommentViews(list)}` |
| 错误映射 | internal |
| 失效声明 | 无 |
| 备注 | 不承诺 `status` 筛选字段（Unit 3 引入 rejected 后可扩展，本单元不加） |

### 6.13 author.get

| 项 | 内容 |
| --- | --- |
| payload | `{}` |
| 服务调用 | `settings.Content()`（`settingsapp/service.go:150-169`，含缺省回退）+ `authorFn()`（`register.go:54-59`，现取 author，随改名生效） |
| 成功 data | `{"content": {"paragraphs": [...], "skills": [...], "friends": [...], "quotes": [...], "projects": [...], "github": "..."}, "profile": {"username": "...", "bio": "...", "avatarUrl": "...", "role": "author", "email": "..."}}` —— content 结构与 `settingsapp.Content`（`settingsapp/service.go:37-45`）序列化一致；profile 对齐 `toUserView`（`profiles.go:18-27`）+ email（作者本人可见，面板口径 `settings.go:44-51`） |
| 错误映射 | `user.ErrNotFound` → not_found（理论不可达，author 必须存在）；其余 → internal |
| 失效声明 | 无 |

### 6.14 author.update —— 对齐 `PUT /api/admin/author/content` 语义，扩展资料字段

| 项 | 内容 |
| --- | --- |
| payload | 全字段可选（**部分更新**：只写传入的字段，与面板"整包覆盖"不同——面板整包写是页面交互使然，agent 部分更新更安全）：`{paragraphs?, skills?, friends?, projects?, quotes?, showcasePosts?, bio?, avatarUrl?, username?}` |
| 服务调用 | 按字段分派（复用既有服务方法）：`settings.SetParagraphs/SetSkills/SetFriends/SetProjects/SetShowcasePosts`（authorAdmin.go:66-79 对应项）、`settings.SetQuotes`（settings.go:76 对应项）、`users.UpdateProfile(userID, bio, avatarURL)`（`userapp/service.go:116-121`，bio<=200）、`users.UpdateUsername(userID, username)`（`userapp/service.go:124-129`，2-32 字符；改名前**先取旧用户名**再改，对齐 settings.go:184-188） |
| 成功 data | `{"username": "<新用户名>"}`（仅当本次改了用户名，否则 `{"updated": true}`） |
| 错误映射 | `*userapp.ValidationError` → validation（message 直取）；`user.ErrUsernameTaken` → validation（协议表无 conflict，语义归入 validation，message "username taken"）；其余 → internal |
| 失效声明 | 分字段聚合：content 类（paragraphs/skills/friends/projects/quotes/showcasePosts）→ `a.InvalidatePage("/author/"+authorNameFn())` + `a.InvalidatePage("/")`（对齐 authorAdmin.go:81-82；quotes 另在 settings.go:79 只失效 `/`，聚合进同一组即可）；profile（bio/avatarUrl）→ 同组作者页 + 首页（对齐 settings.go:254-255）；username → 首页 + 旧路径 + 新路径（对齐 settings.go:198-200，旧用户名取自改名前） |

---

## 7. 安全设计

1. **只认 key**：中间件不读 `ven_auth` cookie、不调 `CookieAuth/CurrentUser`；cookie 带不带都无所谓，缺 key 一律 401 invalid_key。
2. **不暴露登录重定向**：401 响应不设 `X-Ven-Login-Path`；本端点不产生任何重定向响应。
3. **key 不泄露**：响应体不回显 key；访问日志（`internal/httpserver/logging.go:13-25`）只记 method/path/status/耗时，不记请求头——保持现状即可，**不要**在 mcp handler 里新增含 Authorization 的日志。
4. **鉴权失败不区分原因**：未知 key/吊销 key/格式错/后端错统一 `invalid_key`，不泄露 key 是否存在。
5. **请求体大小限制**：
   - 全局 fiber `BodyLimit` 为 10MB（`internal/httpserver/server.go:72`），作为硬上限兜底。
   - **关键事实**：全局 `ErrorHandler`（server.go:74-79）把一切 handler 错误统一写成 500——fiber 触发的 413（超 BodyLimit）也会被吞成 internal。因此中间件内**显式预检** `ctx.Request().Header.ContentLength()`：> 1MB 直接返回 413 + `{"error":{"code":"bad_request","message":"payload too large"}}`（协议表无 413 专属码，归 bad_request；HTTP 状态用 413 更语义化——二选一需评审，默认 HTTP 413 + code bad_request）。1MB 对 markdown 正文绰绰有余（字段上限见 `domain/post/post.go:11-16`、`domain/comment/comment.go:42-50`）。
6. **幂等性**：写操作**非幂等**（post.create/moment.create 每次落新行），本单元不引入 idempotency key；agent 的失败重试策略（超时后重试、重复创建的去重/对账）属 Unit 5 skill 设计约束，本单元只保证：成功响应即已提交、错误响应不承诺未生效。给 agent 的补救建议：`post.list`/`author.get` 读回对账。
7. **越权面**：首批 action 全部落在 author 自有资源（文章/动态/评论审核/个人页），key 持有者即 author，无横向越权模型；`forbidden` 码保留给未来（如评论删除的非 author 分支、Unit 3+ 的多 key 角色）。

---

## 8. 验收标准（独立可验收）

1. 无 cookie、无 key 调 `POST /api/mcp` → HTTP 401，`{"error":{"code":"invalid_key",...}}`，响应头**无** `X-Ven-Login-Path`。
2. 携带面板会话 cookie 但无 key → 同样 401 invalid_key（证明与 cookie 鉴权隔离）。
3. 格式错 key（非 `Bearer ven_xxx`）与吊销 key → 同样 401 invalid_key，响应体完全一致（不区分）。
4. 未知 action（如 `"foo"`）→ HTTP 400，`code: bad_request`。
5. 非法 JSON 请求体 / 缺 `action` → HTTP 400，`code: bad_request`。
6. 表 6.1-6.14 每个 action 用有效 key 走通 happy path，成功响应均为 `{"ok": true, "data": {...}}` 且 ID 为字符串。
7. 写 action 后缓存失效生效（验收方法：创建文章后访问 `/posts` 与文章详情页，内容即时可见；发布动态后 `/moments` 即时可见；审核评论后宿主页即时可见——用浏览器无痕窗口验证，或观察 SSE 推送）。
8. `post.create` 以 key userID 归属：`post.list` 返回的 `authorName` 为 author 本人。
9. 超 1MB 请求体 → HTTP 413 + `code: bad_request`。
10. `comment.reject`/`comment.recover` 在 Unit 3 合并前返回 internal 兜底且不 panic；Unit 3 合并后按契约工作。

## 9. 测试计划

### 9.1 先例调研结论

- **fiber `app.Test()` 先例存在**：`frame/go/hybrid/api_test.go:25`（`app.Server().App().Test(req)`，配合 `httptest.NewRequest`）、`:73,82-83,113-114,157` 多处；辅助 `setupTestApp`（`page_test.go:35`）、`loginAs`（`page_test.go:445`）。
- **build 包目前零测试**（`frame/go/build` 下无 `*_test.go`），且业务仓储是 MySQL 实现（`build/infrastructure/persistence/`）——**不能**直接起真实 DB。因此 `mcp_test.go` 需要：
  - 在 `build/interfaces` 包内新建假仓储（内存 map 实现 `post.Repository`/`moment.Repository`/`comment.Repository`/`setting.Repository`/`user.Repository`，接口见 `domain/*/repository.go`）——本包测试专用，不进生产代码。
  - 假 `KeyAuthenticator`（返回固定 userID 或错误）。
  - 复用 hybrid 测试的服务器构造方式：`httpserver.New(config.Config{...}, fakeSSRClient, ...)` + `hybrid.New(server)` + `RegisterMCP(...)`（照搬 `page_test.go:35` 的构造手法；SSR client 用最小假实现即可——MCP 路由不触发 SSR）。

### 9.2 中间件测试用例

| # | 场景 | 期望 |
| --- | --- | --- |
| M1 | 无 Authorization 头 | 401 invalid_key，无 X-Ven-Login-Path |
| M2 | `Authorization: Basic xxx`（非 Bearer） | 401 invalid_key |
| M3 | `Bearer foo`（无 `ven_` 前缀） | 401 invalid_key（不调 AuthenticateKey） |
| M4 | `Bearer ven_xxx` 但 AuthenticateKey 返回错误 | 401 invalid_key |
| M5 | AuthenticateKey 返回 (0, nil)（理论上不可能） | internal（防御分支） |
| M6 | 带合法 cookie 无 key | 401 invalid_key（隔离性） |
| M7 | 请求体 > 1MB（Content-Length 预检） | 413 + bad_request |
| M8 | 有效 key + 合法 action | 通过，handler 收到注入的 userID |

### 9.3 分发与协议测试用例

| # | 场景 | 期望 |
| --- | --- | --- |
| D1 | 非法 JSON 体 | 400 bad_request |
| D2 | 缺 `action` / action 非字符串 | 400 bad_request |
| D3 | 未知 action | 400 bad_request，message 含 action 名 |
| D4 | `payload` 非法 JSON | 400 bad_request |
| D5 | 成功响应结构 | `{"ok":true,"data":...}`，ID 字段为字符串 |
| D6 | 错误响应结构 | `{"error":{"code","message"}}`，HTTP 状态与码表一致 |

### 9.4 每个 action 的 happy path + 错误映射（假仓储断言）

| action | happy path 断言 | 错误映射用例 |
| --- | --- | --- |
| post.create | 假 repo 收到 authorID=key userID；返回 id | ValidationError → validation；repo 错误 → internal |
| post.update | repo.Update 被调用；返回更新后视图 | ValidationError → validation；ErrNotFound → not_found |
| post.delete | repo.Delete(id) | ErrNotFound → not_found |
| post.list | limit 分支 / 分页分支数据正确 | repo 错误 → internal |
| moment.create | 归属 key userID；id 返回 | ValidationError → validation |
| moment.delete | repo.Delete | ErrNotFound → not_found |
| moment.list | 列表结构与页面一致 | internal |
| comment.list_pending | 仅 pending 列表 | internal |
| comment.approve | repo.SetStatus(approved) 被调；**宿主分支**：Post 宿主断言 declarePostsChanged 路径不报错、Moment 宿主断言 DataChange 路径不报错（失效断言见 9.5） | ErrNotFound → not_found |
| comment.reject / recover | Unit 3 前：返回 internal 兜底不 panic；Unit 3 后补 happy path | ErrNotFound → not_found |
| comment.list | limit 生效（默认 100） | internal |
| author.get | content 回退默认 + profile 字段 | authorFn 错误 → internal |
| author.update | 部分更新只调传入字段对应服务；username 分支校验冲突 | ValidationError / ErrUsernameTaken → validation |

### 9.5 失效声明的验证策略（明确边界）

- `declarePostsChanged`/`DataChange` 的缓存/SSR 行为已由 hybrid 层测试覆盖（`staticPage_test.go`、`sse_test.go`），本单元**不在单测中断言失效副作用**（无观测钩子，强行断言需构造完整 page cache，成本高收益低）。
- 改为两层保障：① code review 清单——每个写 action 对照 §6 契约表逐条核对失效声明与宿主分支（approve 分支对齐 `interactions.go:153-157`）；② §8 验收第 7 条的手动冒烟。
- 可选项（低成本）：在假 repo 上断言"handler 执行后 `a.InvalidatePage` 被调用"需 mock hybrid.App——hybrid.App 为具体结构体不可 mock，放弃，维持上述策略。

---

## 10. 风险与开放问题

1. **Unit 1 未落盘**（`docs/agent-design/unit-1-key-management.md` 不存在，`docs/agent-design/` 目录尚不存在，本文件是首个）：`KeyAuthenticator` 接口按约定签名设计；若 Unit 1 签名变化（如返回 (userID, revokedAt, err)），只需改 `register.go` 接线与中间件一处调用，接口层解耦已兜住。
2. **Unit 3 依赖面**：`comment.reject/recover` 的最终语义（reject 是否只允许 pending；recover 回 pending 还是 approved）以 Unit 3 为准，本文件 6.10/6.11 已标注修订点；`StatusRejected` 常量命名也可能调整。
3. **全局 ErrorHandler 吞 413**：fiber 超 BodyLimit 的错误会被 `server.go:74-79` 统一写成 500。本设计用 Content-Length 预检规避；若未来出现 chunked 无长度请求体绕过预检，会落到 500 internal——可接受（agent 可重试），但应在 Unit 5 skill 重试策略中把 500 视为可重试。
4. **`forbidden` 码暂无触发路径**：首批 action 无越权模型，码表保留但无测试覆盖；待 Unit 3+ 引入权限分支时补。
5. **author.update 部分更新 vs 面板整包覆盖**：两种语义并存（面板 PUT `/api/admin/author/content` 整包写，`authorAdmin.go:61-84`；agent 部分写）。风险：agent 只改 bio 不动 paragraphs 时不会误清空——这正是部分更新要防的；但若 agent 误传空数组会清空对应区块（与面板一致），文档在 Unit 5 skill 中需提醒"未打算改的字段不要传"。
6. **请求体上限 1MB 的取值**：远大于任何单文章体量（字段上限 `domain/post/post.go:11-16`）；如未来支持 base64 图片直传需重估（当前图片走 `/api/upload`，`interfaces/upload.go`，不在本单元范围）。
7. **与 `/api` 前缀冲突面**：原生 fiber 注册 `POST /api/mcp` 与 hybrid `apiRoute`（`hybrid/authz.go:31-43`）不冲突（hybrid 只注册它自己的模式，且页面注册禁止 `/api` 前缀 `hybrid/authz.go:45-51`）；但**禁止**再通过 `a.Post("/mcp", ...)` 注册同名路由（会双重注册 panic 或歧义），代码评审注意。
8. **日志脱敏**：`requestLogger`（`logging.go:13-25`）不记头，现状安全；后续若有人给 mcp 加调试日志，禁止打印 `Authorization`。
9. **无 rate limiting**：本单元不含频率限制（面板也未有）；滥用防护（如 key 粒度的限流）列入 Unit 1/Unit 5 讨论，不在本单元扩权。
