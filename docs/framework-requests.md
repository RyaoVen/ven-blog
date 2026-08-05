# 框架需求 Brief（转交框架仓 agent）

> 来源：博客业务仓（awesomeProject1）与框架仓（ven_hybird）对齐检查 + 安全评审遗留项。
> **第一批 2 项（no-cache #67、visit 埋点上提 #68）已由框架侧吸收并合并，业务仓已同步对齐。**
> 本 Brief 现含**第二批 4 项**，均为框架侧还没有、需要上提的改动。请按 gh-workflow 流程处理（拆 issue → 分支 → 小步提交 → PR 合并）。

---

## 需求 3：pagecache flight 等待无超时

**位置**：`frame/go/internal/pagecache/flight.go`。

**问题**：同 key 并发仅一个 leader 回源，follower 等待 `done` channel 无超时。若 leader 的回源路径异常（panic 未 complete、代码路径遗漏），follower 将永久阻塞挂起请求。

**期望**：follower 等待带超时（建议复用调用方传入的超时或常量兜底，如 30s）；leader 侧保证 `complete` 必然执行（defer 兜底）；超时后 follower 走自身回源或返回错误，绝不无限等待。

**验收**：新增测试——leader 不 complete 时 follower 超时后返回（或回源），不永久挂起；既有 pagecache 测试全过。

## 需求 4：page_proxy 渲染超时 timer 泄漏

**位置**：`frame/go/internal/httpserver/page_proxy.go:213` `case <-time.After(s.config.RenderTimeout)`。

**问题**：`time.After` 创建的 timer 在超时分支命中后不回收，需等超时时长才 GC——每次请求泄漏一个 timer（高并发下放大）。

**期望**：改用 `time.NewTimer` + `defer timer.Stop()`（标准写法），行为不变。

**验收**：`go vet` 通过；渲染超时行为与现有一致（超时返回 504/兜底路径）。

## 需求 5：StaticPage 不支持角色鉴权

**位置**：`frame/go/hybrid/staticPage.go:23` `func (a *App) StaticPage(dynamicUrl string, maxPages int, smartLoad bool, h PageHandler) error`。

**问题**：`Page` 支持 `roles` 声明鉴权，`StaticPage` 签名没有 roles 参数——静态页（ISR 物化）无法声明鉴权，只能公开或靠业务 handler 自行校验。

**期望**：`StaticPage` 增加 `roles []string` 参数（与 `Page` 一致），物化与直发路径都做鉴权；保持向后兼容可接受（如新增 `StaticPageWithRoles` 或改签名时同步更新框架内调用点与测试）。

**验收**：声明 roles 的 StaticPage 未登录被 401/302 拦截；公开（nil roles）行为不变；ISR 直发路径同样生效。

## 需求 6：Node worker 内部 token 比较非 timing-safe

**位置**：`frame/node/http-transport/httpController.ts:156` `return headers["x-ven-internal-token"] === this.options.internalToken`。

**问题**：字符串 `===` 比较存在时序侧信道，理论可逐字符猜解内部 token。

**期望**：用 `crypto.timingSafeEqual`（`Buffer.from` 等长比较，长度不等直接 false）；`internalToken` 未配置时保持现状（直接放行，本地开发）。

**验收**：新增单测——正确 token 通过、错误 token 拒绝、长度不同拒绝、空/未配置放行；既有 node 测试全过。

---

## 其他说明

- 框架仓本地工作区如有未提交改动，请先整理提交或分支隔离，本批需求独立成 issue/分支，避免混入无关改动。
- 业务仓侧待框架吸收后自行同步对齐（当前无业务侧联动改动），无需框架侧处理。


## 需求 1：SSR 页面响应加 Cache-Control no-cache

**背景**：SSR 页面内容每次渲染可变（文章/评论/动态更新后重新渲染）。若浏览器或中间缓存层缓存旧响应，部署后用户仍看到旧页面（业务仓曾因此踩坑，是真实线上 bug 的修复）。与 SSE 响应同策略（同样 no-cache）。

**位置**：`frame/go/internal/httpserver/page_proxy.go`，共 2 处（SSR 渲染成功响应 + 回退路径各一处）。

**改动内容**（业务仓现状，可直接吸收）：

```diff
 // SSR 渲染成功路径
+	// SSR 页面内容每次渲染可变：no-cache 防止浏览器/中间层缓存部署前的旧页面（与 SSE 同策略）
+	ctx.Set(fiber.HeaderCacheControl, "no-cache")
```

（第二处同样一行，在另一渲染出口前。）

**验收**：`GET /` 响应头含 `Cache-Control: no-cache`；data-only 取数请求与 API 不受影响；既有 httpserver 测试全过。

---

## 需求 2：页面访问统计埋点——上提为框架通用能力

