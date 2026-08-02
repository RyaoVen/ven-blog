# Unit 3 详细设计：UGC（评论 + 留言板）审核语义

> 目标版本基线：本文所有文件路径相对仓库根 `C:\Users\25108\GolandProjects\awesomeProject1`。
> Go 业务代码位于 `frame/go/build/`（DDD 四层：domain / application / infrastructure / interfaces），组装根 `frame/go/build/register.go`；前端位于 `src/`（Node SSR）。
> 配套单元：Unit 1（密钥管理，占 migration 008）、Unit 2（/api/mcp 网关，消费本单元的 comment.reject/recover 服务语义）、Unit 4（自动审核 worker，调用本单元的 AutoReview 位置）。

## 1. 目标

把评论与留言板升级为"可审核的 UGC"：

1. **评论**：现有 approved/pending 两态扩展为三态（+ rejected），新增驳回原因（≤200），面板支持驳回/恢复（AI 误杀恢复），服务层语义供 Unit 2 MCP action 与 Unit 4 worker 复用。
2. **留言板**：从"无状态、直接放行"升级为与评论同构的三态审核；公开查询（`GET /api/guestbook` 与 `/author/:name` 页）只返回 approved —— 这是本单元唯一动现有公开行为的点，验收必须覆盖。
3. **面板**：评论管理页加 rejected 视图与恢复按钮；新增留言板管理页（独立页，结论见 §6.2）。
4. **铺路**：定好 AutoReview 用例位置（本单元不实现 worker）与驳回邮件的触发点（本单元不发邮件，只留接口层读取通道）。

**不做**：不做完整状态机与状态历史表；面板人工操作不发邮件；不实现自动审核 worker；不改评论公开查询（已只返回 approved，见 §8）。

## 2. 涉及文件清单

### Go 业务代码（frame/go/build/）

| 文件 | 改动 |
|---|---|
| `domain/comment/comment.go` | +StatusRejected 常量、+RejectedReason 字段、+ValidateRejectedReason |
| `domain/comment/repository.go` | +ErrInvalidState、+ListRejected、+SetRejected、SetStatus 语义变更（清 reason） |
| `application/commentapp/service.go` | +Reject / +Recover / +ListRejected / +Get（邮件触发点） |
| `infrastructure/persistence/comment_repository.go` | scanComment/commentSelect 加 rejected_reason；+ListRejected、+SetRejected；SetStatus 清 reason；MomentCommentCounts 过滤 approved（建议，§8.7） |
| `domain/guestbook/guestbook.go` | +三态常量、+Status/RejectedReason 字段、+ValidateRejectedReason |
| `domain/guestbook/repository.go` | +ErrInvalidState、List 语义变更（仅 approved）、+ListAll/ListPending/ListRejected/SetStatus/SetRejected |
| `application/guestbookapp/service.go` | NewService 签名变更（+moderation func() bool）；Create 定初始状态；+Approve/Reject/Recover/ListPending/ListRejected/ListAll |
| `infrastructure/persistence/guestbook_repository.go` | List 加 status 过滤；+ListAll/ListPending/ListRejected/SetStatus/SetRejected；Get/Create 带 status/rejected_reason |
| `interfaces/interactions.go` | 评论 approve 旁 +POST /comments/:id/reject、/recover |
| `interfaces/admin.go` | adminCommentView +RejectedReason；/admin/comments 页 +rejected 列表 |
| `interfaces/guestbook.go` | GuestbookView +Status；+RegisterGuestbookAdmin（管理页 + approve/reject/recover API） |
| `infrastructure/persistence/migrations/009_ugc_status.sql` | **新建**（见 §3） |
| `infrastructure/persistence/db.go` | +go:embed migration009 并追加进 migrations 切片（注意与 Unit 1 的 008 保持数字序） |
| `register.go` | :85 guestbook NewService 注入 moderation 开关；接线 RegisterGuestbookAdmin |

### 前端（src/）

