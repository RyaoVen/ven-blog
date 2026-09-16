# Unit 6 — 插件系统治理设计（草案 v0.1）

> 状态：**已实施（2026-09-16，issue #2~#4，PR #3~#5；review 修复见 issue #13）**
> 归属：ven-blog 业务仓库（Phase 1 全部落在 `frame/go/build/`，不触框架层）
> 上游：无（本 unit 是新的治理基座）
> 下游：Unit 7（docs 插件，首个真实消费者）
> 日期：2026-09-15

---

## 1. 背景与动机

### 1.1 现状

ven-blog 的业务组装是**静态组合根**模式（`frame/go/build/register.go`）：

- 每新增一个业务模块 = 新增 4 层目录（domain/application/infrastructure/interfaces）+ 组合根里追加 1~3 个 `RegisterXxx` 调用；
- 模块间依赖靠手工传参（如 `RegisterAdmin(a, posts, comments, interactions, moments, subscribe, users, settings, visits)` 已有 9 个参数）；
- 没有"模块"这个一等公民概念：无清单、无启停、无路由所有权声明、无统一生命周期；
- MCP 网关的 action 表是包级闭集 map（`mcpActions`），新业务想贡献 action 只能改 `interfaces/mcp.go` 本体。

当前组合根已有 **20+ 个 Register 调用**、11 个仓储、13 个应用服务。docs 插件（树形内容 + MCP + webhook + admin + 前端页面）若继续静态散点注册，组合根与 mcp.go 会继续膨胀。

### 1.2 动机

**基础不牢，地动山摇**：先立插件契约与治理规则，docs 作为第一个消费者验证契约充分性，后续模块按需迁移（strangler 模式）。

### 1.3 关键源码事实（设计依据）

| 事实 | 出处 | 对设计的影响 |
|---|---|---|
| `hybrid.App` 公开面：`RegisterRole` / `Page` / `StaticPage` / `Get/Post/Put/Delete` / `SetVisitRecorder` / `SetLoginRedirect` / `InvalidatePage` / `Close` / `Listen` | `hybrid/app.go`, `page.go`, `api.go` | 这就是现成的"插件能力面"，插件内核只需包装，不需新造 |
| 注册顺序约束：角色须在页面前注册；fallback 路由由 `Listen` 强制最后；MCP 走原生 fiber 路由与顺序无关 | `app.go:98-103`、`register.go:199-201` | 内核必须保序执行插件注册 |
| `mcpActions` 是**包级** map，`MCP` 结构体持具体服务指针 | `interfaces/mcp.go:133-163` | 需要"实例化 action 注册 API"改造点 |
| Node 页面路由在**构建期**由 `src/**/page.tsx` 扫描生成（`[xxx]`→`:xxx` 单段动态；**无 catch-all**），运行期不可变 | `frame/node/page-builder/pageRouter.ts:108-126` | "页面插件"只能是源码级模块；运行时动态仅限 Go 侧 API/MCP |
| Go ISR `Declaration.Match` 同样要求段数相等，无通配 | `internal/isr/pattern.go:50-66` | docs 树形 URL 依赖上游 catch-all 需求（见 §10） |
| 失效唯一路径 `DataChange(ChangeEvent{Pattern, Params})`，仅在接口层调用；flush 后联动 SSE | `hybrid/staticPage.go`、`internal/event/bus.go` | 插件写操作沿用同一失效纪律 |
| `SetVisitRecorder(fn)` 是框架既有"回调注册钩子"先例；unit-2 用最小接口 `KeyAuthenticator` 解耦 | `framework-requests.md` 需求 2 | 窄接口注入是仓库既有惯例，Runtime 能力面照此设计 |

---

## 2. 目标与非目标

### 目标

- **G1 插件契约**：统一的 `Plugin` 接口 + 元数据（名称/版本/路由前缀所有权/依赖）。
- **G2 注册内核**：有序、可校验、fail-fast、可诊断的插件启动流程。
- **G3 治理面**：清单校验（名称唯一、路由前缀无冲突）、启停开关（settings）、失效声明纪律。
- **G4 生命周期**：`Register → Start → Stop` 三段式，worker 类插件可优雅关停。
- **G5 MCP 可扩展**：插件可贡献 `doc.*` 式 action，无需修改 mcp.go 本体。

### 非目标（Phase 1 明确不做）

