// 业务 API 注册：文章 CRUD（框架自动 /api 前缀）+ 失效声明。
package interfaces

import (
	"errors"
	"strconv"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/hybrid"
)

// postInput 是文章创建/更新的请求体。
type postInput struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Content  string   `json:"content"`
	Summary  string   `json:"summary"`
	CoverURL string   `json:"coverUrl"`
	Tags     []string `json:"tags"`
}

// toServiceInput 请求体 → 用例入参。
func (in postInput) toServiceInput() postapp.PostInput {
	return postapp.PostInput{
		Title:    in.Title,
		Category: in.Category,
		Content:  in.Content,
		Summary:  in.Summary,
		CoverURL: in.CoverURL,
		Tags:     in.Tags,
	}
}

// currentUserID 从会话取调用者用户 ID（接口守卫已确保登录与角色，这里只做转换与兜底）。
func currentUserID(c *hybrid.ApiCtx) (int64, error) {
	userID, _, ok := c.User()
	if !ok {
		return 0, errors.New("unauthenticated")
	}
	return strconv.ParseInt(userID, 10, 64)
}

// RegisterAPIs 注册文章 CRUD API。发文/编辑归属经 c.User() 取调用者；
// notifyNewPost 为创建成功后的通知回调（订阅通知器，组装根注入；nil 表示不通知）。
func RegisterAPIs(a *hybrid.App, posts *postapp.Service, notifyNewPost PostNotifier) error {
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
		authorID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		p, err := posts.Create(authorID, in.toServiceInput())
		if err != nil {
			return writePostError(c, err)
		}
		declarePostsChanged(a, p.ID)
		// 仅创建成功触发订阅通知（异步发信，不阻塞响应）；更新不通知
		if notifyNewPost != nil {
			notifyNewPost(p)
		}
		return c.JSON(201, map[string]any{"id": strconv.FormatInt(p.ID, 10)})
	}); err != nil {
		return err
	}

	if err := a.Put("/posts/:id", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in postInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		p, err := posts.Update(mustID(c.Param("id")), in.toServiceInput())
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

// declarePostsChanged 声明文章数据变更：动态页缓存失效（/posts 列表与 / 首页最近文章）
// + 单篇静态详情失效。永远异步立即返回；ISR 再生与 SSE 推送由事件总线在 debounce 合批后完成。
// /posts 是带 ?tag=/?page= 查询串的动态页，不能走 DataChange（静态页通道，会报 static page not declared）。
func declarePostsChanged(a *hybrid.App, id int64) {
	a.InvalidatePage("/posts")
	a.InvalidatePage("/")
	if id > 0 {
		_ = a.DataChange("/posts/:id", strconv.FormatInt(id, 10))
	}
}
