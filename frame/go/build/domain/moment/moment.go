// Package moment 动态（说说）聚合：实体与领域规则。
package moment

import (
	"strings"
	"time"
	"unicode/utf8"
)

// MaxContentLen 动态正文最大字符数。
const MaxContentLen = 1000

// Moment 动态实体。
// AuthorName 是读取模型字段（仓储查询时联表填充），写回时不持久化；
// Pinned 置顶标记（0/1，排序加分项，写回经仓储 SetPinned 独立维护）。
type Moment struct {
	ID         int64
	AuthorID   int64
	AuthorName string
	Content    string
	Pinned     bool
	CreatedAt  time.Time
}

// Validate 校验正文：去空白后非空且不超过 MaxContentLen 字符，返回错误消息（空串表示通过）。
func Validate(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "content is required"
	}
	if utf8.RuneCountInString(trimmed) > MaxContentLen {
		return "content exceeds 1000 characters"
	}
	return ""
}

// Apply 应用编辑：去除首尾空白（时间戳由仓储层维护）。
func (m *Moment) Apply(content string) {
	m.Content = strings.TrimSpace(content)
}
