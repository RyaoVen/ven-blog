// 用户个人页（/users/:name）与作者主页（/author/:name）注册：均为公开动态页。
package interfaces

import (
	"errors"
	"time"

	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
)

// UserView 是用户公开信息的 JSON 视图（个人页/作者主页共用，时间格式与 PostView 一致）。
type UserView struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Bio       string    `json:"bio"`
	AvatarURL string    `json:"avatarUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

// ProfileStats 是个人页的作品统计。
type ProfileStats struct {
	Posts    int `json:"posts"`
	Comments int `json:"comments"`
}

// toUserView 领域实体 → JSON 视图（不含任何凭据字段）。
func toUserView(u *user.User) UserView {
	return UserView{
		Username:  u.Username,
		Role:      u.Role.String(),
		Bio:       u.Bio,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
	}
}

// RegisterProfiles 注册用户个人页与作者主页。
// 个人页对任意注册用户开放；作者主页仅当目标用户是 author 时存在，其余一律 404（不暴露角色信息）。
func RegisterProfiles(a *hybrid.App, users *userapp.Service, posts *postapp.Service, gb *guestbookapp.Service, settings *settingsapp.Service) error {
	// 用户个人页：公开信息 + 文章/评论统计；isAuthor 供前端展示作者主页入口；
	// viewer 即本人时附带收藏列表（仅本人可见，他人不可见）
	if err := a.Page("/users/:name", nil, func(c *hybrid.PageCtx) error {
		profile, err := users.GetProfile(c.Param("name"))
		if errors.Is(err, user.ErrNotFound) {
			return c.NotFound()
		}
		if err != nil {
			return err
		}
		payload := map[string]any{
			"user":     toUserView(profile.User),
			"stats":    ProfileStats{Posts: profile.Posts, Comments: profile.Comments},
			"isAuthor": profile.User.Role == user.RoleAuthor,
		}
		if viewerID, _, ok := c.User(); ok {
			if vid, parseErr := parseID(viewerID); parseErr == nil && vid == profile.User.ID {
				payload["email"] = profile.User.Email
				favs, favErr := posts.ListFavorites(vid)
				if favErr != nil {
					return favErr
				}
				payload["favorites"] = toPostViews(favs)
			}
		}
		return c.JSON(payload)
	}); err != nil {
		return err
	}

	// 作者主页：四模块（介绍/展示柜/友链/留言板）；非 author 视同不存在
	return a.Page("/author/:name", nil, func(c *hybrid.PageCtx) error {
		profile, err := users.GetProfile(c.Param("name"))
		if errors.Is(err, user.ErrNotFound) {
			return c.NotFound()
		}
		if err != nil {
			return err
		}
		if profile.User.Role != user.RoleAuthor {
			return c.NotFound()
		}
		// 展示柜：2 最新文章 + 静态项目卡
		latest, err := posts.ListRecent(2)
		if err != nil {
			return err
		}
		// 留言板
		entries, err := gb.List(50)
		if err != nil {
			return err
		}
		content, err := settings.Content()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{
			"author":      toUserView(profile.User),
			"intro":       map[string]any{"paragraphs": content.Paragraphs, "skills": content.Skills},
			"showcase":    map[string]any{"projects": content.Projects, "articles": toPostViews(latest)},
			"friendLinks": content.Friends,
			"guestbook":   toGuestbookViews(entries),
		})
	})
}
