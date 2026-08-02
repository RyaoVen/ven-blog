# Unit 1 详细设计：API Key 密钥管理系统

> 单元范围：管理员在后台面板自助生成 / 查看 / 吊销 API key；agent 请求携带 `Authorization: Bearer ven_xxx` 时可校验出 author 身份（校验入口 `AuthenticateKey` 本单元定签名，**网关消费方是 Unit 2 的 /api/mcp 中间件**，本单元不实现网关）。
> 机制对标大模型厂商：明文仅创建时显示一次、服务端只存 sha256、多 key 并存按用途命名、吊销即时生效（每次请求查库，无缓存）。

---

## 1. 目标

1. 新增 `api_keys` 表（migration `008_api_keys.sql`，**仅本单元使用**，其它单元不得占用 008 编号）。
2. 新增 DDD 聚合 `domain/apikey`：实体 `ApiKey`、密钥生成 / 哈希 / 校验领域规则、仓储接口。
3. 新增 MySQL 实现 `persistence.NewApiKeyRepository(db)`，风格对齐 `email_code_repository.go`。
4. 新增应用服务 `apikeyapp.Service`：`CreateKey` / `ListKeys` / `Revoke` / `AuthenticateKey`（签名本单元定稿）。
5. 新增接口层 `interfaces/keys.go` 的 `RegisterKeysAdmin`：cookie 鉴权（roles=`["author"]`）的后台管理 API，挂入 `register.go`。
6. Node 后台设置页新增「API 访问密钥」区块：列表 + 新建（明文弹窗仅一次）+ 吊销。
7. 安全基线：明文仅显示一次、hash 存储、吊销即时生效、无过期时间（v1 简化，见 §7）。

## 2. 涉及文件清单

| 文件 | 动作 | 说明 |
|---|---|---|
| `frame/go/build/infrastructure/persistence/migrations/008_api_keys.sql` | 新建 | api_keys 表 DDL（§3） |
| `frame/go/build/infrastructure/persistence/db.go` | 修改 | embed 008 并追加进 `migrations` 切片（现为 :16-38） |
| `frame/go/build/domain/apikey/apikey.go` | 新建 | 实体 + 领域规则（§4.1） |
| `frame/go/build/domain/apikey/repository.go` | 新建 | 仓储接口 + 领域错误（§4.1） |
| `frame/go/build/infrastructure/persistence/api_key_repository.go` | 新建 | MySQL 实现（§4.2） |
| `frame/go/build/application/apikeyapp/service.go` | 新建 | 用例服务 + `KeyView` 脱敏视图（§4.3） |
| `frame/go/build/application/apikeyapp/service_test.go` | 新建 | 假 repo 纯逻辑测试（§9） |
| `frame/go/build/interfaces/keys.go` | 新建 | `RegisterKeysAdmin`（§5） |
| `frame/go/build/register.go` | 修改 | repo → service → RegisterKeysAdmin 接线（§5.2） |
| `src/admin/settings/page.tsx` | 修改 | 新增 `KeysSection` 区块（§6） |
| `src/admin/settingsTypes.ts` | 修改 | 新增 `ApiKeyView` 类型（§6） |

## 3. 数据模型（migration 008_api_keys.sql）

对齐既有风格：`CREATE TABLE IF NOT EXISTS` 幂等（`001_init.sql:4`、`007_email.sql:33`）、`BIGINT UNSIGNED` 主键 + `DATETIME` 时间戳（`001_init.sql:4-14`）、外键 `REFERENCES users (id)`（`001_init.sql:27`）、utf8mb4 引擎（`007_email.sql:42`）。

```sql
-- 008：API 访问密钥（程序化鉴权凭据，agent 调用网关用）
-- 服务端只存 sha256 哈希，明文仅在创建响应中出现一次；吊销即终态，不可恢复

CREATE TABLE IF NOT EXISTS api_keys (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id      BIGINT UNSIGNED NOT NULL,
    name         VARCHAR(64)     NOT NULL COMMENT '用途备注，如 zcode-agent',
    key_hash     CHAR(64)        NOT NULL COMMENT 'sha256(明文) 十六进制，唯一检索键',
    prefix       VARCHAR(16)     NOT NULL COMMENT '明文前 8 位（如 ven_ab12），列表展示用，不可还原',
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME        NULL COMMENT '最近一次鉴权成功时间，NULL=从未使用',
    revoked_at   DATETIME        NULL COMMENT '吊销时间，非 NULL=终态失效',
    PRIMARY KEY (id),
    UNIQUE KEY uk_api_keys_hash (key_hash),
    KEY idx_api_keys_user (user_id),
    CONSTRAINT fk_api_keys_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
```

