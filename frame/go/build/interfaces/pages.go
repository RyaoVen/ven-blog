// 页面注册：hybrid PageCtx → 应用服务 → initialState JSON。
package interfaces

import (
	"errors"
	"strconv"
	"strings"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/hybrid"
)

// RegisterPages 注册文章相关页面。
// /posts 是带标签筛选与分页的公开动态页（?tag=/?page= 查询串场景不能走 ISR——
// 物化文件按 path 直发，不同 query 会被同一份基础文件截胡）；
// 详情是公开 ISR 静态页（物化落盘直发，DataChange 失效再生）；
// /write 是 author 专属动态页；/login 与 /403 是框架守卫要求的公开空数据页。
// 详情 initialState 只放公开数据（文章/计数/评论列表）；viewer 个性化状态由 /api 互动接口下发。
func RegisterPages(a *hybrid.App, posts *postapp.Service, comments *commentapp.Service, inter *interactionapp.Service, settings *settingsapp.Service) error {
	// 文章列表：?category= 分类筛选 + ?page= 分页，categories 为设置分类列表（筛选框用）
	if err := a.Page("/posts", nil, func(c *hybrid.PageCtx) error {
		page, _ := strconv.Atoi(c.Query("page"))
		category := strings.TrimSpace(c.Query("category"))
		paged, err := posts.List(postapp.ListFilter{Category: category, Page: page})
		if err != nil {
			return err
		}
		categories, err := settings.Categories()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{
			"posts":      toPostViews(paged.Posts),
			"total":      paged.Total,
			"page":       paged.Page,
			"pageSize":   paged.PageSize,
			"category":   category,
			"categories": categories,
		})
	}); err != nil {
		return err
	}

	// 文章详情（ISR，上限 1000 页，全局更新时按热度预渲染）
	// initialState：post + 公开互动计数 + 评论列表（viewer 状态不在此下发，ISR 共享物化不烘个人数据）
	// 评论总开关（comments_enabled）关闭时评论列表下发空数组——前端据此不渲染评论区。
	if err := a.StaticPage("/posts/:id", 1000, true, func(c *hybrid.PageCtx) error {
		p, err := posts.Get(mustID(c.Param("id")))
		if errors.Is(err, post.ErrNotFound) {
			return c.NotFound()
		}
		if err != nil {
			return err
		}
		likeCount, favoriteCount, err := inter.Counts(p.ID)
		if err != nil {
			return err
		}
		commentsEnabled, err := settings.CommentsEnabled()
		if err != nil {
			return err
		}
		cmtViews := []CommentView{}
		if commentsEnabled {
			cmts, err := comments.ListForPost(p.ID)
			if err != nil {
				return err
			}
			cmtViews = toCommentViews(cmts)
		}
		return c.JSON(map[string]any{
			"post":          toPostView(p),
			"likeCount":     likeCount,
			"favoriteCount": favoriteCount,
			"comments":      cmtViews,
		})
	}); err != nil {
		return err
	}

	// 发文/编辑页已迁入 /admin（src/admin/posts/new 与 [id]/edit）；
	// 旧 /write 地址由 interfaces/admin.go 的 raw fiber 路由 301 重定向，此处不再注册。

	// 登录页、注册页与 403 页：空数据页。
	// 必须 c.JSON(nil) 而非 c.Render()——Render() 会强制 SSR 连 data-only 请求也回 HTML，
	// SPA router 取数拿到 HTML 解析失败卡死（直接访问与 SPA 跳转都须正常）。
	if err := a.Page("/login", nil, func(c *hybrid.PageCtx) error { return c.JSON(nil) }); err != nil {
		return err
	}
	if err := a.Page("/register", nil, func(c *hybrid.PageCtx) error { return c.JSON(nil) }); err != nil {
		return err
	}
	return a.Page("/403", nil, func(c *hybrid.PageCtx) error { return c.JSON(nil) })
}

// mustID 解析路径/查询参数中的 ID；非法输入归一为 0（必然 miss，走 NotFound）。
func mustID(raw string) int64 {
	id, err := parseID(raw)
	if err != nil {
		return 0
	}
	return id
}
