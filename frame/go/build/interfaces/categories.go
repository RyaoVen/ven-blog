// 分类管理接口：增、改、删（非空一键迁移），全部 author 守卫。
package interfaces

import (
	"net/url"
	"strings"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/hybrid"
)

// categoryInput 新增/改名请求体。
type categoryInput struct {
	Name string `json:"name"`
}

// RegisterCategories 注册分类管理 API（/api/admin/categories*）。
func RegisterCategories(a *hybrid.App, posts *postapp.Service, settings *settingsapp.Service) error {
	admin := []string{"author"}

	// 新增分类
	if err := a.Post("/admin/categories", admin, func(c *hybrid.ApiCtx) error {
		var in categoryInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		name := strings.TrimSpace(in.Name)
		if name == "" || len([]rune(name)) > 24 {
			return c.Error(400, "category name must be 1-24 characters")
		}
		categories, err := settings.Categories()
		if err != nil {
			return c.Error(500, "internal error")
		}
		for _, existing := range categories {
			if existing == name {
				return c.Error(409, "category exists")
			}
		}
		if err := settings.SetCategories(append(categories, name)); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(201, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 改名（迁移文章 + 同步列表）
	if err := a.Put("/admin/categories/:name", admin, func(c *hybrid.ApiCtx) error {
		oldName := pathName(c)
		var in categoryInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		newName := strings.TrimSpace(in.Name)
		if newName == "" || len([]rune(newName)) > 24 {
			return c.Error(400, "category name must be 1-24 characters")
		}
		categories, err := settings.Categories()
		if err != nil {
			return c.Error(500, "internal error")
		}
		found := false
		for i, existing := range categories {
			if existing == newName && newName != oldName {
				return c.Error(409, "category exists")
			}
			if existing == oldName {
				categories[i] = newName
				found = true
			}
		}
		if !found {
			return c.Error(404, "category not found")
		}
		if oldName != newName {
			if err := posts.MigrateCategory(oldName, newName); err != nil {
				return c.Error(500, "internal error")
			}
		}
		if err := settings.SetCategories(categories); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 删除分类：非空且无 migrateTo 时 409 带 count；带 migrateTo 时一键迁移后删除
	return a.Delete("/admin/categories/:name", admin, func(c *hybrid.ApiCtx) error {
		name := pathName(c)
		migrateTo := queryString(c, "migrateTo")
		categories, err := settings.Categories()
		if err != nil {
			return c.Error(500, "internal error")
		}
		found := false
		rest := make([]string, 0, len(categories))
		for _, existing := range categories {
			if existing == name {
				found = true
				continue
			}
			rest = append(rest, existing)
		}
		if !found {
			return c.Error(404, "category not found")
		}
		count, err := posts.CategoryCount(name)
		if err != nil {
			return c.Error(500, "internal error")
		}
		if count > 0 {
			if migrateTo == "" {
				return c.JSON(409, map[string]any{"error": "category not empty", "count": count})
			}
			valid := false
			for _, existing := range rest {
				if existing == migrateTo {
					valid = true
					break
				}
			}
			if !valid {
				return c.Error(400, "migrateTo must be an existing category")
			}
			if err := posts.MigrateCategory(name, migrateTo); err != nil {
				return c.Error(500, "internal error")
			}
		}
		if err := settings.SetCategories(rest); err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true, "migrated": count})
	})
}

// pathName 读取并 URL 解码路径参数（中文分类名经 percent-encoding 传输）。
func pathName(c *hybrid.ApiCtx) string {
	raw := c.Param("name")
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
	return strings.TrimSpace(raw)
}

// queryString 读取并 URL 解码查询参数（中文值经 percent-encoding 传输）。
func queryString(c *hybrid.ApiCtx, key string) string {
	raw := c.Query(key)
	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}
	return strings.TrimSpace(raw)
}
