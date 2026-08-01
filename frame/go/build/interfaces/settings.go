// 站点设置接口：设置页 + 管理 API（全部 author 守卫）。
package interfaces

import (
	"errors"
	"strconv"

	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/hybrid"
)

// RegisterSettings 注册设置页与设置 API。
func RegisterSettings(a *hybrid.App, settings *settingsapp.Service, users *userapp.Service) error {
	admin := []string{"author"}

	// 设置页（含当前作者资料）
	if err := a.Page("/admin/settings", admin, func(c *hybrid.PageCtx) error {
		content, err := settings.Content()
		if err != nil {
			return err
		}
		moderation, err := settings.Moderation()
		if err != nil {
			return err
		}
		categories, err := settings.Categories()
		if err != nil {
			return err
		}
		profile := map[string]any{"username": "", "bio": "", "avatarUrl": ""}
		if userID, _, ok := c.User(); ok {
			if uid, parseErr := strconv.ParseInt(userID, 10, 64); parseErr == nil {
				if u, findErr := users.FindByID(uid); findErr == nil {
					profile = map[string]any{"username": u.Username, "bio": u.Bio, "avatarUrl": u.AvatarURL}
				}
			}
		}
		return c.JSON(map[string]any{
			"content":    content,
			"moderation": moderation,
			"categories": categories,
			"profile":    profile,
		})
	}); err != nil {
		return err
	}

	// 保存内容配置
	if err := a.Put("/admin/settings/content", admin, func(c *hybrid.ApiCtx) error {
		var in settingsapp.Content
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if err := settings.SetContent(&in); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 保存文章分类
	if err := a.Put("/admin/settings/categories", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Categories []string `json:"categories"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if err := settings.SetCategories(in.Categories); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 评论审核开关
	if err := a.Put("/admin/settings/moderation", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			On bool `json:"on"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if err := settings.SetModeration(in.On); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true, "on": in.On})
	}); err != nil {
		return err
	}

	// 修改密码（校验旧密码）
	if err := a.Post("/admin/settings/password", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		err = users.ChangePassword(userID, in.OldPassword, in.NewPassword)
		var vErr *userapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 保存作者资料（bio/头像 URL，头像经 /api/upload 上传后写入）
	return a.Put("/admin/settings/profile", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Bio       string `json:"bio"`
			AvatarURL string `json:"avatarUrl"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		err = users.UpdateProfile(userID, in.Bio, in.AvatarURL)
		var vErr *userapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		// 资料展示在首页/作者主页（动态页），失效刷新 + SSE
		a.InvalidatePage("/")
		a.InvalidatePage("/author/"+usernameOf(a, users, userID))
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// usernameOf 取用户名（失效作者主页用；失败时静默为空串不致命）。
func usernameOf(a *hybrid.App, users *userapp.Service, userID int64) string {
	u, err := users.FindByID(userID)
	if err != nil {
		return ""
	}
	return u.Username
}
