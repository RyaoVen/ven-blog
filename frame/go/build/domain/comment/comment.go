// Package comment 评论聚合：实体与领域规则。
package comment

import (
	"strings"
	"time"
)

// Comment 评论实体。Username 是读取模型字段（仓储联表填充），写回时不持久化。
type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Username  string
	Content   string
	CreatedAt time.Time
}

// Validate 校验评论内容，返回错误消息（空串表示通过）。
// 领域规则：去空白后非空，且不超过 2000 字符。
func Validate(content string) string {
	if strings.TrimSpace(content) == "" {
		return "content is required"
	}
	if len(content) > 2000 {
		return "content too long (max 2000)"
	}
	return ""
}