字段说明：

- `key_hash CHAR(64)`：sha256 的 hex 恰好 64 字符；唯一索引保证 `FindByHash` O(1)，也杜绝两把 key 撞 hash。
- `prefix VARCHAR(16)`：明文前 8 位（`ven_` + 4 位随机段），列表展示用；8 位内可辨认用途又不泄露明文。
- `last_used_at NULL`：未使用为 NULL；鉴权成功每次更新（每次请求查库，无缓存，见 §7）。
- `revoked_at NULL`：NULL = 在用；非 NULL = 终态。**不提供恢复操作**（无 unrevoke）。
- 不建 `expires_at`：v1 不做过期，理由见 §7。

配套修改 `frame/go/build/infrastructure/persistence/db.go`（现有 embed 模式 :16-38）：

```go
//go:embed migrations/008_api_keys.sql
var migration008 string

var migrations = []string{migration001, ..., migration007, migration008}
```

## 4. 领域与应用层接口契约

### 4.1 domain/apikey（`domain/apikey/apikey.go` + `repository.go`）

领域校验函数风格对齐 `domain/post/post.go:60`（`Validate` 返回错误消息，空串 = 通过）、`domain/user/user.go:34`。

```go
// Package apikey API 密钥聚合：程序化鉴权凭据的实体与领域规则。
package apikey

// 领域常量。
const (
    KeyPrefix        = "ven_" // 明文固定前缀（凭据体系标识，避免与 cookie/验证码混淆）
    RandomBytes      = 32     // 随机段字节数 → 明文总长 = 4 + 43 = 47（base64url 无填充）
    MaxNameLen       = 64     // name 上限（对齐 api_keys.name VARCHAR(64)）
    DisplayPrefixLen = 8      // 展示 prefix 截取长度（对齐 api_keys.prefix VARCHAR(16)）
)

// 领域错误（风格对齐 domain/user/repository.go:14-17）。
var (
    ErrNotFound = errors.New("api key not found")
    ErrRevoked  = errors.New("api key revoked")
)

// ApiKey API 密钥实体。
// 服务端唯一存储形态是 KeyHash；明文生命周期只存在于「创建时的一次返回」。
type ApiKey struct {
    ID         int64
    UserID     int64
    Name       string    // 用途备注（如 "zcode-agent"）
    KeyHash    string    // sha256(明文) 十六进制
    Prefix     string    // 明文前 8 位，展示用
    CreatedAt  time.Time
    LastUsedAt time.Time // 零值 = 从未使用
    RevokedAt  time.Time // 零值 = 在用；非零 = 终态，不可恢复
}

// Revoked 是否已吊销（终态判定）。
func (k *ApiKey) Revoked() bool { return !k.RevokedAt.IsZero() }

// GenerateKey 生成明文密钥：ven_ + base64.RawURLEncoding(32 随机字节)，总长 47。
// 用 crypto/rand（不落日志、不落库，仅返回给创建方一次）。
func GenerateKey() (string, error)

// HashKey 计算明文密钥的 sha256 十六进制（存储形态）。
func HashKey(raw string) string

// ValidateName 校验备注名：trim 后非空且 ≤ 64 字符；返回错误消息（空串 = 通过）。
func ValidateName(name string) string

// DisplayPrefix 截取明文前 8 位作展示前缀。
func DisplayPrefix(raw string) string
```

仓储接口（`domain/apikey/repository.go`，风格对齐 `domain/setting/setting.go:31-37`）：

```go
// Repository API 密钥仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
    // Create 新建密钥（user_id/name/key_hash/prefix），回填 ID 与 CreatedAt。
    Create(k *ApiKey) error
    // FindByHash 按 hash 精确查找（唯一索引），不存在返回 ErrNotFound。
    FindByHash(hash string) (*ApiKey, error)
    // ListByUser 返回某用户全部密钥（创建时间倒序，含已吊销）。
    ListByUser(userID int64) ([]*ApiKey, error)
    // Revoke 吊销（写 revoked_at）：仅限本人（user_id 匹配且未吊销）。
    // 不存在 / 已吊销 / 非本人统一返回 ErrNotFound（不泄露 key 存在性）。
    Revoke(userID, id int64) error
    // UpdateLastUsedAt 写最后使用时间（鉴权成功后调用）。
    UpdateLastUsedAt(id int64, t time.Time) error
}
```

