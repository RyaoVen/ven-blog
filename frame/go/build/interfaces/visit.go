// 访问统计上报接口：SPA 导航成功后公开上报（双埋点之一，另一路是 Go 网关中间件）。
package interfaces

import (
	"errors"
	"time"

	"ven_hybird/build/application/visitapp"
	"ven_hybird/hybrid"

	"github.com/gofiber/fiber/v2"
)

// RegisterVisitAPI 注册 POST /api/visit（公开，无需登录）。
// body {"path":"..."}；校验：/ 开头、无 query/片段、≤255 字符；
// 30s 内同 path 去重（visitapp 服务内简单内存节流），重复上报 200 静默吞掉。
func RegisterVisitAPI(a *hybrid.App, visits *visitapp.Service) error {
	return a.Post("/visit", nil, func(c *hybrid.ApiCtx) error {
		var body struct {
			Path string `json:"path"`
		}
		if err := c.Bind(&body); err != nil {
			return c.Error(fiber.StatusBadRequest, "invalid body")
		}
		if err := visits.Report(time.Now(), body.Path); err != nil {
			if errors.Is(err, visitapp.ErrInvalidPath) {
				return c.Error(fiber.StatusBadRequest, "invalid path")
			}
			return c.Error(fiber.StatusInternalServerError, "record failed")
		}
		return c.JSON(fiber.StatusOK, fiber.Map{"ok": true})
	})
}
