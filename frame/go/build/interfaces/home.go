// 首页数据装配：hero 作者卡 + 双短列表 + 仪表盘 + 文章时间线 + 订阅区。
package interfaces

import (
	"strconv"
	"time"

	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
)

// homeMomentView 动态短列表视图（首页用，精简字段）。
type homeMomentView struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// homeTimelineItem 时间线条目（精简文章字段）。
type homeTimelineItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

// homeAuthorView hero 作者卡视图。
type homeAuthorView struct {
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatarUrl"`
	GitHub    string `json:"github"`
}

// RegisterHome 注册首页（"/"）：聚合最近文章/动态、站点统计、静态策展内容与时间线。
// 首页是动态页（数据杂、更新频率低，1min 页面缓存足够；文章变更时 InvalidatePage("/") 刷新）。
func RegisterHome(a *hybrid.App, posts *postapp.Service, moments *momentapp.Service, author *user.User) error {
	return a.Page("/", nil, func(c *hybrid.PageCtx) error {
		recentPosts, err := posts.ListRecent(5)
		if err != nil {
			return err
		}
		allMoments, err := moments.List()
		if err != nil {
			return err
		}
		recentMoments := make([]homeMomentView, 0, 5)
		for i, m := range allMoments {
			if i >= 5 {
				break
			}
			recentMoments = append(recentMoments, homeMomentView{
				ID:        strconv.FormatInt(m.ID, 10),
				Content:   m.Content,
				CreatedAt: m.CreatedAt,
			})
		}
		postCount, totalChars, err := posts.SiteStats()
		if err != nil {
			return err
		}
		allPosts, err := posts.ListRecent(0)
		if err != nil {
			return err
		}
		timeline := make([]homeTimelineItem, 0, len(allPosts))
		for _, p := range allPosts {
			timeline = append(timeline, homeTimelineItem{
				ID:        strconv.FormatInt(p.ID, 10),
				Title:     p.Title,
				CreatedAt: p.CreatedAt,
			})
		}
		return c.JSON(map[string]any{
			"recentPosts":   toPostViews(recentPosts),
			"recentMoments": recentMoments,
			"stats":         map[string]any{"posts": postCount, "words": totalChars},
			"projects":      homeProjects,
			"quotes":        homeQuotes,
			"timeline":      timeline,
			"author": homeAuthorView{
				Username:  author.Username,
				Bio:       author.Bio,
				AvatarURL: author.AvatarURL,
				GitHub:    authorGitHub,
			},
		})
	})
}

// RegisterSiteInfo 注册站点公开信息接口（导航栏作者头像等全局展示用，无需登录）。
func RegisterSiteInfo(a *hybrid.App, author *user.User) error {
	return a.Get("/site", nil, func(c *hybrid.ApiCtx) error {
		return c.JSON(200, map[string]any{
			"name":       "ven-blog",
			"authorName": author.Username,
			"github":     authorGitHub,
		})
	})
}