### 4.2 infrastructure/persistence（`persistence/api_key_repository.go`）

风格对齐 `email_code_repository.go`（结构体持 `*sql.DB`、构造器、SQL 字符串、`sql.ErrNoRows` 分支）。

```go
// ApiKeyRepository 是 apikey.Repository 的 MySQL 实现。
type ApiKeyRepository struct{ db *sql.DB }

func NewApiKeyRepository(db *sql.DB) *ApiKeyRepository
```

实现要点：

- `Create`：`INSERT INTO api_keys (user_id, name, key_hash, prefix) VALUES (?, ?, ?, ?)`，`LastInsertId()` 回填 ID，`CreatedAt` 用 `time.Now()`（或读库回填）。
- `FindByHash`：`SELECT id, user_id, name, key_hash, prefix, created_at, last_used_at, revoked_at FROM api_keys WHERE key_hash = ?`；`last_used_at`/`revoked_at` 用 `sql.NullTime` 扫描，NULL → 零值；`sql.ErrNoRows` → `apikey.ErrNotFound`。
- `ListByUser`：`... WHERE user_id = ? ORDER BY id DESC`。
- `Revoke`：`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`；`RowsAffected() == 0` → `apikey.ErrNotFound`（一行 SQL 同时保证本人归属 + 幂等 + 不泄露存在性）。
- `UpdateLastUsedAt`：`UPDATE api_keys SET last_used_at = ? WHERE id = ?`。

### 4.3 application/apikeyapp（`application/apikeyapp/service.go`）

错误与视图类型风格对齐 `userapp/service.go:86-89`（`ValidationError`）与 `interfaces/dto.go:15-31`（ID 字符串下发）。

```go
// Package apikeyapp API 密钥用例服务：生成 / 列表 / 吊销 / 鉴权。
package apikeyapp

// ValidationError 用例入参校验失败（接口层映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// KeyView 脱敏视图（永不含明文；接口层直接下发）。
type KeyView struct {
    ID         string     `json:"id"`         // 字符串下发（对齐 PostView 契约）
    Name       string     `json:"name"`
    Prefix     string     `json:"prefix"`     // 如 ven_ab12（展示用）
    CreatedAt  time.Time  `json:"createdAt"`
    LastUsedAt *time.Time `json:"lastUsedAt"` // null = 从未使用
    RevokedAt  *time.Time `json:"revokedAt"`  // null = 在用
}

// Service API 密钥用例服务。
type Service struct{ repo apikey.Repository }

func NewService(repo apikey.Repository) *Service

// CreateKey 生成密钥：GenerateKey → HashKey → repo.Create，返回明文与脱敏视图。
// 明文仅此一次返回，调用方展示后必须丢弃；name 非法返回 *ValidationError。
func (s *Service) CreateKey(userID int64, name string) (raw string, view KeyView, err error)

// ListKeys 返回该用户全部密钥的脱敏视图（创建时间倒序，含已吊销），永不含明文。
func (s *Service) ListKeys(userID int64) ([]KeyView, error)

// Revoke 吊销本人密钥（吊销即终态，无恢复入口）；
// 不存在 / 已吊销 / 非本人返回 apikey.ErrNotFound。
func (s *Service) Revoke(userID, id int64) error

// AuthenticateKey 用明文密钥换取用户身份：HashKey → FindByHash →
// 已吊销返回 apikey.ErrRevoked → UpdateLastUsedAt（失败仅记日志，不阻断鉴权）→ 返回 userID。
// 每次调用实时查库（无缓存），吊销即刻生效。
// 【调用方】Unit 2 的 /api/mcp 网关中间件（Bearer 解析在网关侧做，本单元只定签名，不实现网关）。
func (s *Service) AuthenticateKey(rawKey string) (userID int64, err error)
```

业务规则（在 Service 内编排，不泄漏到接口层）：

1. 明文生成后立即哈希，**只有 hash 进入仓储**；返回明文是唯一的出口。
2. `AuthenticateKey` 失败路径统一返回领域错误：hash 查无 → `ErrNotFound`；查到但 `Revoked()` → `ErrRevoked`。
3. `last_used_at` 更新失败只记日志不影响鉴权结果（个人博客写放大可忽略，见 §10）。

## 5. HTTP 接口契约

### 5.1 interfaces/keys.go（`RegisterKeysAdmin`）

