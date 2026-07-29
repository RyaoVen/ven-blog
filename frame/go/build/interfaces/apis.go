// 业务 API 注册：文章 CRUD（框架自动 /api 前缀）+ 失效声明。
package interfaces

import (
	"errors"
	"strconv"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
)

// postInput 是文章创建/更新的请求体。
type postInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// RegisterAPIs 注册文章 CRUD API。
// author 是当前唯一可发文账号（框架会话只携带角色、尚无用户身份；
// 待框架支持 CurrentUser 后改为取调用者）。仅 author 角色能调写接口，归属它是正确的。
func RegisterAPIs(a *hybrid.App, posts *postapp.Service, author *user.User) error {
	if err := a.Get("/posts", nil, func(c *hybrid.ApiCtx) error {
		list, err := posts.ListRecent(0)
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"posts": toPostViews(list)})
	}); err != nil {
		return err
	}

	if err := a.Post("/posts", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in postInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		p, err := posts.Create(author.ID, in.Title, in.Content)
		if err != nil {
			return writePostError(c, err)
		}
		declarePostsChanged(a, p.ID)
		return c.JSON(201, map[string]any{"id": strconv.FormatInt(p.ID, 10)})
	}); err != nil {
		return err
	}

	if err := a.Put("/posts/:id", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in postInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		p, err := posts.Update(mustID(c.Param("id")), in.Title, in.Content)
		if err != nil {
			return writePostError(c, err)
		}
		declarePostsChanged(a, p.ID)
		return c.JSON(200, map[string]any{"post": toPostView(p)})
	}); err != nil {
		return err
	}

	return a.Delete("/posts/:id", []string{"author"}, func(c *hybrid.ApiCtx) error {
		id := mustID(c.Param("id"))
		if err := posts.Delete(id); err != nil {
			return writePostError(c, err)
		}
		declarePostsChanged(a, id)
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// writePostError 应用层错误 → HTTP 状态映射。
func writePostError(c *hybrid.ApiCtx, err error) error {
	var vErr *postapp.ValidationError
	switch {
	case errors.As(err, &vErr):
		return c.Error(400, vErr.Message)
	case errors.Is(err, post.ErrNotFound):
		return c.Error(404, "post not found")
	default:
		return c.Error(500, "internal error")
	}
}

// declarePostsChanged 声明文章数据变更：列表全局失效 + 单篇详情失效。
// 永远异步立即返回；ISR 再生与 SSE 推送由事件总线在 debounce 合批后完成。
func declarePostsChanged(a *hybrid.App, id int64) {
	_ = a.DataChange("/posts")
	if id > 0 {
		_ = a.DataChange("/posts/:id", strconv.FormatInt(id, 10))
	}
}