- **N1 运行时热加载**（Go plugin .so / 解释器）：跨平台脆弱（本仓库开发机混合 macOS/Windows），Phase 2 再评估（§9）。
- **N2 改框架层**：`hybrid/`、`internal/` 一行不动；框架缺口走上游 issue（§10）。
- **N3 迁移存量模块**：posts/moments/comments/author 留在原注册路径，保持 bit-exact。

---

## 3. 术语

- **插件（plugin）**：一个自包含业务模块，Go 侧实现 `Plugin` 接口，可选携带 `src/<name>/**` 页面源码。编译期进二进制，启动期按清单启停。
- **内核（kernel）**：`build/plugin` 包，负责清单校验、排序、启停、能力面装配。
- **能力面（Runtime）**：内核递给插件的受控依赖集合（窄接口，非裸传一切）。
- **页面插件**：带前端页面的插件；页面属源码级模块（Node 构建期入 bundle），这是框架既定事实，不是本设计引入的限制。

---

## 4. 分期策略

| 阶段 | 内容 | 决策门 |
|---|---|---|
| **Phase 1（本 unit）** | 编译期插件集合 + 启动期清单校验 + settings 启停 + 生命周期 + MCP 扩展点 | docs 插件（unit-7）完整跑通 |
| **Phase 2（独立 unit，暂缓）** | 运行时动态加载评估：yaegi（Go 解释器）/ wazero（WASM 沙箱）/ go-plugin（子进程 gRPC）对比矩阵 | Phase 1 契约稳定 **且** 出现真实的"不改代码装插件"需求 |

> 诚实说明：Phase 1 的"动态"= **启动期可启停、可配置**，不是运行期热插拔。Node 页面路由的构建期事实决定了完全热插拔在 VenHybird 架构下不可达（除非连 SSR bundle 一起动态化，那已是另一个量级的工程）。

---

## 5. 契约设计（Phase 1）

### 5.1 插件接口（`build/plugin/plugin.go`）

```go
package plugin

// Meta 插件元数据（Register 前由内核校验）。
type Meta struct {
    Name        string   // 唯一，kebab-case，如 "docs"
    Version     string   // semver
    Description string
    PagePrefix  []string // 声明占用的页面路由前缀（如 "/docs"），内核做冲突校验
    Depends     []string // 依赖的插件名（Phase 1 仅用于拓扑排序，不做版本区间）
    DefaultOn   bool     // 未配置 settings 开关时的默认启停
}

// Runtime 受控能力面：内核装配，窄接口注入（对齐 unit-2 KeyAuthenticator 先例）。
// 禁止塞 *sql.DB、具体 Service 指针等宽依赖。
type Runtime struct {
    App        *hybrid.App      // 框架注册面（Page/StaticPage/API/Role）
    MCP        MCPRegistry      // 注册 mcp action（见 5.2）
    Search     SearchRegistry   // 注册搜索 provider（见 5.4）
    Settings   SettingsStore    // 插件配置命名空间读写（plugin.<name>.*）
    DataChange DataChangeFunc   // 失效辅助（包装 ChangeEvent 构造）
    Logger     *log.Logger
}

// Plugin 插件契约。Start/Stop 为可选能力（拆为 Optionalet 接口，避免空实现泛滥）。
type Plugin interface {
    Meta() Meta
    Register(rt *Runtime) error // 声明页面/API/MCP action；只做注册，不做 IO 副作用
}

type Startable interface { Start() error }  // worker 类插件实现（如 moderator 先例：goroutine + ticker）
type Stoppable interface { Stop() error }   // 优雅关停；main.go 退出前逆序调用
```

### 5.2 MCP 扩展点（改造 `interfaces/mcp.go`）

现状：`mcpActions` 为包级 map + `MCP` 结构体闭集持依赖。

改造（**零行为变更**为硬约束）：

1. `mcpActions` 包级 map 保留为**内置 action 表**，运行期只读不变；
2. `MCP` 实例新增二级注册表 `extra map[string]mcpActionFunc` + 互斥锁，暴露：

```go
// RegisterAction 注册插件贡献的 action（名称冲突返回 error，内核 fail-fast）。
// 命名规范：必须以 "<plugin-name>." 开头（如 "doc.create"），内核校验。
func (m *MCP) RegisterAction(name string, fn mcpActionFunc) error
```

