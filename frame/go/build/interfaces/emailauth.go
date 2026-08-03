// 邮箱认证接口：验证码签发/登录 + 绑定邮箱（注册含邮箱字段）。
package interfaces

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/application/emailauth"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
)

// emailCodeInput 签发验证码请求体。
type emailCodeInput struct {
	Email string `json:"email"`
}

// emailLoginInput 验证码登录请求体。
type emailLoginInput struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// RegisterEmailAuth 注册邮箱认证接口（raw fiber，不走 /api 前缀）。
func RegisterEmailAuth(a *hybrid.App, emailAuthSvc *emailauth.Service, users *userapp.Service, settings *settingsapp.Service) {
	server := a.Server()

	// 签发验证码（不泄露邮箱是否注册）；受用户注册登录开关约束，关闭时 403
	server.App().Post("/auth/email/code", func(ctx *fiber.Ctx) error {
		if enabled, err := settings.AuthEnabled(); err != nil || !enabled {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "login disabled"})
		}
		var in emailCodeInput
		if err := ctx.BodyParser(&in); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
		}
		err := emailAuthSvc.RequestCode(in.Email)
		var vErr *emailauth.ValidationError
		if errors.As(err, &vErr) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": vErr.Message})
		}
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return ctx.JSON(fiber.Map{"ok": true})
	})

	// 验证码登录（命中即下发双 cookie）；受用户注册登录开关约束，关闭时 403
	server.App().Post("/auth/email/login", func(ctx *fiber.Ctx) error {
		if enabled, err := settings.AuthEnabled(); err != nil || !enabled {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "login disabled"})
		}
		var in emailLoginInput
		if err := ctx.BodyParser(&in); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
		}
		u, err := emailAuthSvc.Verify(in.Email, in.Code)
		if errors.Is(err, emailauth.ErrInvalidCode) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired code"})
		}
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return grantAuthJSON(ctx, a, u)
	})
}

// bindEmailInput 绑定邮箱请求体。
type bindEmailInput struct {
	Email string `json:"email"`
}

// RegisterMeEmail 注册"绑定/修改邮箱"接口（登录用户，唯一性 409）。
func RegisterMeEmail(a *hybrid.App, users *userapp.Service) error {
	return a.Put("/me/email", loginRoles, func(c *hybrid.ApiCtx) error {
		var in bindEmailInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if msg := user.ValidateEmail(in.Email); msg != "" {
			return c.Error(400, msg)
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		err = users.UpdateEmail(userID, in.Email)
		switch {
		case errors.Is(err, user.ErrEmailTaken):
			return c.Error(409, "email taken")
		case err != nil:
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	})
}