模式完全对齐 `interfaces/settings.go:15-25`（`admin := []string{"author"}` + `a.Post(a, roles, handler)`），注册函数签名对齐 `settings.go:15` 的 `RegisterXxx(a *hybrid.App, deps...) error`。路由经 `hybrid/api.go:15-32` 自动加 `/api` 前缀并做 cookie 角色守卫（`hybrid/authz.go:33-45`）；身份取 `c.User()`（`hybrid/apictx.go:42-44`）后经 `strconv.ParseInt` 转 int64（复用 `interfaces/apis.go:36` 的 `currentUserID(c)` 助手）。

| 方法 | 路径（实际注册 pattern） | 守卫 | 语义 |
|---|---|---|---|
| POST | `/admin/keys`（实际 `/api/admin/keys`） | `["author"]` | 生成密钥，响应含明文 + 警告 |
| GET | `/admin/keys` | `["author"]` | 列表（脱敏视图） |
| DELETE | `/admin/keys/:id` | `["author"]` | 吊销（即时生效） |

```go
// RegisterKeysAdmin 注册 API 密钥管理接口（cookie 鉴权，author 守卫）。
// 注意：这是后台管理面；程序化鉴权消费方是 Unit 2 的 /api/mcp 网关（调 apikeyapp.AuthenticateKey）。
func RegisterKeysAdmin(a *hybrid.App, keys *apikeyapp.Service) error
```

**POST /api/admin/keys**

- 请求体：`{"name": "zcode-agent"}`（name 为用途备注，1-64 字符）。
- 成功 `200`：

```json
{
  "key": "ven_<43 位 base64url>",
  "warning": "密钥明文仅此一次展示，关闭后无法再次查看；请立即复制妥善保存。",
  "view": {
    "id": "1", "name": "zcode-agent", "prefix": "ven_ab12",
    "createdAt": "2026-08-02T10:00:00+08:00", "lastUsedAt": null, "revokedAt": null
  }
}
```

- 失败：`400 {"error": "name is required"}` / `{"error": "name too long (max 64)"}`（`ValidationError.Message`）；`401 {"error": "unauthenticated"}`、`403 {"error": "forbidden"}`（框架守卫，`hybrid/api.go:48-54`）；`500 {"error": "internal error"}`。

**GET /api/admin/keys**

- 成功 `200`：`{"keys": [KeyView, ...]}`（按创建时间倒序，含已吊销；永不含明文）。
- 失败：401 / 403 / 500，同上。

**DELETE /api/admin/keys/:id**

- `:id` 为字符串数字（对齐 `interfaces/admin.go:208` 的 `mustID(c.Param("id"))` 用法；`parseID` 在 `dto.go:52`）。
- 成功 `200`：`{"ok": true}`。
- 失败：`400 {"error": "invalid key id"}`（id 非数字）；`404 {"error": "api key not found"}`（不存在 / 已吊销 / 非本人，统一不泄露存在性）；401 / 403 / 500。

### 5.2 register.go 接线（`frame/go/build/register.go`）

按组装根三段式（repo 构造 :39-48 → 应用服务 :70-85 → 接口注册 :88-145）：

```go
// 基础设施段（:48 后追加）
apiKeyRepo := persistence.NewApiKeyRepository(db)

// 应用服务段（:85 后追加）
apiKeys := apikeyapp.NewService(apiKeyRepo)

// 接口注册段（:145 RegisterAdmin 之后追加）
if err := interfaces.RegisterKeysAdmin(a, apiKeys); err != nil {
    return err
}
```

## 6. 面板设计

> 注：任务描述中的 `frame/node/src/admin/` 实际位于仓库根 `src/admin/`（`frame/node` 是 SSR worker，`src/**/page.tsx` 才是页面唯一真相源，见 AGENTS.md:7）。

**改动文件**：`src/admin/settings/page.tsx` 新增 `KeysSection` 组件，在 `ModerationSection` 之后渲染（:55）；`src/admin/settingsTypes.ts` 新增 `ApiKeyView` 接口（与 Go 侧 `KeyView` JSON 同形）。

**数据流**（客户端现取，不进 SSR initialState）：

1. **列表**：组件 mount 时 `fetch("/api/admin/keys")` → `keys: ApiKeyView[]` 本地 state；每行展示 `prefix`（等宽字体，`ven_ab12…` 样式）、`name`、`createdAt`、`lastUsedAt`（null 显示"从未使用"）、已吊销行灰显并禁用操作。
2. **新建**：输入 name → `POST /api/admin/keys`（JSON body）→ 成功后弹窗（复用 `src/lib/modal.tsx` 的 `Modal`）展示明文一次 + 醒目警告「密钥只显示这一次，关闭后将无法再次查看」+ 「我已复制」确认按钮 → 关窗即丢弃，重新拉列表。
3. **吊销**：行内「吊销」按钮 → 确认弹窗（提示即时生效）→ `DELETE /api/admin/keys/${id}` → 成功刷新列表（或本地置 `revokedAt`）。