3. dispatch 顺序：内置表先查、`extra` 后查（重名在注册期即被拒，运行期无歧义）；
4. `RegisterMCP` 签名增加返回 `*MCP`（或经 `MCPRegistry` 接口交给内核），供插件注册 action；**调用时机移到插件注册之后**（register.go 链尾位置不变，仅内部顺序调整）。

### 5.3 目录与入口约定（已拍板：单入口文件 + 主体执行函数）

每个插件只有**一个规范文件** `plugin.go` 作为对外接口面：包含 `Meta` 与入口函数 `New()`；目录内其余文件自由组织（实现细节，不作规范）。

```
frame/go/build/plugins/<name>/
├── plugin.go      # 唯一规范文件：func New() plugin.Plugin（入口）——纯构造，无副作用
└── ...            # 实现文件自由命名（domain/repo/mcp/webhook 等，非规范）

src/<name>/**/page.tsx             # 插件页面（构建期入 SSR bundle）
```

主体（`register.go`）只写**一个执行函数**，函数体即插件清单，每插件一行、只调用一次入口：

```go
// register.go —— 主体执行函数：新增插件 = 清单加一行
func registerPlugins(rt *plugin.Runtime) error {
    return plugin.Bootstrap([]func() plugin.Plugin{
        docs.New, // plugins/docs
    }, rt)
}
```

规则：

- **`New()` 必须纯构造**（无 IO、无全局副作用），接线一律发生在 `Register(rt)` 内——内核时序是"构造 → 读 Meta → 查 settings 启停 → 才调 Register"；若 New 自带副作用，插件会在开关判定前半启动，启停治理失效；
- 内核经清单列表控制拓扑序与启停；**不做 init() 自注册、不做反射扫描**（与仓库显式装配惯例一致）；
- 与存量 DDD 四层目录的关系：插件取**内聚优先**（一个插件可整体摘除）；存量模块不动，两种布局并存，新插件一律走 `plugins/`。

### 5.4 搜索扩展点（SearchRegistry，2026-09-16 增补）

```go
// SearchProvider 搜索结果贡献者：插件经 Runtime.Search 注册（查询接口带 ctx——请求作用域超时，与 Register 无 ctx 不冲突）。
type SearchHit struct {
    Title, Summary, URL string
    UpdatedAt           time.Time
}
type SearchProvider interface {
    Name() string // provider 名 = 插件名（如 "docs"），同时是对外 scope 取值
    Search(ctx context.Context, q string, limit int) ([]SearchHit, error)
}
```

- 存量 posts 搜索包装为内置 provider，名 `blog`（行为不变）；
- scope 取值空间 = `all`（默认，合并全部已启用 provider）+ 各 provider 名（`blog` / `docs` / 未来插件）；
- 聚合器落在存量 `interfaces/search.go`：合并、排序、封顶 N；
- **bit-exact 约束**：无插件注册 provider 时，`scope=all` 等价现状（仅 blog）；scope 参数缺省即 all。

---

## 6. 注册内核流程

`register.go` 改造后的启动序列：

```
1. registerRoles(a)                    ← 不变（内核之前）
2. 基础设施：DB/仓储/加密/应用服务       ← 不变（存量模块照旧手工装配）
3. 存量接口注册（20+ RegisterXxx）      ← 不变
4. registerPlugins(rt) → plugin.Bootstrap(清单)  ← 新增（§5.3 主体执行函数，每插件一行入口）：
   a. 清单校验：Name 唯一 / PagePrefix 两两不重叠且不与内置路由表冲突 / Depends 成环检测+拓扑排序
   b. 启停判定：settings key `plugin.<name>.enabled`，缺省取 Meta.DefaultOn（boot 读一次，改后重启生效）
   c. 按（拓扑序 + 注册序）逐个调 Plugin.Register(rt)，任一 error 即启动失败（fail-fast）
5. RegisterMCP(...)                     ← 移到插件注册后，聚合 extra actions
6. 各 Startable 插件 Start()            ← moderator 先例：goroutine 不阻塞
7. a.Listen(addr)                       ← 不变（fallback 仍由 Listen 强制最后）
关停：App.Close() 后逆序调 Stoppable.Stop()
```

### 治理规则（内核强制）