| 文件 | 改动 |
|---|---|
| `admin/comments/page.tsx` | +rejected 区块（含驳回原因展示、恢复按钮）；FilterSelect +"已驳回"；approve/reject/recover 后本地 state 迁移 |
| `admin/types.ts` | AdminComment +rejectedReason；AdminCommentsState +rejected |
| `admin/guestbook/page.tsx` | **新建**：pending / rejected / 全量三区块 + approve/reject/recover/delete |
| `admin/adminLayout.tsx` | TABS 增加 `{ href: "/admin/guestbook", label: "留言" }`（:10-17） |
| `author/[name]/page.tsx` | GuestbookSection 发表响应 status==="pending" 时提示"待作者审核通过后展示"（对齐评论 :82-85 模式） |
| `author/types.ts` | GuestbookEntry +status |
| `admin/settings/page.tsx` | :718 文案"所有新评论"→"所有新评论与留言"（审核开关现同时管留言） |

## 3. 数据模型（migration 009_ugc_status.sql）

### 3.1 现状确认（已读代码）

- `comments.status` 已由 `frame/go/build/infrastructure/persistence/migrations/005_settings.sql:9-36` 提供：`VARCHAR(16) NOT NULL DEFAULT 'approved'` + 索引 `idx_comments_status`。**VARCHAR 无枚举约束，写入 'rejected' 无需改列**。
- `comments` 无 rejected_reason 列；`guestbook`（004_guestbook.sql）无 status / rejected_reason 列。
- 迁移执行方式（`infrastructure/persistence/db.go:16-38,70-73`）：`//go:embed` + 启动时按序全量执行，脚本必须幂等。幂等条件 ALTER 风格以 `003_comments_reply_to.sql:4-16` / `005_settings.sql:9-36` 为准（information_schema 探测 + PREPARE/EXECUTE）。

### 3.2 完整 DDL

```sql
-- 009：UGC 审核三态（approved | pending | rejected）+ 驳回原因（评论与留言板）
-- 幂等条件 ALTER，对齐 003/005/006/007 风格；
-- comments.status 已由 005 提供（VARCHAR(16) 无枚举约束），rejected 值无需改列。

-- 1) guestbook.status：存量行经 DEFAULT 'approved' 自动回填为已通过
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND COLUMN_NAME = 'status'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE guestbook ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT ''approved'' AFTER content',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2) guestbook.rejected_reason
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND COLUMN_NAME = 'rejected_reason'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE guestbook ADD COLUMN rejected_reason VARCHAR(200) NOT NULL DEFAULT '''' AFTER status',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 3) guestbook.status 索引（公开列表与面板按状态过滤）
SET @idx := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND INDEX_NAME = 'idx_guestbook_status'
);
SET @sql := IF(
    @idx = 0,
    'ALTER TABLE guestbook ADD INDEX idx_guestbook_status (status)',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 4) comments.rejected_reason（rejected 状态值写入现有 status 列即可）
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'rejected_reason'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE comments ADD COLUMN rejected_reason VARCHAR(200) NOT NULL DEFAULT '''' AFTER status',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
```

要点：
- **存量回填**：guestbook 存量行在 ADD COLUMN 时经 `DEFAULT 'approved'` 自动回填为已通过（无需单独 UPDATE）；comments 存量行在 005 已默认 approved。
- **db.go 登记**：新增 `//go:embed migrations/009_ugc_status.sql` + `var migration009 string`，并把 `migration009` **追加到 migrations 切片末尾**（`db.go:37-38`）。与 Unit 1 的 008 合并时保持 008 在前 009 在后（数字序）。

## 4. 领域规则

### 4.1 评论（domain/comment/comment.go）

常量与字段：

```go
const (
    StatusApproved = "approved"
    StatusPending  = "pending"
    StatusRejected = "rejected" // 新增
)

type Comment struct {
    // ...现有字段（comment.go:16-27）
    RejectedReason string // 新增：≤200；仅 Status==rejected 时非空；其余状态由仓储保证清空
}

// 新增领域校验：去空白后非空且 ≤200 字符（错误消息空串表示通过）
func ValidateRejectedReason(reason string) string
```

状态迁移模型（简单迁移模型，不做重状态机）：

| 从 \ 到 | approved | pending | rejected |
|---|---|---|---|
| approved | 幂等（现有 Approve 行为） | — | 允许（记 reason） |
| pending | approve 允许 | — | 允许（记 reason） |
| rejected | **recover 允许**（reason 清空）；Approve 亦允许（等价 recover） | — | 允许（覆盖 reason） |

