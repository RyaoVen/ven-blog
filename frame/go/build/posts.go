// 博客文章业务：页面（ISR 列表/详情 + 动态页）与 API（CRUD）注册。
package build

import (
	"strings"

	"ven_hybird/hybrid"
)

// registerPostPages 注册文章相关页面。
// 列表/详情是公开 ISR 静态页（物化落盘直发，DataChange 失效再生）；
// /write 是 author 专属动态页；/login 与 /403 是框架守卫要求的公开空数据页。
func registerPostPages(a *hybrid.App, s *blogStore) error {
	// 首页：最近 5 篇文章
	if err := a.Page("/", nil, func(c *hybrid.PageCtx) error {
		posts := s.listPosts()
		if len(posts) > 5 {
			posts = posts[:5]
		}
		return c.JSON(map[string]any{"posts": posts})
	}); err != nil {
		return err
	}

	// 文章列表（ISR，全站仅一页）
	if err := a.StaticPage("/posts", 1, true, func(c *hybrid.PageCtx) error {
		return c.JSON(map[string]any{"posts": s.listPosts()})
	}); err != nil {
		return err
	}

	// 文章详情（ISR，上限 1000 页，全局更新时按热度预渲染）
	if err := a.StaticPage("/posts/:id", 1000, true, func(c *hybrid.PageCtx) error {
		p, ok := s.getPost(c.Param("id"))
		if !ok {
			return c.NotFound()
		}
		return c.JSON(map[string]any{"post": p})
	}); err != nil {
		return err
	}

	// 发文/编辑页：?id= 时回传被编辑文章供表单回填
	if err := a.Page("/write", []string{"author"}, func(c *hybrid.PageCtx) error {
		if id := c.Query("id"); id != "" {
			p, ok := s.getPost(id)
			if !ok {
				return c.NotFound()
			}
			return c.JSON(map[string]any{"post": p})
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

// postInput 是文章创建/更新的请求体。
type postInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// validate 校验请求体，返回错误消息（空串表示通过）。
func (in *postInput) validate() string {
	if strings.TrimSpace(in.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(in.Content) == "" {
		return "content is required"
	}
	return ""
}

// registerPostAPIs 注册文章 CRUD API（框架自动加 /api 前缀）。
func registerPostAPIs(a *hybrid.App, s *blogStore) error {
	if err := a.Get("/posts", nil, func(c *hybrid.ApiCtx) error {
		return c.JSON(200, map[string]any{"posts": s.listPosts()})
	}); err != nil {
		return err
	}

	if err := a.Post("/posts", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in postInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if msg := in.validate(); msg != "" {
			return c.Error(400, msg)
		}
		p := s.createPost(strings.TrimSpace(in.Title), in.Content)
		declarePostsChanged(a, p.ID)
		return c.JSON(201, map[string]any{"id": p.ID})
	}); err != nil {
		return err
	}

	if err := a.Put("/posts/:id", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in postInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if msg := in.validate(); msg != "" {
			return c.Error(400, msg)
		}
		p, ok := s.updatePost(c.Param("id"), strings.TrimSpace(in.Title), in.Content)
		if !ok {
			return c.Error(404, "post not found")
		}
		declarePostsChanged(a, p.ID)
		return c.JSON(200, map[string]any{"post": p})
	}); err != nil {
		return err
	}

	return a.Delete("/posts/:id", []string{"author"}, func(c *hybrid.ApiCtx) error {
		id := c.Param("id")
		if !s.deletePost(id) {
			return c.Error(404, "post not found")
		}
		declarePostsChanged(a, id)
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// declarePostsChanged 声明文章数据变更：列表全局失效 + 单篇详情失效。
// 永远异步立即返回；ISR 再生与 SSE 推送由事件总线在 debounce 合批后完成。
func declarePostsChanged(a *hybrid.App, id string) {
	_ = a.DataChange("/posts")
	if id != "" {
		_ = a.DataChange("/posts/:id", id)
	}
}
