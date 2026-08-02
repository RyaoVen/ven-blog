// API 密钥管理接口：后台生成 / 列表 / 吊销（cookie 鉴权，author 守卫）。
// 注意：这是后台管理面；程序化鉴权消费方是 Unit 2 的 /api/mcp 网关（调 apikeyapp.AuthenticateKey）。
package interfaces

import (
	"errors"

	"ven_hybird/build/application/apikeyapp"
	"ven_hybird/build/domain/apikey"
	"ven_hybird/hybrid"
)

// RegisterKeysAdmin 注册 API 密钥管理接口（cookie 鉴权，author 守卫）。
func RegisterKeysAdmin(a *hybrid.App, keys *apikeyapp.Service) error {
	admin := []string{"author"}

	// 生成密钥（明文仅此一次返回，调用方展示后丢弃）
	if err := a.Post("/admin/keys", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		raw, view, err := keys.CreateKey(userID, in.Name)
		var vErr *apikeyapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{
			"key":     raw,
			"warning": "密钥明文仅此一次展示，关闭后无法再次查看；请立即复制妥善保存。",
			"view":    view,
		})
	}); err != nil {
		return err
	}

	// 密钥列表（脱敏视图，永不含明文）
	if err := a.Get("/admin/keys", admin, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		views, err := keys.ListKeys(userID)
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"keys": views})
	}); err != nil {
		return err
	}

	// 吊销（即时生效；不存在 / 已吊销 / 非本人统一 404，不泄露存在性）
	return a.Delete("/admin/keys/:id", admin, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		id, err := parseID(c.Param("id"))
		if err != nil {
			return c.Error(400, "invalid key id")
		}
		if err := keys.Revoke(userID, id); err != nil {
			if errors.Is(err, apikey.ErrNotFound) {
				return c.Error(404, "api key not found")
			}
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	})
}