- 任意状态可 reject（必须记 reason）——理由：Unit 2 的 MCP action 可能对已通过评论做"违规复核驳回"，不该被状态机挡在门外。
- rejected→approved 即 recover，不引入独立"召回"语义——理由：恢复只有一种意图（误杀纠偏），人（面板）与机（Unit 4 恢复策略）共用同一用例。
- 不做状态机的原因：现状是 VARCHAR 状态 + 单条 UPDATE（`comment_repository.go:198-204` 的 SetStatus），无任何状态机设施；单人博客引入迁移历史表/状态机收益为零。规则集中在应用层 Reject/Recover 两个方法 + 一个领域校验函数即可完整表达，用 switch 显式列出即可。

新增领域错误（domain/comment/repository.go:6-10 旁）：

```go
var ErrInvalidState = errors.New("comment not in rejected state") // Recover 前置校验用
```

### 4.2 留言板（domain/guestbook/guestbook.go）

与评论完全同构：

```go
const (
    StatusApproved = "approved"
    StatusPending  = "pending"
    StatusRejected = "rejected"
)

type Entry struct {
    // ...现有字段（guestbook.go:10-16）
    Status         string // approved | pending | rejected
    RejectedReason string // 仅 rejected 非空
}

func ValidateRejectedReason(reason string) string // 同 4.1
```

迁移模型同 4.1 的表。新增 `var ErrInvalidState = errors.New("guestbook entry not in rejected state")`（domain/guestbook/repository.go:6-9 旁）。

## 5. 应用层用例契约（Go 签名级）

### 5.1 commentapp（application/commentapp/service.go）

现有 `Approve(commentID int64) (comment.Target, error)`（:35-44，先 Get 再 SetStatus）**保持不变**——对 rejected 调用 Approve 即等价 recover（SetStatus 会清 reason，见下）。

仓储接口变更（domain/comment/repository.go:13-34）：

```go
type Repository interface {
    // ...现有方法不变
    ListRejected() ([]*Comment, error)          // 新增：创建时间正序（对齐 ListPending 的"先到先处理"）
    SetRejected(id int64, reason string) error  // 新增：UPDATE comments SET status='rejected', rejected_reason=? WHERE id=?
    // SetStatus 语义变更：UPDATE comments SET status=?, rejected_reason='' WHERE id=?
    // —— 任何"非 rejected"写入都清空 reason，保持"仅 rejected 有 reason"不变量
}
```

新用例（均返回宿主供失效声明，对齐 Approve 的返回风格）：

```go
// Reject 驳回评论：reason 必填且 ≤200（ValidationError→400），任意状态可驳回；返回宿主。
func (s *Service) Reject(commentID int64, reason string) (comment.Target, error)
// 实现：ValidateRejectedReason → repo.Get（ErrNotFound 透传）→ repo.SetRejected → Target

// Recover 恢复被驳回的评论（AI 误杀恢复）：仅 rejected→approved（reason 随 SetStatus 清空）；其余状态返回 ErrInvalidState。
func (s *Service) Recover(commentID int64) (comment.Target, error)

// ListRejected 被驳回评论（面板 rejected 视图）。
func (s *Service) ListRejected() ([]*comment.Comment, error)

// Get 按 ID 取评论（薄封装 repo.Get）——驳回邮件触发点的读取通道（§7.2）。
func (s *Service) Get(commentID int64) (*comment.Comment, error)
```

### 5.2 guestbookapp（application/guestbookapp/service.go）

构造签名变更（对齐 commentapp.NewService 的开关注入模式，register.go:78-81）：

```go
// NewService 构造留言板用例服务；moderation 为审核开关解析函数（组装根注入，随设置实时生效）。
func NewService(repo guestbook.Repository, moderation func() bool) *Service
```

`Create`（:24-33 改造）初始状态逻辑对齐 commentapp.Create（service.go:72-75）：

```go
status := guestbook.StatusApproved
if s.moderation != nil && s.moderation() {
    status = guestbook.StatusPending
}
```

仓储接口变更（domain/guestbook/repository.go:12-21）：