**背景**：业务方（博客）需要页面浏览计数（访问统计功能）。业务仓当前实现把埋点中间件写在了框架层 `internal/httpserver/visit.go`，且注入回调走 `hybrid.App.Server()`（返回 `*httpserver.Server` 的基线遗留通道）——属于"业务功能放框架层 + 钻洞注入"。请上提为框架正式能力，业务侧只调用公开 API。

**期望设计**（照此实现，参考实现见下节）：

1. `hybrid.App` 提供公开注册方法，如 `func (a *App) SetVisitRecorder(fn func(path string))`（nil = 关闭埋点）。**不新增也不依赖** `App.Server()` 这类暴露 internal 类型的通道（约束：internal 的类型不暴露到 hybrid 公开签名）。
2. `internal/httpserver` 实现埋点中间件，挂在**最外层 Use**（先于 ISR 物化直发与业务路由，ISR 直发也计数）。
3. 计数规则：
   - 仅 `GET` 请求；
   - 跳过 data-only 取数请求（请求头 `X-Ven-Data-Only` 非空——SPA 内部取数，页面浏览由前端导航后上报，避免双埋点）；
   - 跳过非页面路径前缀：`/api`、`/assets`、`/auth`、`/_internal`、`/images`、`/healthz`、`/mcp`（业务仓当前清单，可微调）。
4. 回调在请求 goroutine 同步调用，**必须快速返回**；回调 panic/错误由埋点侧兜住或由业务侧吞掉，绝不阻断页面响应。
5. 注入时机：启动期注册完成前注入，运行期只读——字段用 `sync.RWMutex` 保护（或启动后不可变，二选一，前者与业务仓现状一致）。

**参考实现**（业务仓现状 `internal/httpserver/visit.go` 全文，可直接吸收；吸收后业务仓删除本地副本改走公开 API）：

```go
// 访问统计埋点中间件：最外层 Use，ISR 直发也计数。
package httpserver

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// dataOnlyHeader 是 SPA 路由取数请求头（与 hybrid 层同值）。
// 此类请求是 SPA 内部 data-only 取数，页面浏览由前端 navigate 成功后上报，
// 网关不再重复计数，避免一次 SPA 导航被双埋点各计一次。
const dataOnlyHeader = "X-Ven-Data-Only"

// visitExcludedPrefixes 非页面路径前缀白名单（埋点跳过）：
// 接口/静态资源/鉴权/内部端点/图片/健康检查等都不算页面浏览。
var visitExcludedPrefixes = []string{
	"/api", "/assets", "/auth", "/_internal", "/images", "/healthz", "/mcp",
}

// SetVisitRecorder 设置访问统计埋点回调（业务层注入；nil = 关闭埋点）。
// 回调在请求 goroutine 中同步调用，必须快速返回；失败由业务层自行吞掉（不影响页面响应）。
func (s *Server) SetVisitRecorder(fn func(path string)) {
	s.visitMu.Lock()
	s.visitRec = fn
	s.visitMu.Unlock()
}

// visitTracking 访问统计埋点中间件：仅 GET 页面请求计数。
// 挂在最外层 Use：先于 ISR 物化直发与页面兜底，所有页面流量（含 ISR 直发）都经过；
// 埋点回调失败静默，绝不阻断请求。
func (s *Server) visitTracking() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		s.visitMu.RLock()
		rec := s.visitRec
		s.visitMu.RUnlock()
		if rec != nil &&
			ctx.Method() == fiber.MethodGet &&
			ctx.Get(dataOnlyHeader) == "" &&
			!isExcludedVisitPath(ctx.Path()) {
			rec(ctx.Path())
		}
		return ctx.Next()
	}
}

// isExcludedVisitPath 判断路径是否命中非页面前缀白名单。
func isExcludedVisitPath(path string) bool {
	for _, p := range visitExcludedPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
```

`server.go` 侧接入（业务仓现状，吸收时按 internal 编码风格落位）：

```go
// Server 结构体字段：
visitMu  sync.RWMutex      // 保护 visitRec（埋点回调启动期注入，运行期只读）
visitRec func(path string) // 访问统计埋点回调（业务层注入；nil = 关闭埋点）

// 中间件注册（最外层，先于 ISR 直发与业务路由）：
app.Use(s.visitTracking())
```

**验收**：
- `hybrid.App.SetVisitRecorder` 可从业务组装根调用，不暴露 internal 类型；
- 埋点中间件对 GET 页面计数、跳过 data-only/非页面前缀；ISR 直发计数；
- 回调 panic 不拖垮请求（可加恢复或由调用方保证，建议中间件内 `defer recover` 兜底）；
- 既有 httpserver/hybrid 测试全过。

---

## 其他说明

- 框架仓本地工作区已有未提交改动（事件总线 #21 等），请先整理提交或分支隔离，本 Brief 的两个需求独立成 issue/分支，避免混入无关改动。
- 业务仓侧待框架吸收后自行对齐（删除本地 visit.go、改用公开 API），无需框架侧处理。