| 规则 | 违约后果 |
|---|---|
| 路由前缀所有权：插件 PagePrefix 与内置路由/其他插件冲突 | 启动失败（防静默劫持） |
| MCP action 命名必须 `<plugin>.` 前缀 | 注册期拒绝 |
| 写 action 必须声明失效 pattern（unit-2 §6 契约表纪律） | 评审清单项（devflow-review 专项检查） |
| Runtime 只给窄接口；internal 类型不泄漏（AGENTS.md 红线） | 评审 P0 |
| settings 命名空间 `plugin.<name>.*` 自管 | 越界访问 = 评审 P1 |
| 插件间通信：只允许经 Depends 声明的服务接口，禁止全局单例 | 评审 P1 |
| 外部资源自治：插件自建 DB 连接等资源（不走主业务 persistence 包），`Stop()` 必须释放；连接池上限约定 MaxOpenConns ≤ 4 | Stop 不释放 = P0；池超限/越界 import 主业务包 = P1 |

---

## 7. 安全与信任模型（Phase 1）

- 插件与宿主**同进程同权限**——信任边界 = 编译期代码审计（谁能合入 PR 谁就能写任意业务代码，这与现状一致，不劣化）。
- Phase 2 若引入 WASM/子进程才谈沙箱、能力授权。Phase 1 的窄 Runtime 是**架构治理**手段（防漂移），不是安全边界。

---

## 8. 测试与验收标准

1. **内核单测**：清单冲突（重名/前缀重叠/依赖成环）→ 启动失败；启停开关生效；拓扑序正确；Stop 逆序。
2. **契约测试插件**：一个 echo 测试插件走完 Register→MCP action 注册→搜索 provider 注册→Start→Stop 全流程。
3. **回归硬指标**：所有插件关闭时，启动行为与改造前 **bit-exact**（14 个内置 action、路由表、失效行为零变化）。
4. **真实消费者验收**：docs 插件（unit-7）完全经内核注册，不修改 register.go 主体与 mcp.go 内置区。
5. 常规门禁：`go build ./... && go vet ./... && go test ./...` 全绿。

---

## 9. Phase 2 展望（仅备忘，不承诺）

| 方案 | 优势 | 劣势 | 初判 |
|---|---|---|---|
| yaegi（Go 解释器） | 插件写 Go；无 ABI 问题 | 性能损耗；stdlib 覆盖不全；调试难 | 备选 |
| wazero（WASM） | 沙箱隔离；语言无关 | ABI 设计成本高（proxy-wasm 风格）；生态新 | 若要隔离则首选 |
| go-plugin（子进程） | 进程隔离成熟 | 部署重（博客场景过重）；需 gRPC | 不倾向 |

决策门：Phase 1 契约稳定 + 出现真实需求（如第三方想在不改源码情况下装插件）。

---

## 10. 上游框架需求（随本设计一并提出）

**需求 7：catch-all 页面路由**（docs 插件硬依赖，unit-7 详述）：

- Node 侧：`[...slug]` 目录推导为 `/docs/*` 通配 pattern；`matchRoute` 支持尾段通配（段数不等时尾部 `*` 吸收）；
- Go 侧：`isr.ParseDeclaration`/`Declaration.Match` 支持尾段 `*`；fiber `StaticPage` 注册 `/docs/*` 与 ISR 物化路径映射；
- 兜底方案（上游未合入前的过渡）：注册固定深度 `/docs/:a`、`/docs/:a/:b`、`/docs/:a/:b/:c` 三条声明，树深上限 3——丑但零框架改动。

---

## 11. 设计决议（2026-09-16 用户拍板）

1. **`Plugin.Register` 不带 `context.Context`**——Phase 1 无异步注册场景（YAGNI）。
2. **启停粒度只做整插件一层**：settings 键 `plugin.<name>.enabled` 一刀切；插件内部的功能级开关（如 docs 的 webhook on/off）由插件在自己的 `plugin.<name>.*` 命名空间自管，内核不感知。
3. **目录与入口规范：单入口文件 + 主体执行函数**——每插件唯一规范文件 `plugin.go`（入口 `New()` 纯构造、无副作用），实现文件自由组织；主体 `register.go` 写一个执行函数，清单每插件一行、只调用一次入口（详见 §5.3）。附带约束：`New()` 纯构造是启停时序（构造 → 读 Meta → 查开关 → 才 Register）成立的前提，评审作为 P0 检查项。
