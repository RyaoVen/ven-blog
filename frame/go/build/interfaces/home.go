// 首页数据装配：hero 作者卡 + 双短列表 + 仪表盘 + 文章时间线 + 订阅区。
package interfaces

import (
	"strconv"
	"time"

	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
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

// RegisterHome 注册首页（"/"）：聚合最近文章/动态、站点统计、内容配置与时间线。
// 首页是动态页（数据杂、更新频率低，1min 页面缓存足够；文章变更时 InvalidatePage("/") 刷新）。
func RegisterHome(a *hybrid.App, posts *postapp.Service, moments *momentapp.Service, authorFn func() (*user.User, error), settings *settingsapp.Service) error {
	return a.Page("/", nil, func(c *hybrid.PageCtx) error {
		author, err := authorFn()
		if err != nil {
			return err
		}
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
		// 运营起点（最早一篇文章）、最新文章与其"几天前"文案
		launchAt := ""
		latestID := ""
		latestAgo := ""
		days := 0
		if len(allPosts) > 0 {
			latest := allPosts[0].CreatedAt
			oldest := allPosts[len(allPosts)-1].CreatedAt
			launchAt = oldest.Format(time.RFC3339)
			latestID = strconv.FormatInt(allPosts[0].ID, 10)
			days = int(time.Since(oldest).Hours()/24) + 1
			ago := int(time.Since(latest).Hours() / 24)
			if ago <= 0 {
				latestAgo = "今天"
			} else {
				latestAgo = strconv.Itoa(ago) + " 天前"
			}
		}
		content, err := settings.Content()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{
			"recentPosts":   toPostViews(recentPosts),
			"recentMoments": recentMoments,
			"stats":         map[string]any{"posts": postCount, "words": totalChars, "days": days, "launchAt": launchAt, "latestID": latestID, "latestAgo": latestAgo},
			"projects":      content.Projects,
			"quotes":        content.Quotes,
			"timeline":      timeline,
			"author": homeAuthorView{
				Username:  author.Username,
				Bio:       author.Bio,
				AvatarURL: author.AvatarURL,
				GitHub:    content.GitHub,
			},
		})
	})
}

// RegisterSiteInfo 注册站点公开信息接口（导航栏作者头像等全局展示用，无需登录）。
func RegisterSiteInfo(a *hybrid.App, authorFn func() (*user.User, error)) error {
	return a.Get("/site", nil, func(c *hybrid.ApiCtx) error {
		author, err := authorFn()
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{
			"name":       "ven-blog",
			"authorName": author.Username,
			"avatarUrl":  author.AvatarURL,
			"github":     authorGitHub,
		})
	})
}
