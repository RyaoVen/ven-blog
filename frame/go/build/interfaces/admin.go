// 后台管理面板：数据面板与文章/评论/动态管理页（全部 author 守卫）+ /write 兼容重定向。
package interfaces

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/subscribeapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/hybrid"
)

// adminCommentView 后台评论管理视图（含所属文章标题）。
type adminCommentView struct {
	ID        string    `json:"id"`
	PostID    string    `json:"postId"`
	PostTitle string    `json:"postTitle"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// RegisterAdmin 注册后台页面（动态页，仅 author）。
func RegisterAdmin(
	a *hybrid.App,
	posts *postapp.Service,
	comments *commentapp.Service,
	inter *interactionapp.Service,
	moments *momentapp.Service,
	subscribe *subscribeapp.Service,
	users *userapp.Service,
) error {
	admin := []string{"author"}

	// 后台独立登入页（公开；登录成功跳 next 或 /admin）
	if err := a.Page("/admin/login", nil, func(c *hybrid.PageCtx) error {
		return c.JSON(nil)
	}); err != nil {
		return err
	}

	// /write 兼容重定向：?id= 保留到编辑页
	a.Server().App().Get("/write", func(ctx *fiber.Ctx) error {
		if id := ctx.Query("id"); id != "" {
			return ctx.Redirect("/admin/posts/"+id+"/edit", fiber.StatusMovedPermanently)
		}
		return ctx.Redirect("/admin/posts/new", fiber.StatusMovedPermanently)
	})

	// 数据面板
	if err := a.Page("/admin", admin, func(c *hybrid.PageCtx) error {
		postCount, totalChars, err := posts.SiteStats()
		if err != nil {
			return err
		}
		commentCount, err := comments.Count()
		if err != nil {
			return err
		}
		likes, favorites, err := inter.Totals()
		if err != nil {
			return err
		}
		userCount, err := users.Count()
		if err != nil {
			return err
		}
		momentCount, err := moments.Count()
		if err != nil {
			return err
		}
		subscriberCount, err := subscribe.Count()
		if err != nil {
			return err
		}
		recent, err := comments.ListAll(5)
		if err != nil {
			return err
		}
		recentViews := make([]adminCommentView, 0, len(recent))
		for _, cm := range recent {
			recentViews = append(recentViews, adminCommentView{
				ID:        strconv.FormatInt(cm.ID, 10),
				PostID:    strconv.FormatInt(cm.PostID, 10),
				PostTitle: cm.PostTitle,
				Username:  cm.Username,
				Content:   cm.Content,
				CreatedAt: cm.CreatedAt,
			})
		}
		return c.JSON(map[string]any{
			"stats": map[string]any{
				"posts": postCount, "words": totalChars, "comments": commentCount,
				"likes": likes, "favorites": favorites, "users": userCount,
				"moments": momentCount, "subscribers": subscriberCount,
			},
			"recentComments": recentViews,
		})
	}); err != nil {
		return err
	}

	// 文章管理
	if err := a.Page("/admin/posts", admin, func(c *hybrid.PageCtx) error {
		list, err := posts.ListRecent(0)
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"posts": toPostViews(list)})
	}); err != nil {
		return err
	}

	// 新建文章（编辑器空态）
	if err := a.Page("/admin/posts/new", admin, func(c *hybrid.PageCtx) error {
		return c.JSON(map[string]any{"post": nil})
	}); err != nil {
		return err
	}

	// 编辑文章（回填）
	if err := a.Page("/admin/posts/:id/edit", admin, func(c *hybrid.PageCtx) error {
		p, err := posts.Get(mustID(c.Param("id")))
		if err == post.ErrNotFound {
			return c.NotFound()
		}
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"post": toPostView(p)})
	}); err != nil {
		return err
	}

	// 评论管理
	if err := a.Page("/admin/comments", admin, func(c *hybrid.PageCtx) error {
		list, err := comments.ListAll(100)
		if err != nil {
			return err
		}
		views := make([]adminCommentView, 0, len(list))
		for _, cm := range list {
			views = append(views, adminCommentView{
				ID:        strconv.FormatInt(cm.ID, 10),
				PostID:    strconv.FormatInt(cm.PostID, 10),
				PostTitle: cm.PostTitle,
				Username:  cm.Username,
				Content:   cm.Content,
				CreatedAt: cm.CreatedAt,
			})
		}
		return c.JSON(map[string]any{"comments": views})
	}); err != nil {
		return err
	}

	// 动态管理
	return a.Page("/admin/moments", admin, func(c *hybrid.PageCtx) error {
		list, err := moments.List()
		if err != nil {
			return err
		}
		views := make([]MomentView, 0, len(list))
		for _, m := range list {
			views = append(views, toMomentView(m))
		}
		return c.JSON(map[string]any{"moments": views})
	})
}
