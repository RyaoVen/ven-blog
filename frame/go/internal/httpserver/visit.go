// 访问统计埋点中间件：最外层 Use，ISR 直发也计数。
package httpserver

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// dataOnlyHeader 是 SPA 路由取数请求头（与 hybrid 层同值）。
// 此类请求是 SPA 内部 data-only 取数，页面浏览由前端 navigate 成功后 POST /api/visit 上报，
// 网关不再重复计数，避免一次 SPA 导航被双埋点各计一次。
const dataOnlyHeader = "X-Ven-Data-Only"

// visitExcludedPrefixes 非页面路径前缀白名单（埋点跳过）：
// 接口/静态资源/鉴权/内部端点/图片/健康检查/MCP 等都不算页面浏览。
var visitExcludedPrefixes = []string{
	"/api", "/assets", "/auth", "/_internal", "/images", "/healthz", "/mcp",
}

// SetVisitRecorder 设置访问统计埋点回调（业务层经组装根注入；nil = 关闭埋点）。
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