**为什么客户端现取而非塞进 `/admin/settings` 页面 initialState**：密钥数据动态（`last_used_at` 随鉴权变化、可随时吊销），且含高敏操作，与设置页其它静态配置区块（quotes/categories 等，`interfaces/settings.go:52-63`）性质不同；现取还避免 SSR 数据函数里多一次查库。

**既有可复用物**：`src/lib/modal.tsx`（Modal 弹窗，`settings/page.tsx:664` 已有先例）、`useToast`（:23-36）、`ven-card`/`ven-btn ven-btn-danger` 样式、`v` 主题变量。

## 7. 安全设计

1. **明文仅显示一次**：明文由 `apikeyapp.CreateKey` 内部生成，落库的只有 sha256；明文经接口响应返回一次，前端只在弹窗内存中展示，关闭即丢弃。任何日志（`log.Printf` 等）禁止打印明文，调试只允许打印 `prefix`。
2. **hash 存储**：`key_hash = sha256(明文)` hex。与密码不同，key 是 256 位随机熵，不需要加盐/慢哈希（bcrypt 反而无用武之地）；sha256 单向性 + 高熵保证拖库后无法还原。
3. **吊销即时生效**：`AuthenticateKey` 每次请求实时 `FindByHash` 查库（唯一索引 O(1)），**无内存缓存、无 TTL**；`revoked_at` 非 NULL 即返回 `ErrRevoked`——吊销后下一请求立即失效，无需等缓存过期。
4. **Bearer 解析格式**（Unit 2 网关中间件职责，此处定规范）：`Authorization: Bearer ven_xxx`；解析后先做格式校验（`ven_` 前缀 + 总长 47 + base64url 字符集，`apikey.IsValidFormat` 风格），格式不对直接 401 拒绝，不进 DB；再调 `AuthenticateKey(raw)`。
5. **不做 key 过期时间（v1 简化）**：① 吊销已覆盖"收回权限"的全部需求，过期只是自动化的吊销；② 过期需要创建/续期/到期提醒的完整 UX 闭环，v1 不值得；③ `last_used_at` 已提供观测依据（哪个 key 在用、多久没用），后续要加只需 `ALTER TABLE ADD expires_at DATETIME NULL` 追加一期 migration，不影响现有结构。故 008 不建 `expires_at`。
6. **存在性不泄露**：`Revoke` 与 `FindByHash` 的失败统一为 `ErrNotFound`，吊销/不存在/非本人不区分。

## 8. 验收标准（可独立验收清单）

1. [ ] 启动服务（`BLOG_MYSQL_DSN` 指向真实 MySQL）自动建 `api_keys` 表；重复启动幂等不报错；`information_schema` 中表结构与 §3 一致（含 `uk_api_keys_hash` 唯一索引、外键）。
2. [ ] 登录 author（ven_auth cookie）后 `POST /api/admin/keys` 返回 200，响应含 `ven_` 开头的 47 位明文 + warning + view；请求体缺 name → 400；name 超 64 字符 → 400。
3. [ ] 数据库中该行 `key_hash` 为 64 位 hex 且不等于明文；`prefix` 为明文前 8 位。
4. [ ] `GET /api/admin/keys` 返回列表（倒序），每条只有 prefix/name/时间，**不含明文**；`lastUsedAt` 初始为 null。
5. [ ] 未登录访问三个接口 → 401；登录为 reader 角色 → 403（cookie 守卫生效）。
6. [ ] `DELETE /api/admin/keys/:id` → 200；重复吊销同一 id → 404；吊销他人 key（如伪造 id 不存在的用户隔离）→ 404。
7. [ ] 吊销后（unit 联调或临时测试入口调 `AuthenticateKey`）立即返回 `apikey.ErrRevoked`；未吊销且 hash 正确返回 userID；乱写 key（格式对但不存在）→ `ErrNotFound`。
8. [ ] 鉴权成功后 `last_used_at` 被更新为非 NULL。
9. [ ] 面板：设置页出现「API 访问密钥」区块；列表展示 prefix/name/创建时间/最后使用/吊销按钮；新建弹窗明文只显示一次（关窗后无法再取）；吊销有确认步骤且成功后列表刷新。
10. [ ] `cd frame/go && go build ./... && go vet ./... && go test ./...` 全绿（含 §9 新增测试）。

