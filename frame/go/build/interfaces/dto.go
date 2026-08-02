// Package interfaces 接口层：hybrid 适配——页面/API/认证注册，DTO 与错误映射。
// 依赖方向：interfaces → application → domain；infrastructure 由组装根（build.Register）注入。
package interfaces

import (
	"strconv"
	"time"

	"ven_hybird/build/domain/post"
)

// PostView 是文章对外的 JSON 视图（ID 以字符串下发，与前端既有类型契约一致）。
type PostView struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Summary    string    `json:"summary"`
	Content    string    `json:"content"`
	CoverURL   string    `json:"coverUrl"`
	AuthorName string    `json:"authorName"`
	Tags       []string  `json:"tags"`
	Pinned     bool      `json:"pinned"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// toPostView 领域实体 → JSON 视图。
func toPostView(p *post.Post) PostView {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return PostView{
		ID:         strconv.FormatInt(p.ID, 10),
		Title:      p.Title,
		Category:   p.Category,
		Summary:    p.Summary,
		Content:    p.Content,
		CoverURL:   p.CoverURL,
		AuthorName: p.AuthorName,
		Tags:       tags,
		Pinned:     p.Pinned,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// toPostViews 批量转换。
func toPostViews(posts []*post.Post) []PostView {
	views := make([]PostView, 0, len(posts))
	for _, p := range posts {
		views = append(views, toPostView(p))
	}
	return views
}

// parseID 解析路径参数中的文章 ID。
func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}