```go
type Repository interface {
    // List 语义变更：公开列表，仅返回 approved（创建时间倒序）。**改 SQL 而非新增 ListApproved**——
    // 结论依据：全仓仅两个消费方（interfaces/guestbook.go:53、interfaces/profiles.go:112）都是公开场景，
    // 没有"全量"消费方，改一处 SQL 即可同时覆盖 /api/guestbook 与 /author/:name 页，行为变更面最小。
    List(limit int) ([]*Entry, error)
    ListAll(limit int) ([]*Entry, error)        // 新增：面板全量（含全部状态，倒序）
    ListPending() ([]*Entry, error)             // 新增：待审核（正序）
    ListRejected() ([]*Entry, error)            // 新增：被驳回（正序）
    SetStatus(id int64, status string) error    // 新增：UPDATE guestbook SET status=?, rejected_reason='' WHERE id=?
    SetRejected(id int64, reason string) error  // 新增：UPDATE guestbook SET status='rejected', rejected_reason=? WHERE id=?
    // Get / Create / Delete 不变（Get 需补扫 status、rejected_reason 两列）
}
```

新用例：

```go
func (s *Service) Approve(id int64) error                     // Get → SetStatus(approved)
func (s *Service) Reject(id int64, reason string) error       // ValidateRejectedReason → Get → SetRejected
func (s *Service) Recover(id int64) error                     // Get → 非 rejected 则 ErrInvalidState → SetStatus(approved)
func (s *Service) ListPending() ([]*guestbook.Entry, error)
func (s *Service) ListRejected() ([]*guestbook.Entry, error)
func (s *Service) ListAll(limit int) ([]*guestbook.Entry, error)
```

`List(limit)`、`Delete(userID, role, id)` 签名不变。错误映射沿用现有 `ValidationError`（→400）+ `guestbook.ErrNotFound`（→404）。

### 5.3 组装根（register.go）

- :85 `guestbook := guestbookapp.NewService(guestbookRepo, guestbookRepo, func() bool { on, err := settings.Moderation(); return err == nil && on })` —— 与 :78-81 评论的注入模式完全一致（同一 `settings.Moderation()` 开关同时管评论与留言，设置页文案同步见 §6.4）。
- 接线 RegisterGuestbookAdmin（置于 :111 RegisterGuestbookAPI 之后）。

## 6. 接口与面板变更

### 6.1 评论面板操作接口（interfaces/interactions.go）

结论：**就地扩展**，紧邻现有 approve（:145-161），不走独立 admin 文件——评论的全部变更操作已集中在 RegisterInteractions，保持一处维护；角色守卫沿用 `[]string{"author"}`。

```go
// POST /comments/:id/reject   body {"reason": "..."}（必填 ≤200，ValidationError→400；ErrNotFound→404）
// 成功 {ok:true}；不做读者失效（见 §7.1 规则表）
// POST /comments/:id/recover  （ErrNotFound→404；ErrInvalidState→400；成功 {ok:true}）
// 失效声明：宿主——MomentID>0 → a.DataChange("/moments")，否则 declarePostsChanged(a, postID)（复用 :153-157 模式）
```

### 6.2 留言板管理（结论：独立页 /admin/guestbook，不并入 /admin/comments）

理由：留言板是独立 domain/repository/service；评论管理页已含 pending+全量+搜索，再塞一个实体会让数据形状和筛选混乱；仓库现有 admin 页面（posts/comments/moments）均为"每实体一页"，独立页与之同构；侧边栏加一项仅一行（adminLayout.tsx:10-17）。新函数 `RegisterGuestbookAdmin`（放 interfaces/guestbook.go 文件内，与 RegisterGuestbookAPI 同包）：

```go
// RegisterGuestbookAdmin 注册留言板管理页与审核 API（全部 author 守卫）。
func RegisterGuestbookAdmin(a *hybrid.App, gb *guestbookapp.Service, authorNameFn func() string) error {
    // a.Page("/admin/guestbook", ["author"]) → {entries: gb.ListAll(200), pending: gb.ListPending(), rejected: gb.ListRejected()}
    //   adminGuestbookView{ID, UserID, Username, Content, Status, RejectedReason, CreatedAt}
    // POST /api/guestbook/:id/approve → gb.Approve → a.InvalidatePage("/author/"+authorNameFn())
    // POST /api/guestbook/:id/reject  body{"reason"} → gb.Reject → 无失效
    // POST /api/guestbook/:id/recover → gb.Recover → a.InvalidatePage("/author/"+authorNameFn())
    // DELETE /api/guestbook/:id 复用现有（interfaces/guestbook.go:87-107，author 可删）
}
```

