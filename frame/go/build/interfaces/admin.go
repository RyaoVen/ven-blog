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
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/subscribeapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/application/visitapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/build/domain/post"
	"ven_hybird/hybrid"
)

// adminCommentView 后台评论管理视图（含所属文章标题与审核状态）。
type adminCommentView struct {
	ID             string    `json:"id"`
	PostID         string    `json:"postId"`
	PostTitle      string    `json:"postTitle"`
	Username       string    `json:"username"`
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	RejectedReason string    `json:"rejectedReason"`
	CreatedAt      time.Time `json:"createdAt"`
}

// toAdminCommentView 领域实体 → 后台评论管理视图。
func toAdminCommentView(c *comment.Comment) adminCommentView {
	return adminCommentView{
		ID:             strconv.FormatInt(c.ID, 10),
		PostID:         strconv.FormatInt(c.PostID, 10),
		PostTitle:      c.PostTitle,
		Username:       c.Username,
		Content:        c.Content,
		Status:         c.Status,
		RejectedReason: c.RejectedReason,
		CreatedAt:      c.CreatedAt,
	}
}

// toAdminCommentViews 批量转换。
func toAdminCommentViews(list []*comment.Comment) []adminCommentView {
	views := make([]adminCommentView, 0, len(list))
	for _, cm := range list {
		views = append(views, toAdminCommentView(cm))
	}
	return views
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
	settings *settingsapp.Service,
	visits *visitapp.Service,
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
		// 访问统计：全站访问量 / 文章点击总量 / 近 30 天 PV
		visitTotal, postHitsTotal, err := visits.Totals()
		if err != nil {
			return err
		}
		pv30, err := visits.Daily(30)
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
		// 用户增长折线（7/30/365）与增量对比
		growth7, err := users.DailyRegistrations(7)
		if err != nil {
			return err
		}
		growth30, err := users.DailyRegistrations(30)
		if err != nil {
			return err
		}
		growth365, err := users.DailyRegistrations(365)
		if err != nil {
			return err
		}
		now := time.Now()
		dayAgo, err2 := users.CountSince(now.AddDate(0, 0, -1))
		if err2 != nil {
			return err2
		}
		dayAgo2, err3 := users.CountSince(now.AddDate(0, 0, -2))
		if err3 != nil {
			return err3
		}
		weekAgo, err4 := users.CountSince(now.AddDate(0, 0, -7))
		if err4 != nil {
			return err4
		}
		weekAgo2, err5 := users.CountSince(now.AddDate(0, 0, -14))
		if err5 != nil {
			return err5
		}
		monthAgo, err6 := users.CountSince(now.AddDate(0, 0, -30))
		if err6 != nil {
			return err6
		}
		monthAgo2, err7 := users.CountSince(now.AddDate(0, 0, -60))
		if err7 != nil {
			return err7
		}
		// 发布日历热力图（文章+动态合并，近一年）
		postDaily, err := posts.DailyPublication(365)
		if err != nil {
			return err
		}
		momentDaily, err := moments.DailyCounts(365)
		if err != nil {
			return err
		}
		type heatDay struct {
			Date  string `json:"date"`
			Posts int    `json:"posts"`
			Chars int    `json:"chars"`
		}
		heatmap := make([]heatDay, 0, len(postDaily))
		for _, d := range postDaily {
			heatmap = append(heatmap, heatDay{Date: d.Date, Posts: d.Count + momentDaily[d.Date], Chars: d.Chars})
		}
		// 分类计数（雷达图）
		categoryCounts, err := posts.CategoryCounts()
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
				"visits": visitTotal, "postHits": postHitsTotal,
			},
			"pv30":            pv30,
			"recentComments": recentViews,
			"userGrowth": map[string]any{
				"d7": growth7, "d30": growth30, "d365": growth365,
				"deltas": map[string]any{
					"yesterday": dayAgo - (dayAgo2 - dayAgo),
					"week":      weekAgo - (weekAgo2 - weekAgo),
					"month":     monthAgo - (monthAgo2 - monthAgo),
				},
			},
			"heatmap":        heatmap,
			"categoryCounts": categoryCounts,
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
		views := toPostViews(list)
		// 批量统计（各一次查询出 map，避免 N+1）：点击来自 visits 聚合，点赞/收藏来自互动表
		hits, err := visits.PostHits()
		if err != nil {
			return err
		}
		likeCounts, err := inter.PostLikeCounts()
		if err != nil {
			return err
		}
		favoriteCounts, err := inter.PostFavoriteCounts()
		if err != nil {
			return err
		}
		for i := range views {
			id := mustID(views[i].ID)
			views[i].Hits = hits[id]
			views[i].Likes = likeCounts[id]
			views[i].Favorites = favoriteCounts[id]
		}
		return c.JSON(map[string]any{"posts": views})
	}); err != nil {
		return err
	}

	// 新建文章（编辑器空态 + 分类列表）
	if err := a.Page("/admin/posts/new", admin, func(c *hybrid.PageCtx) error {
		categories, err := settings.Categories()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"post": nil, "categories": categories})
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
		categories, err := settings.Categories()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"post": toPostView(p), "categories": categories})
	}); err != nil {
		return err
	}

	// 评论管理（全量 + 待审核 + 被驳回）
	if err := a.Page("/admin/comments", admin, func(c *hybrid.PageCtx) error {
		list, err := comments.ListAll(100)
		if err != nil {
			return err
		}
		views := toAdminCommentViews(list)
		pending, err := comments.ListPending()
		if err != nil {
			return err
		}
		pendingViews := toAdminCommentViews(pending)
		rejected, err := comments.ListRejected()
		if err != nil {
			return err
		}
		rejectedViews := toAdminCommentViews(rejected)
		return c.JSON(map[string]any{"comments": views, "pending": pendingViews, "rejected": rejectedViews})
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
