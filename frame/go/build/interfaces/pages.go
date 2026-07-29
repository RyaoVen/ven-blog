// 页面注册：hybrid PageCtx → 应用服务 → initialState JSON。
package interfaces

import (
	"errors"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/hybrid"
)

// RegisterPages 注册文章相关页面。
// 列表/详情是公开 ISR 静态页（物化落盘直发，DataChange 失效再生）；
// /write 是 author 专属动态页；/login 与 /403 是框架守卫要求的公开空数据页。
func RegisterPages(a *hybrid.App, posts *postapp.Service) error {
	// 首页：最近 5 篇文章
	if err := a.Page("/", nil, func(c *hybrid.PageCtx) error {
		list, err := posts.ListRecent(5)
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"posts": toPostViews(list)})
	}); err != nil {
		return err
	}

	// 文章列表（ISR，全站仅一页）
	if err := a.StaticPage("/posts", 1, true, func(c *hybrid.PageCtx) error {
		list, err := posts.ListRecent(0)
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"posts": toPostViews(list)})
	}); err != nil {
		return err
	}

	// 文章详情（ISR，上限 1000 页，全局更新时按热度预渲染）
	if err := a.StaticPage("/posts/:id", 1000, true, func(c *hybrid.PageCtx) error {
		p, err := posts.Get(mustID(c.Param("id")))
		if errors.Is(err, post.ErrNotFound) {
			return c.NotFound()
		}
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"post": toPostView(p)})
	}); err != nil {
		return err
	}

	// 发文/编辑页：?id= 时回传被编辑文章供表单回填
	if err := a.Page("/write", []string{"author"}, func(c *hybrid.PageCtx) error {
		if raw := c.Query("id"); raw != "" {
			p, err := posts.Get(mustID(raw))
			if errors.Is(err, post.ErrNotFound) {
				return c.NotFound()
			}
			if err != nil {
				return err
			}
			return c.JSON(map[string]any{"post": toPostView(p)})
		}
		return c.JSON(map[string]any{"post": nil})
	}); err != nil {
		return err
	}

	// 登录页与 403 页：空数据渲染
	if err := a.Page("/login", nil, func(c *hybrid.PageCtx) error { return c.Render() }); err != nil {
		return err
	}
	return a.Page("/403", nil, func(c *hybrid.PageCtx) error { return c.Render() })
}

// mustID 解析路径/查询参数中的 ID；非法输入归一为 0（必然 miss，走 NotFound）。
func mustID(raw string) int64 {
	id, err := parseID(raw)
	if err != nil {
		return 0
	}
	return id
}
