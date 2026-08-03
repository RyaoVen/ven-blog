// 动态接口注册：/moments 公开 ISR 页 + 发布/删除 API（框架自动 /api 前缀）。
package interfaces

import (
	"errors"
	"strconv"
	"time"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/domain/moment"
	"ven_hybird/hybrid"
)

// MomentView 是动态对外的 JSON 视图（ID 以字符串下发，与前端类型契约一致）。
type MomentView struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	AuthorName string    `json:"authorName"`
	Pinned     bool      `json:"pinned"`
	CreatedAt  time.Time `json:"createdAt"`
}

// toMomentView 领域实体 → JSON 视图。
func toMomentView(m *moment.Moment) MomentView {
	return MomentView{
		ID:         strconv.FormatInt(m.ID, 10),
		Content:    m.Content,
		AuthorName: m.AuthorName,
		Pinned:     m.Pinned,
		CreatedAt:  m.CreatedAt,
	}
}

// toMomentViews 批量转换。
func toMomentViews(moments []*moment.Moment) []MomentView {
	views := make([]MomentView, 0, len(moments))
	for _, m := range moments {
		views = append(views, toMomentView(m))
	}
	return views
}

// momentInput 是动态发布请求体。
type momentInput struct {
	Content string `json:"content"`
}

// RegisterMoments 注册 /moments 页面与发布/删除 API。
// 列表是公开 ISR 静态页（物化落盘直发，DataChange 失效再生）；
// 发布/删除仅 author，归属经 c.User() 取调用者；页面数据附带各动态评论数。
// settings 提供评论总开关（comments_enabled）：关闭时评论数一律为 0。
func RegisterMoments(a *hybrid.App, moments *momentapp.Service, comments *commentapp.Service, inter *interactionapp.Service, settings *settingsapp.Service) error {
	// 动态时间线（ISR，全站仅一页）
	if err := a.StaticPage("/moments", 1, true, func(c *hybrid.PageCtx) error {
		list, err := moments.List()
		if err != nil {
			return err
		}
		countViews := make(map[string]int)
		if enabled, err := settings.CommentsEnabled(); err == nil && enabled {
			counts, err := comments.MomentCounts()
			if err != nil {
				return err
			}
			countViews = make(map[string]int, len(counts))
			for id, n := range counts {
				countViews[strconv.FormatInt(id, 10)] = n
			}
		}
		likeCounts, err := inter.MomentLikeCounts()
		if err != nil {
			return err
		}
		likeViews := make(map[string]int, len(likeCounts))
		for id, n := range likeCounts {
			likeViews[strconv.FormatInt(id, 10)] = n
		}
		return c.JSON(map[string]any{"moments": toMomentViews(list), "commentCounts": countViews, "likeCounts": likeViews})
	}); err != nil {
		return err
	}

	// 发布动态：创建后声明 /moments 失效（异步 debounce 再生 + SSE 推送）
	if err := a.Post("/moments", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in momentInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		authorID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		m, err := moments.Create(authorID, in.Content)
		if err != nil {
			return writeMomentError(c, err)
		}
		_ = a.DataChange("/moments")
		return c.JSON(201, map[string]any{"id": strconv.FormatInt(m.ID, 10)})
	}); err != nil {
		return err
	}

	// 删除动态：同样声明 /moments 失效
	return a.Delete("/moments/:id", []string{"author"}, func(c *hybrid.ApiCtx) error {
		if err := moments.Delete(mustID(c.Param("id"))); err != nil {
			return writeMomentError(c, err)
		}
		_ = a.DataChange("/moments")
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// writeMomentError 应用层错误 → HTTP 状态映射。
func writeMomentError(c *hybrid.ApiCtx, err error) error {
	var vErr *momentapp.ValidationError
	switch {
	case errors.As(err, &vErr):
		return c.Error(400, vErr.Message)
	case errors.Is(err, moment.ErrNotFound):
		return c.Error(404, "moment not found")
	default:
		return c.Error(500, "internal error")
	}
}
