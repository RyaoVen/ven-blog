// 互动接口：评论/点赞/收藏（JSON，框架自动 /api 前缀）。
package interfaces

import (
	"errors"
	"strconv"
	"time"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/interactionapp"
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

// loginRoles 评论/点赞/收藏对全部登录用户开放（扁平角色下两个都列）。
var loginRoles = []string{"reader", "author"}

// RegisterInteractions 注册互动 API。
// 详情页是 ISR 共享物化——viewer 个性化状态一律走本文件的 JSON 接口，不进页面 initialState。
func RegisterInteractions(a *hybrid.App, comments *commentapp.Service, inter *interactionapp.Service) error {
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

	// 发表评论
	if err := a.Post("/posts/:id/comments", loginRoles, func(c *hybrid.ApiCtx) error {
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
		return c.JSON(201, toCommentView(cm))
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
		target, err := comments.Delete(uid, role, mustID(c.Param("id")))
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
		return c.JSON(200, map[string]any{"ok": true})
	})
}

// RegisterMomentComments 注册动态评论 API（列表公开，发表需登录）。
func RegisterMomentComments(a *hybrid.App, comments *commentapp.Service) error {
	// 动态评论列表（公开）
	if err := a.Get("/moments/:id/comments", nil, func(c *hybrid.ApiCtx) error {
		list, err := comments.ListForMoment(mustID(c.Param("id")))
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"comments": toCommentViews(list)})
	}); err != nil {
		return err
	}

	// 发表动态评论
	return a.Post("/moments/:id/comments", loginRoles, func(c *hybrid.ApiCtx) error {
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
		return c.JSON(201, toCommentView(cm))
	})
}

// RegisterMe 注册当前用户信息接口（前端评论区等场景取 viewer 身份）。
func RegisterMe(a *hybrid.App) error {
	return a.Get("/me", loginRoles, func(c *hybrid.ApiCtx) error {
		userID, role, ok := c.User()
		if !ok {
			return c.Error(401, "unauthenticated")
		}
		return c.JSON(200, map[string]any{"userId": userID, "role": role})
	})
}
