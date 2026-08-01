// 认证接口：登录/注册/注销（挂在 server 原生 fiber 上，不走 /api 前缀，不做角色守卫）。
package interfaces

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
)

// credentialsInput 是登录/注册的请求体（注册需带邮箱——验证码登录依赖）。
type credentialsInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// RegisterAuth 注册认证接口。
func RegisterAuth(a *hybrid.App, users *userapp.Service) {
	// 全站受 role 守卫的页面仅 /admin/*（博客页面均公开，登录走导航弹窗与 /login 落地页），
	// 故 401 跳转统一指向后台独立登入页
	a.SetLoginRedirect("/admin/login")

	server := a.Server()
	server.App().Post("/auth/login", func(ctx *fiber.Ctx) error {
		var in credentialsInput
		if err := ctx.BodyParser(&in); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
		}
		u, err := users.Authenticate(in.Username, in.Password)
		if errors.Is(err, userapp.ErrInvalidCredentials) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return grantAuthJSON(ctx, a, u)
	})
	server.App().Post("/auth/register", func(ctx *fiber.Ctx) error {
		var in credentialsInput
		if err := ctx.BodyParser(&in); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
		}
		u, err := users.Register(in.Username, in.Password, in.Email)
		var vErr *userapp.ValidationError
		switch {
		case errors.As(err, &vErr):
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": vErr.Message})
		case errors.Is(err, user.ErrUsernameTaken):
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "username taken"})
		case errors.Is(err, user.ErrEmailTaken):
			return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email taken"})
		case err != nil:
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		// 注册即登录
		return grantAuthJSON(ctx, a, u)
	})
	server.App().Post("/auth/logout", func(ctx *fiber.Ctx) error {
		server.RevokeAuth(ctx)
		return ctx.SendStatus(fiber.StatusNoContent)
	})
}

// grantAuthJSON 下发双 cookie（会话携带用户 ID）并返回角色。
func grantAuthJSON(ctx *fiber.Ctx, a *hybrid.App, u *user.User) error {
	if err := a.Server().GrantAuthWithUser(ctx, u.Role.String(), strconv.FormatInt(u.ID, 10)); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(fiber.Map{"ok": true, "role": u.Role.String()})
}