## 9. 测试计划

**应用层（`application/apikeyapp/service_test.go`，纯逻辑 + 假 repo，风格对齐 AGENTS.md:24-28）**

- 假 repo：内存 map 实现 `apikey.Repository`（`map[string]*apikey.ApiKey` 按 hash 索引 + 按 user 索引），记录 `UpdateLastUsedAt` 调用次数。
- 用例：
  1. `CreateKey` 返回明文：以 `ven_` 开头、总长 47、两次生成互不相同（随机性）；假 repo 里存的是 `HashKey(raw)` 而非明文。
  2. name 校验：空 / 纯空白 / 65 字符 → `*ValidationError`；正常 name → 视图 `Prefix == raw[:8]`。
  3. `ListKeys`：多 key 倒序、视图不含明文字段（结构断言）。
  4. `AuthenticateKey`：正确明文 → userID 匹配且 `last_used_at` 更新调用 ≥ 1；错误明文 → `ErrNotFound`；吊销后 → `ErrRevoked`（且吊销前正常、吊销后立即失败，验证"即时生效"）。
  5. `Revoke`：本人可吊销；已吊销再吊销 → `ErrNotFound`；假 repo 中非本人 id → `ErrNotFound`。
- 另加 `domain/apikey` 的小用例：`HashKey` 输出 64 位 hex；`ValidateName` 边界；`DisplayPrefix` 截断。

**接口层**：`build/interfaces` 无测试先例（全仓 `frame/go/build` 下无 `*_test.go`）——按手工验证执行，步骤：

```bash
# 1) 登录拿 cookie
curl -i -c jar.txt -H "Content-Type: application/json" -d '{"username":"<author>","password":"<pwd>"}' http://127.0.0.1:8080/auth/login
# 2) 创建 key
curl -b jar.txt -H "Content-Type: application/json" -d '{"name":"zcode-agent"}' http://127.0.0.1:8080/api/admin/keys
# 3) 列表 / 吊销
curl -b jar.txt http://127.0.0.1:8080/api/admin/keys
curl -b jar.txt -X DELETE http://127.0.0.1:8080/api/admin/keys/1
# 4) 未登录 → 401；reader 登录 → 403
# 5) AuthenticateKey 行为：本单元无网关，联调时用临时 Go 测试/调试端点确认（见验收 7）
```

**migration**：用真实 MySQL（`BLOG_MYSQL_DSN`）验证——首次启动建表成功；再次启动幂等；`SHOW CREATE TABLE api_keys` 与 §3 一致；007 已有先例（`007_email.sql` 的 `CREATE TABLE IF NOT EXISTS` 方式）。

**提交前**：`cd frame/go && go build ./... && go vet ./... && go test ./...`；Node 侧 `cd frame/node && npm run typecheck && npm test`（本单元 Node 改动为纯页面组件，typecheck 覆盖）。

## 10. 风险与开放问题

| # | 风险 / 开放问题 | 决策与影响 |
|---|---|---|
| 1 | **key 权限粒度**：所有 key 等价 author，无 scope/role 区分 | v1 简化（个人博客单作者）。若 Unit 2 需要按 key 限权（如只读），需给 api_keys 加 `scopes` 列——追加 migration，不影响本单元验收 |
| 2 | **AuthenticateKey 返回 userID 而非角色** | 身份语义（author/reader）由 Unit 2 按 userID 反查或直接信任为 author 决定；本单元保持最小签名 |
| 3 | **last_used_at 每次鉴权一次 UPDATE** | 个人博客量级无压力；若未来放大可降采样（如 1 小时粒度）或挪到异步 |
| 4 | **无鉴权失败速率限制** | 47 位随机 key 枚举在计算上不可行；如需纵深防御，Unit 2 网关加按 IP 限速即可，不阻塞本单元 |
| 5 | **明文经 HTTP 传输** | 生产环境须置于 HTTPS 之后（网关 TLS 终止）；README/部署文档应注明，本单元不改框架 |
| 6 | **已吊销 key 无法重新启用** | 有意设计（终态不可恢复）；如后续需要可另做"重建"流程（生成新 key）而非恢复 |
| 7 | **008 编号独占** | 本单元用 008_api_keys.sql；后续单元从 009 起编号，避免迁移顺序冲突 |