同时 `GuestbookView`（interfaces/guestbook.go:15-21）加 `Status string \`json:"status"\`` —— 供发表接口响应携带 pending，前端据此提示"待审核"（公开列表全是 approved，无泄露）。

### 6.3 评论管理页（/admin/comments）

- `adminCommentView`（admin.go:21-30）加 `RejectedReason string \`json:"rejectedReason"\``。
- 页面 handler（admin.go:225-261）增加 `rejected` 数组（`comments.ListRejected()`），initialState 变为 `{comments, pending, rejected}`。注：现有 `ListAll(100)`（admin.go:226）无状态过滤，rejected 会自然出现在"全部"列表，与 FilterSelect 新增"已驳回"选项配合展示。
- 前端 `admin/comments/page.tsx`：新增 rejected 区块（内容 + 驳回原因 + **恢复按钮** + 删除）；`approve/reject/recover` 成功后本地 state 三表迁移（沿用现有 :33-38 的 setPending 模式）；`admin/types.ts` 的 AdminComment +rejectedReason、AdminCommentsState +rejected。

### 6.4 前端其余变更

- `src/admin/guestbook/page.tsx` 新建：pending 区（通过/驳回[原因输入]/删除）、rejected 区（原因展示 + 恢复/删除）、全量区（搜索 + 状态筛选，对齐评论页结构）。
- `src/admin/adminLayout.tsx:10-17` TABS 加 `{ href: "/admin/guestbook", label: "留言", exact: false }`。
- `src/author/[name]/page.tsx` GuestbookSection（:382-431）：发表响应 `status === "pending"` 时提示"留言已提交，待作者审核通过后展示"（对齐评论页 `src/posts/comments.tsx:82-85` 模式）；列表项加"待审核"chip。
- `src/author/types.ts:38-44` GuestbookEntry 加 `status?: string`。
- `src/admin/settings/page.tsx:718` 开关文案改为"开启评论与留言审核（开启后，新评论与新留言需人工审核通过才会公开显示）"。

## 7. 失效与邮件规则

### 7.1 页面失效规则表（完整）

| 操作 | 服务方法 | 读者可见性变化 | 读者失效声明 | 面板刷新 |
|---|---|---|---|---|
| 评论 approve | commentapp.Approve | 不可见→可见 | 宿主：MomentID>0 → `DataChange("/moments")`；否则 `declarePostsChanged(a, postID)`（现有 interactions.go:153-157） | 前端本地 state（现有 page.tsx:33-38） |
| 评论 recover | commentapp.Recover | 不可见→可见 | 同 approve（宿主） | 前端本地 state：rejected → approved |
| 评论 reject | commentapp.Reject | **无**（面板仅对 pending/rejected 提供驳回；见注 1） | **无** | 前端本地 state：pending/rejected → rejected |
| 评论 delete | commentapp.Delete | 可见→不可见 | 宿主（现有 interactions.go:182-186） | 前端本地 state（现有） |
| 留言 approve / recover | guestbookapp.Approve / Recover | 不可见→可见 | `InvalidatePage("/author/"+authorNameFn())`（对齐 guestbook.go:105 删除时的写法） | 前端本地 state |
| 留言 reject | guestbookapp.Reject | 无（pending→rejected 均不公开） | 无 | 前端本地 state |
| 留言 delete | guestbookapp.Delete | 可见→不可见 | `InvalidatePage("/author/"+authorNameFn())`（现有 guestbook.go:105） | 前端本地 state（现有 :421-430） |

注 1：服务层允许任意状态 reject（Unit 2 MCP 可能驳回已通过评论），但**本单元面板 UI 只对 pending/rejected 提供驳回操作，不对 approved 提供**——避免"读者已见内容消失但页面未失效"的坑。若 Unit 2 的 MCP action 允许对 approved 执行 reject，其文档须声明宿主失效责任（风险项，§11）。

### 7.2 邮件规则与触发点

**决策：面板人工操作（approve/reject/recover/delete）一律不主动发邮件**（人就在面板前，通知无意义）。驳回邮件只由两个场景触发：

- **Unit 4 审核 worker**：AutoReview 调用 `commentapp.Reject` 后自行发送。
- **Unit 2 MCP action**：`comment.reject` action 调用 `commentapp.Reject` 后自行发送。

**本单元留出的触发点**（不实现发送）：

