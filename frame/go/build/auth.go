// 登录与注销：校验内存用户表，通过后 GrantAuth 下发双 cookie（ven_auth/ven_role）。
package build

import (
	"github.com/gofiber/fiber/v2"

	"ven_hybird/hybrid"
)

// loginInput 是登录请求体。
type loginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// registerAuth 注册登录/注销接口（挂在 server 原生 fiber 上，不走 /api 前缀，不做角色守卫）。
func registerAuth(a *hybrid.App, s *blogStore) {
	a.SetLoginRedirect("/login")

	server := a.Server()
	server.App().Post("/auth/login", func(ctx *fiber.Ctx) error {
		var in loginInput
		if err := ctx.BodyParser(&in); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
		}
		u, ok := s.authenticate(in.Username, in.Password)
		if !ok {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		if err := server.GrantAuth(ctx, u.Role); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(fiber.Map{"ok": true, "role": u.Role})
	})
	server.App().Post("/auth/logout", func(ctx *fiber.Ctx) error {
		server.RevokeAuth(ctx)
		return ctx.SendStatus(fiber.StatusNoContent)
	})
}
