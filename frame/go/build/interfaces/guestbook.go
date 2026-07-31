// 留言板接口：列表公开，发表/删除需登录（本人或 author 可删）。
package interfaces

import (
	"errors"
	"strconv"
	"time"

	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/domain/guestbook"
	"ven_hybird/hybrid"
)

// GuestbookView 留言 JSON 视图（ID 字符串下发）。
type GuestbookView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// toGuestbookView 领域实体 → JSON 视图。
func toGuestbookView(e *guestbook.Entry) GuestbookView {
	return GuestbookView{
		ID:        strconv.FormatInt(e.ID, 10),
		UserID:    strconv.FormatInt(e.UserID, 10),
		Username:  e.Username,
		Content:   e.Content,
		CreatedAt: e.CreatedAt,
	}
}

// toGuestbookViews 批量转换。
func toGuestbookViews(entries []*guestbook.Entry) []GuestbookView {
	views := make([]GuestbookView, 0, len(entries))
	for _, e := range entries {
		views = append(views, toGuestbookView(e))
	}
	return views
}

// guestbookInput 留言请求体。
type guestbookInput struct {
	Content string `json:"content"`
}

// RegisterGuestbookAPI 注册留言板 API。
// authorName 用于发表/删除后失效作者主页（动态页缓存 + SSE 推送）。
func RegisterGuestbookAPI(a *hybrid.App, gb *guestbookapp.Service, authorName string) error {
	// 留言列表（公开）
	if err := a.Get("/guestbook", nil, func(c *hybrid.ApiCtx) error {
		list, err := gb.List(50)
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"entries": toGuestbookViews(list)})
	}); err != nil {
		return err
	}

	// 发表留言
	if err := a.Post("/guestbook", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		var in guestbookInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		e, err := gb.Create(userID, in.Content)
		var vErr *guestbookapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/author/" + authorName)
		return c.JSON(201, toGuestbookView(e))
	}); err != nil {
		return err
	}

	// 删除留言（本人或 author）
	return a.Delete("/guestbook/:id", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, role, ok := c.User()
		if !ok {
			return c.Error(401, "unauthenticated")
		}
		uid, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		err = gb.Delete(uid, role, mustID(c.Param("id")))
		switch {
		case errors.Is(err, guestbook.ErrNotFound):
			return c.Error(404, "entry not found")
		case errors.Is(err, guestbook.ErrForbidden):
			return c.Error(403, "forbidden")
		case err != nil:
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/author/" + authorName)
		return c.JSON(200, map[string]any{"ok": true})
	})
}
