// 互动接口：评论/点赞/收藏（JSON，框架自动 /api 前缀）。
package interfaces

import (
	"errors"
	"strconv"
	"time"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/emailauth"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/hybrid"
)

// CommentView 评论 JSON 视图（ID 字符串下发）。
type CommentView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	ReplyTo   string    `json:"replyTo"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// toCommentView 领域实体 → JSON 视图。
func toCommentView(c *comment.Comment) CommentView {
	return CommentView{
		ID:        strconv.FormatInt(c.ID, 10),
		UserID:    strconv.FormatInt(c.UserID, 10),
		Username:  c.Username,
		Content:   c.Content,
		ReplyTo:   c.ReplyTo,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
	}
}

// toCommentViews 批量转换。
func toCommentViews(comments []*comment.Comment) []CommentView {
	views := make([]CommentView, 0, len(comments))
	for _, c := range comments {
		views = append(views, toCommentView(c))
	}
	return views
}

// commentInput 评论请求体（replyTo 为回复目标用户名，可空）。
type commentInput struct {
	Content string `json:"content"`
	ReplyTo string `json:"replyTo"`
}

// rejectInput 驳回请求体（reason 必填 ≤200）。
type rejectInput struct {
	Reason string `json:"reason"`
}

// loginRoles 评论/点赞/收藏对全部登录用户开放（扁平角色下两个都列）。
var loginRoles = []string{"reader", "author"}

// RegisterInteractions 注册互动 API。
// 详情页是 ISR 共享物化——viewer 个性化状态一律走本文件的 JSON 接口，不进页面 initialState。
// emailAuthSvc 用于评论 @ 时的邮件通知（异步不阻塞）；siteURL 拼接原文链接。
// settings 提供评论总开关（comments_enabled）：关闭时发表评论一律 403，后台管理不受影响。
func RegisterInteractions(a *hybrid.App, comments *commentapp.Service, inter *interactionapp.Service, emailAuthSvc *emailauth.Service, users *userapp.Service, settings *settingsapp.Service, siteURL string) error {
	// viewer 互动状态（登录用户挂载后查询）
	if err := a.Get("/posts/:id/interactions", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		status, err := inter.ViewerStatus(userID, mustID(c.Param("id")))
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{
			"userId":        strconv.FormatInt(userID, 10),
			"liked":         status.Liked,
			"favorited":     status.Favorited,
			"likeCount":     status.LikeCount,
			"favoriteCount": status.FavoriteCount,
		})
	}); err != nil {
		return err
	}

	// 切换点赞
	if err := a.Post("/posts/:id/like", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		postID := mustID(c.Param("id"))
		liked, count, err := inter.ToggleLike(userID, postID)
		if err != nil {
			return c.Error(500, "internal error")
		}
		declarePostsChanged(a, postID)
		return c.JSON(200, map[string]any{"liked": liked, "likeCount": count})
	}); err != nil {
		return err
	}

	// 切换收藏
	if err := a.Post("/posts/:id/favorite", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		postID := mustID(c.Param("id"))
		favorited, count, err := inter.ToggleFavorite(userID, postID)
		if err != nil {
			return c.Error(500, "internal error")
		}
		declarePostsChanged(a, postID)
		return c.JSON(200, map[string]any{"favorited": favorited, "favoriteCount": count})
	}); err != nil {
		return err
	}

	// 发表评论（评论总开关关闭时拒绝：全站停收新评论，读者侧 403）
	if err := a.Post("/posts/:id/comments", loginRoles, func(c *hybrid.ApiCtx) error {
		enabled, err := settings.CommentsEnabled()
		if err != nil {
			return c.Error(500, "internal error")
		}
		if !enabled {
			return c.Error(403, "comments disabled")
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		var in commentInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		postID := mustID(c.Param("id"))
		cm, err := comments.Create(userID, comment.Target{PostID: postID}, in.Content, in.ReplyTo)
		var vErr *commentapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		declarePostsChanged(a, postID)
		invalidateUserPage(a, users, cm.UserID)
		emailAuthSvc.NotifyMentioned(in.ReplyTo, "/posts/"+strconv.FormatInt(postID, 10), excerptOf(in.Content), siteURL)
		return c.JSON(201, toCommentView(cm))
	}); err != nil {
		return err
	}

	// 审核通过评论（仅 author）
	if err := a.Post("/comments/:id/approve", []string{"author"}, func(c *hybrid.ApiCtx) error {
		target, err := comments.Approve(mustID(c.Param("id")))
		switch {
		case errors.Is(err, comment.ErrNotFound):
			return c.Error(404, "comment not found")
		case err != nil:
			return c.Error(500, "internal error")
		}
		if target.MomentID > 0 {
			_ = a.DataChange("/moments")
		} else {
			declarePostsChanged(a, target.PostID)
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 驳回评论（仅 author；reason 必填 ≤200）。approved→rejected 是可见性变化，宿主页必须失效。
	if err := a.Post("/comments/:id/reject", []string{"author"}, func(c *hybrid.ApiCtx) error {
		var in rejectInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		target, err := comments.Reject(mustID(c.Param("id")), in.Reason)
		var vErr *commentapp.ValidationError
		switch {
		case errors.As(err, &vErr):
			return c.Error(400, vErr.Message)
		case errors.Is(err, comment.ErrNotFound):
			return c.Error(404, "comment not found")
		case err != nil:
			return c.Error(500, "internal error")
		}
		invalidateCommentHost(a, target)
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 恢复被驳回评论（仅 author）：读者可见性 不可见→可见，按宿主做失效声明。
	if err := a.Post("/comments/:id/recover", []string{"author"}, func(c *hybrid.ApiCtx) error {
		target, err := comments.Recover(mustID(c.Param("id")))
		switch {
		case errors.Is(err, comment.ErrNotFound):
			return c.Error(404, "comment not found")
		case errors.Is(err, comment.ErrInvalidState):
			return c.Error(400, "comment not in rejected state")
		case err != nil:
			return c.Error(500, "internal error")
		}
		if target.MomentID > 0 {
			_ = a.DataChange("/moments")
		} else {
			declarePostsChanged(a, target.PostID)
		}
		return c.JSON(200, map[string]any{"ok": true})
	}); err != nil {
		return err
	}

	// 删除评论（本人或 author）
	return a.Delete("/comments/:id", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, role, ok := c.User()
		if !ok {
			return c.Error(401, "unauthenticated")
		}
		uid, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		cid := mustID(c.Param("id"))
		// 删除前取评论者（用户页评论数 -1 的失效需要；查不到静默跳过）
		cm, _ := comments.Get(cid)
		target, err := comments.Delete(uid, role, cid)
		switch {
		case errors.Is(err, comment.ErrNotFound):
			return c.Error(404, "comment not found")
		case errors.Is(err, comment.ErrForbidden):
			return c.Error(403, "forbidden")
		case err != nil:
			return c.Error(500, "internal error")
		}
		if target.MomentID > 0 {
			_ = a.DataChange("/moments")
		} else {
			declarePostsChanged(a, target.PostID)
		}
		if cm != nil {
			invalidateUserPage(a, users, cm.UserID)
		}
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// RegisterMomentLikes 注册动态点赞 API：切换点赞与 viewer 已赞列表。
func RegisterMomentLikes(a *hybrid.App, inter *interactionapp.Service) error {
	// viewer 已赞动态 ID 列表（登录用户挂载后拉取）
	if err := a.Get("/moments/interactions", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		ids, err := inter.ViewerMomentLikes(userID)
		if err != nil {
			return c.Error(500, "internal error")
		}
		views := make([]string, 0, len(ids))
		for _, id := range ids {
			views = append(views, strconv.FormatInt(id, 10))
		}
		return c.JSON(200, map[string]any{"liked": views})
	}); err != nil {
		return err
	}

	// 切换动态点赞
	return a.Post("/moments/:id/like", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		momentID := mustID(c.Param("id"))
		liked, count, err := inter.Toggle(userID, "moment", momentID)
		if err != nil {
			return c.Error(500, "internal error")
		}
		_ = a.DataChange("/moments")
		return c.JSON(200, map[string]any{"liked": liked, "likeCount": count})
	})
}

// RegisterMomentComments 注册动态评论 API（列表公开，发表需登录）。
// emailAuthSvc 用于评论 @ 时的邮件通知（异步不阻塞）；siteURL 拼接原文链接（动态以 /moments 落地）。
// settings 提供评论总开关（comments_enabled）：关闭时列表返回空、发表一律 403。
func RegisterMomentComments(a *hybrid.App, comments *commentapp.Service, emailAuthSvc *emailauth.Service, users *userapp.Service, settings *settingsapp.Service, siteURL string) error {
	// 动态评论列表（公开；评论总开关关闭时返回空列表，前端不再渲染评论区）
	if err := a.Get("/moments/:id/comments", nil, func(c *hybrid.ApiCtx) error {
		enabled, err := settings.CommentsEnabled()
		if err != nil {
			return c.Error(500, "internal error")
		}
		if !enabled {
			return c.JSON(200, map[string]any{"comments": []CommentView{}})
		}
		list, err := comments.ListForMoment(mustID(c.Param("id")))
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"comments": toCommentViews(list)})
	}); err != nil {
		return err
	}

	// 发表动态评论（评论总开关关闭时拒绝）
	return a.Post("/moments/:id/comments", loginRoles, func(c *hybrid.ApiCtx) error {
		enabled, err := settings.CommentsEnabled()
		if err != nil {
			return c.Error(500, "internal error")
		}
		if !enabled {
			return c.Error(403, "comments disabled")
		}
		userID, err := currentUserID(c)
		if err != nil {
			return c.Error(401, "unauthenticated")
		}
		var in commentInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		momentID := mustID(c.Param("id"))
		cm, err := comments.Create(userID, comment.Target{MomentID: momentID}, in.Content, in.ReplyTo)
		var vErr *commentapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		_ = a.DataChange("/moments")
		invalidateUserPage(a, users, cm.UserID)
		emailAuthSvc.NotifyMentioned(in.ReplyTo, "/moments", excerptOf(in.Content), siteURL)
		return c.JSON(201, toCommentView(cm))
	})
}

// invalidateUserPage 失效用户个人页（/users/:name 是动态页，只能具体路径失效）。
// 评论/发文改变用户页统计（文章数/评论数）；查询失败静默（不影响请求）。
func invalidateUserPage(a *hybrid.App, users *userapp.Service, userID int64) {
	u, err := users.FindByID(userID)
	if err != nil || u.Username == "" {
		return
	}
	a.InvalidatePage("/users/" + u.Username)
}

// excerptOf 评论内容截断 80 字符（邮件通知引用用）。
func excerptOf(content string) string {
	runes := []rune(content)
	if len(runes) <= 80 {
		return content
	}
	return string(runes[:80]) + "…"
}

// RegisterMe 注册当前用户信息接口（前端评论区/导航栏个人页入口取 viewer 身份）。
func RegisterMe(a *hybrid.App, users *userapp.Service) error {
	return a.Get("/me", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, role, ok := c.User()
		if !ok {
			return c.Error(401, "unauthenticated")
		}
		payload := map[string]any{"userId": userID, "role": role}
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			if u, findErr := users.FindByID(uid); findErr == nil {
				payload["username"] = u.Username
				payload["avatarUrl"] = u.AvatarURL
			}
		}
		return c.JSON(200, payload)
	})
}
