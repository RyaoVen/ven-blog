// Package post 文章聚合：实体与领域规则。
package post

import (
	"strings"
	"time"
	"unicode/utf8"
)

// 字段上限（领域规则，校验与归一化共用）。
const (
	maxSummaryLen = 200
	maxCoverLen   = 512
	maxTags       = 8
	maxTagLen     = 24
)

// Post 文章实体。
// AuthorName 是读取模型字段（仓储查询时联表填充），写回时不持久化；
// Tags 写回时经 NormalizeTags 归一化后由仓储持久化（tags/post_tags 表）。
type Post struct {
	ID         int64
	AuthorID   int64
	AuthorName string
	Title      string
	Category   string // 分类（必选，设置页可维护分类列表）
	Summary    string
	Content    string
	CoverURL   string
	Tags       []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NormalizeTags 归一化标签：去首尾空白、去空、超长截断（24 字符）、保序去重、上限 8 个。
func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > maxTagLen {
			name = string([]rune(name)[:maxTagLen])
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
		if len(out) >= maxTags {
			break
		}
	}
	return out
}

// Validate 校验文章字段，返回错误消息（空串表示通过）。
func Validate(title, category, content, summary, coverURL string) string {
	if strings.TrimSpace(title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(category) == "" {
		return "category is required"
	}
	if strings.TrimSpace(content) == "" {
		return "content is required"
	}
	if utf8.RuneCountInString(summary) > maxSummaryLen {
		return "summary too long (max 200)"
	}
	if utf8.RuneCountInString(coverURL) > maxCoverLen {
		return "cover url too long (max 512)"
	}
	return ""
}

// Apply 应用编辑：更新标题/分类/摘要/正文/封面与标签（时间戳由仓储层维护）。
func (p *Post) Apply(title, category, content, summary, coverURL string, tags []string) {
	p.Title = strings.TrimSpace(title)
	p.Category = strings.TrimSpace(category)
	p.Summary = strings.TrimSpace(summary)
	p.Content = content
	p.CoverURL = strings.TrimSpace(coverURL)
	p.Tags = NormalizeTags(tags)
}