1. `commentapp.Reject(commentID, reason)` 返回宿主且内部已 Get 过评论——为不发邮件污染用例，不把 mailer 注入 commentapp。
2. 新增薄封装 `commentapp.Get(commentID) (*comment.Comment, error)`（§5.1）——后续单元取评论（含 UserID/Content）拼驳回邮件：`Get(commentID) → userapp.FindByID(c.UserID) → mailer.Send(收件人邮箱, subject, 正文含 reason 与原文链接)`，链接拼法对齐 emailauth.NotifyMentioned 的 `siteURL + path` 模式（emailauth/service.go:87-109）；收件人地址缺省可用 `settingsapp.AuthorEmail()`（settingsapp/service.go:121-123）。

## 8. 行为变更清单

**公开行为变更（必须回归验证）：**

1. `GET /api/guestbook`（interfaces/guestbook.go:52-60）：从"全量 50 条"改为**只返回 approved**（仓储 List SQL 加 `WHERE status='approved'`，guestbook_repository.go:22-45）。
2. `/author/:name` 页 initialState 的 guestbook（interfaces/profiles.go:112 `gb.List(50)`）：同源变更，自动变为只含 approved（同一 repo.List）。
3. 发表留言：moderation 开时初始状态 pending（原直接 approved 立即可见）；发表接口响应新增 `status` 字段，作者主页发表后出现"待审核"提示。
4. 审核开关（`comment_moderation` 设置，settingsapp/service.go:75-81）**现在同时管评论与留言**，设置页文案同步（§6.4）。

**确认无需变更（已核对代码）：**

5. `GET /api/posts/:id` 详情页评论（pages.go:62 ListForPost → 仓储 ListByPost）**已只返回 approved**（comment_repository.go:36-38，`WHERE c.post_id = ? AND c.status = ?`）——本单元不动。
6. `GET /api/moments/:id/comments`（interactions.go:232-238 → ListByMoment）同样已只返回 approved（comment_repository.go:41-43）——不动。

**相邻一致性问题（建议本单元顺手处理，风险低；排期紧可降 backlog 并在验收中剔除）：**

7. `MomentCommentCounts`（comment_repository.go:64-80，/moments 页评论数）与用户个人页评论统计（user_repository.go:127）按 `COUNT(*)` 全量统计——rejected 出现后会虚增公开计数。建议两处 SQL 各加 `status='approved'` 过滤。注：pending 虚增问题在 005 引入审核时已存在，非本单元引入，但 reject 让该偏差更显著（AI 批量驳回时计数会大量虚高）。

## 9. 验收标准

**A. 迁移（真实 MySQL）**
- A1. 对含存量数据的库执行 009：guestbook 全表 `status='approved'`、`rejected_reason=''`；comments 存在 `rejected_reason` 列且默认 `''`。
- A2. 重复执行 009 无报错、无重复列（幂等）。
- A3. 服务启动跑完整迁移链无报错（db.go 切片含 009）。

**B. 领域与应用层（Go 单测，§10）**
- B1. `ValidateRejectedReason`：空/纯空白 → 报错；201 字符 → 报错；200 字符 → 通过。
- B2. `commentapp.Reject`：空 reason → ValidationError；合法 → 状态 rejected 且 reason 落库；返回宿主正确；不存在 → ErrNotFound。
- B3. `commentapp.Recover`：对 pending → ErrInvalidState；对 rejected → approved 且 reason 清空；返回宿主正确。
- B4. `commentapp.Approve` 对 rejected 调用 = 等价 recover（approved + reason 清空）。
- B5. `guestbookapp.Create`：moderation on → pending；off → approved。
- B6. `guestbookapp.Reject/Recover` 规则同 B2/B3；`List` 只返回 approved。

**C. 行为变更核心验收（手工 + curl，服务跑在 :8080）**

前置：moderation off 下发表留言 A（approved）→ 设置页开启审核 → 发表留言 B（pending）→ 面板驳回 B（填原因）。

