// Package post 文章聚合：实体与领域规则。
package post

import (
	"strings"
	"time"
)

// Post 文章实体。
// AuthorName 与 Tags 是读取模型字段（仓储查询时联表填充），写回时不持久化。
type Post struct {
	ID         int64
	AuthorID   int64
	AuthorName string
	Title      string
	Summary    string
	Content    string
	CoverURL   string
	Tags       []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate 校验标题与正文，返回错误消息（空串表示通过）。
func Validate(title, content string) string {
	if strings.TrimSpace(title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(content) == "" {
		return "content is required"
	}
	return ""
}

// Apply 应用编辑：更新标题与正文（时间戳由仓储层维护）。
func (p *Post) Apply(title, content string) {
	p.Title = strings.TrimSpace(title)
	p.Content = content
}
