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
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// toGuestbookView 领域实体 → JSON 视图。
func toGuestbookView(e *guestbook.Entry) GuestbookView {
	return GuestbookView{
		ID:        strconv.FormatInt(e.ID, 10),
		UserID:    strconv.FormatInt(e.UserID, 10),
		Username:  e.Username,
		Content:   e.Content,
		Status:    e.Status,
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
// authorNameFn 现取当前作者用户名，用于发表/删除后失效作者主页（改名后旧路径不再有效）。
func RegisterGuestbookAPI(a *hybrid.App, gb *guestbookapp.Service, authorNameFn func() string) error {
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
		a.InvalidatePage("/author/" + authorNameFn())
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
		a.InvalidatePage("/author/" + authorNameFn())
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// adminGuestbookView 后台留言管理视图（含审核状态与驳回原因）。
type adminGuestbookView struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	Username       string    `json:"username"`
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	RejectedReason string    `json:"rejectedReason"`
	CreatedAt      time.Time `json:"createdAt"`
}

// toAdminGuestbookView 领域实体 → 后台管理视图。
func toAdminGuestbookView(e *guestbook.Entry) adminGuestbookView {
	return adminGuestbookView{
		ID:             strconv.FormatInt(e.ID, 10),
		UserID:         strconv.FormatInt(e.UserID, 10),
		Username:       e.Username,
		Content:        e.Content,
		Status:         e.Status,
		RejectedReason: e.RejectedReason,
		CreatedAt:      e.CreatedAt,
	}
}

// toAdminGuestbookViews 批量转换。
func toAdminGuestbookViews(entries []*guestbook.Entry) []adminGuestbookView {
	views := make([]adminGuestbookView, 0, len(entries))
	for _, e := range entries {
		views = append(views, toAdminGuestbookView(e))
	}
	return views
}

// RegisterGuestbookAdmin 注册留言板管理页与审核 API（全部 author 守卫）。
// 删除复用 RegisterGuestbookAPI 的 DELETE /guestbook/:id（本人或 author 可删）。
// 失效规则：approve/recover 使留言从不可见变为可见，失效作者主页；reject 无读者可见性变化，不失效。
func RegisterGuestbookAdmin(a *hybrid.App, gb *guestbookapp.Service, authorNameFn func() string) error {
	admin := []string{"author"}

	// 留言板管理页：全量 + 待审核 + 被驳回三区
	if err := a.Page("/admin/guestbook", admin, func(c *hybrid.PageCtx) error {
		all, err := gb.ListAll(200)
		if err != nil {
			return err
		}
		pending, err := gb.ListPending()
		if err != nil {
			return err
		}
		rejected, err := gb.ListRejected()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{
			"entries":  toAdminGuestbookViews(all),
			"pending":  toAdminGuestbookViews(pending),
			"rejected": toAdminGuestbookViews(rejected),
		})
	}); err != nil {
		return err
	}

	// 审核通过留言
	if err := a.Post("/guestbook/:id/approve", admin, func(c *hybrid.ApiCtx) error {
		if err := gb.Approve(mustID(c.Param("id"))); err != nil {
			switch {
			case errors.Is(err, guestbook.ErrNotFound):
				return c.Error(404, "entry not found")
			default:
				return c.Error(500, "internal error")
			}
		}
		a.InvalidatePage("/author/" + authorNameFn())
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 驳回留言（reason 必填 ≤200）；approved→rejected 是可见性变化，作者主页失效
	if err := a.Post("/guestbook/:id/reject", admin, func(c *hybrid.ApiCtx) error {
		var in rejectInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		err := gb.Reject(mustID(c.Param("id")), in.Reason)
		var vErr *guestbookapp.ValidationError
		switch {
		case errors.As(err, &vErr):
			return c.Error(400, vErr.Message)
		case errors.Is(err, guestbook.ErrNotFound):
			return c.Error(404, "entry not found")
		case err != nil:
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/author/" + authorNameFn())
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 恢复被驳回留言
	return a.Post("/guestbook/:id/recover", admin, func(c *hybrid.ApiCtx) error {
		err := gb.Recover(mustID(c.Param("id")))
		switch {
		case errors.Is(err, guestbook.ErrNotFound):
			return c.Error(404, "entry not found")
		case errors.Is(err, guestbook.ErrInvalidState):
			return c.Error(400, "entry not in rejected state")
		case err != nil:
			return c.Error(500, "internal error")
		}
		a.InvalidatePage("/author/" + authorNameFn())
		return c.JSON(200, map[string]any{"ok": true})
	})
}