- C1. `curl http://127.0.0.1:8080/api/guestbook`：`entries` 含 A **不含 B**（公开查询只返回 approved）。
- C2. 访问 `/author/<作者名>`：页面留言区同 C1（只含 A）。
- C3. `curl http://127.0.0.1:8080/api/guestbook/` + B 的 id（POST approve 前）：B 不出现在列表中；面板 `/admin/guestbook` 中 B 出现在 pending 区 → 驳回（填原因）后进入 rejected 区且展示原因 → 恢复后回到 approved 区；期间公开接口始终不含 B。
- C4. 评论三态闭环：发表评论（审核开）→ `/admin/comments` pending 区通过 → 详情页可见；再发表 → 驳回（填原因）→ rejected 区展示原因（含"全部"列表内）；点恢复 → 详情页重新可见（ISR 场景等待再生物化后验证）。
- C5. 权限：未登录调用 `/api/comments/:id/reject`、`/api/guestbook/:id/approve` 等 → 401；reader 角色 → 401/403。
- C6. 审核开关联动：关闭审核后新留言/新评论直接 approved 并立即可见（公开列表出现）。

## 10. 测试计划

**现状**：`frame/go/build/` 下无任何 `*_test.go`（测试先例仅在 `frame/go/hybrid/` 与 `frame/go/internal/`，均为标准库 testing）。本单元新建以下测试，采用标准库 + 表驱动，不引入新依赖：

| 测试文件（新建） | 覆盖 |
|---|---|
| `build/domain/comment/comment_test.go` | ValidateRejectedReason 边界（B1） |
| `build/application/commentapp/service_test.go` | 内存假 repo 实现 comment.Repository 全 11 方法：Reject（B2）/Recover（B3）/Approve 对 rejected（B4）/Create moderation 开关两态/ListRejected/Delete 权限（本人 vs 他人 vs author） |
| `build/domain/guestbook/guestbook_test.go` | ValidateRejectedReason |
| `build/application/guestbookapp/service_test.go` | 内存假 repo：Create 两态（B5）/Approve/Reject/Recover（B6）/ListPending/ListRejected/List 仅 approved |

**SQL 变更（真实 MySQL，非单测）**：本地起 MySQL → 造存量数据（插入若干 guestbook/comments 行，含各种 status）→ 执行 `009_ugc_status.sql` → 断言回填与默认值（A1）→ 再执行一次（A2）→ 跑 `go run` 主程序验证迁移链（A3）。

**interfaces 层**：无测试先例，不新造测试基建；按 §9-C 手工验证步骤走一遍（curl 序列 + 面板点击路径清单），并在 PR 描述中附执行结果。

## 11. 风险与开放问题

1. **Unit 2 契约**：`comment.reject/recover` MCP action 的 reason 必填/长度约束以此文档为准（≤200 必填）；若 Unit 2 允许对 **approved** 评论执行 reject，则读者失效责任在 Unit 2（本单元接口层不做，见 §7.1 注 1）——需在 Unit 2 文档明确，避免读者侧内容残留。
2. **Unit 4 AutoReview 位置（占位契约，本单元不实现）**：建议新建 `build/application/autoreview/service.go`：`type Classifier interface { Decide(content string) (Approved bool, Reason string) }` + `Service{comments *commentapp.Service, guestbook *guestbookapp.Service, classify Classifier}`，用例 `ReviewPending() (int, error)` 循环 `ListPending()` → 通过则 `Approve` / 驳回则 `Reject`（带 reason）+ 自行发驳回邮件（经 §7.2 触发点）。worker 调度（定时/队列）由 Unit 4 决定。
3. **008/009 合并顺序**：两单元都改 `db.go` 的 migrations 切片，合入时必须 008 在前 009 在后；009 不依赖 008 的列，可独立合入。
4. **驳回原因无审计历史**：覆盖式存储（重复 reject 覆盖旧 reason）。如需审计由 Unit 4 扩展（如 reason 前缀记录）。
5. **计数偏差**：§8.7 的 MomentCommentCounts / 个人页评论统计含 pending/rejected——建议本单元顺手修，否则 Unit 4 批量驳回后 /moments 计数显著虚高。
6. **长度单位**：`ValidateRejectedReason` 与现有 `Validate` 一致用 `len()`（字节数）——中文 200 字符约 600 字节会误判超长，属现状一致的已知取舍，不扩 scope；如需按 rune 计需同步改评论/留言的现有 Validate（超出本单元）。
7. **审核开关共用**：留言板复用 `comment_moderation`，若后续想要"评论审核开、留言审核关"的独立粒度，需新增设置 key（本单元不拆，控制面复杂度最小）。
