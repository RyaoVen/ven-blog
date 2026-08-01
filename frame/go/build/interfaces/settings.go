// 站点设置接口：设置页 + 管理 API（全部 author 守卫）。
package interfaces

import (
	"errors"
	"strconv"

	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/user"
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
		host, port, smtpUser, _, fromName, err := settings.SMTPConfig()
		if err != nil {
			return err
		}
		authorEmail, err := settings.AuthorEmail()
		if err != nil {
			return err
		}
		siteIcon, err := settings.SiteIcon()
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
			"siteIcon":   siteIcon,
			"email": map[string]any{
				"host": host, "port": port, "user": smtpUser,
				"fromName": fromName, "passwordSet": host != "" || smtpUser != "",
				"authorEmail": authorEmail,
			},
		})
	}); err != nil {
		return err
	}

	// 保存内容配置（首页句子；项目与展示柜移至 /admin/author 编辑）
	if err := a.Put("/admin/settings/content", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Quotes []settingsapp.Quote `json:"quotes"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if err := settings.SetQuotes(in.Quotes); err != nil {
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/")
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

	// 邮箱配置（SMTP + 作者个人邮箱；作者邮箱同步写入 author 账号）
	if err := a.Put("/admin/settings/email", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Host        string `json:"host"`
			Port        string `json:"port"`
			User        string `json:"user"`
			Password    string `json:"password"`
			FromName    string `json:"fromName"`
			AuthorEmail string `json:"authorEmail"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if in.AuthorEmail != "" {
			if msg := user.ValidateEmail(in.AuthorEmail); msg != "" {
				return c.Error(400, msg)
			}
		}
		if err := settings.SetSMTP(in.Host, in.Port, in.User, in.Password, in.FromName); err != nil {
			return c.Error(500, "internal error")
		}
		if err := settings.SetAuthorEmail(in.AuthorEmail); err != nil {
			return c.Error(500, "internal error")
		}
		// 作者个人邮箱同步进 author 账号（验证码登录与 @ 通知依赖）
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		if in.AuthorEmail != "" {
			if err := users.UpdateEmail(userID, in.AuthorEmail); err != nil {
				return c.Error(500, "internal error")
			}
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 保存站点图标（导航品牌标 + favicon，前端经 /api/site 现取，无需失效页面）
	if err := a.Put("/admin/settings/site", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Icon string `json:"icon"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if err := settings.SetSiteIcon(in.Icon); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 修改作者用户名（改名后新旧作者主页与首页作者卡都需失效刷新）
	if err := a.Put("/admin/settings/username", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Username string `json:"username"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		old, err := users.FindByID(userID)
		if err != nil {
			return c.Error(500, "internal error")
		}
		err = users.UpdateUsername(userID, in.Username)
		var vErr *userapp.ValidationError
		switch {
		case errors.As(err, &vErr):
			return c.Error(400, vErr.Message)
		case errors.Is(err, user.ErrUsernameTaken):
			return c.Error(409, "username taken")
		case err != nil:
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/")
		a.InvalidatePage("/author/" + old.Username)
		a.InvalidatePage("/author/" + in.Username)
		return c.JSON(200, map[string]any{"ok": true, "username": in.Username})
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
