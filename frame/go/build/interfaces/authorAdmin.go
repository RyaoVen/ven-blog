// 个人主页后台编辑接口：介绍段落/技能/展示柜/友链 的读取与保存（全部 author 守卫）。
package interfaces

import (
	"strconv"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/hybrid"
)

// authorContentInput 个人主页内容保存请求体。
type authorContentInput struct {
	Paragraphs    []string                 `json:"paragraphs"`
	Skills        []settingsapp.Skill      `json:"skills"`
	Friends       []settingsapp.FriendLink `json:"friends"`
	Projects      []settingsapp.Project    `json:"projects"`
	ShowcasePosts []int64                  `json:"showcasePosts"`
}

// RegisterAuthorAdmin 注册个人主页后台编辑页与保存接口。
func RegisterAuthorAdmin(a *hybrid.App, settings *settingsapp.Service, posts *postapp.Service, authorName string) error {
	admin := []string{"author"}

	// 编辑页（读取当前内容 + 全部文章供展示柜选取）
	if err := a.Page("/admin/author", admin, func(c *hybrid.PageCtx) error {
		content, err := settings.Content()
		if err != nil {
			return err
		}
		showcaseIDs, err := settings.ShowcasePosts()
		if err != nil {
			return err
		}
		all, err := posts.ListRecent(0)
		if err != nil {
			return err
		}
		type postOption struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		options := make([]postOption, 0, len(all))
		for _, p := range all {
			options = append(options, postOption{ID: strconv.FormatInt(p.ID, 10), Title: p.Title})
		}
		return c.JSON(map[string]any{
			"paragraphs":    content.Paragraphs,
			"skills":        content.Skills,
			"friends":       content.Friends,
			"projects":      content.Projects,
			"showcasePosts": showcaseIDs,
			"allPosts":      options,
		})
	}); err != nil {
		return err
	}

	// 保存内容（分项写入 + 失效作者主页/首页——项目卡同时展示在首页仪表盘）
	if err := a.Put("/admin/author/content", admin, func(c *hybrid.ApiCtx) error {
		var in authorContentInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		if err := settings.SetParagraphs(in.Paragraphs); err != nil {
			return c.Error(500, "internal error")
		}
		if err := settings.SetSkills(in.Skills); err != nil {
			return c.Error(500, "internal error")
		}
		if err := settings.SetFriends(in.Friends); err != nil {
			return c.Error(500, "internal error")
		}
		if err := settings.SetProjects(in.Projects); err != nil {
			return c.Error(500, "internal error")
		}
		if err := settings.SetShowcasePosts(in.ShowcasePosts); err != nil {
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/author/" + authorName)
		a.InvalidatePage("/")
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	return nil
}
