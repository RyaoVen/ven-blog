// Package guestbook 留言板聚合：作者主页留言。
package guestbook

import (
	"strings"
	"time"
)

// Entry 一条留言。Username 是读取模型字段（仓储联表填充），写回时不持久化。
type Entry struct {
	ID        int64
	UserID    int64
	Username  string
	Content   string
	CreatedAt time.Time
}

// Validate 校验留言内容，返回错误消息（空串表示通过）。
// 领域规则：去空白后非空，且不超过 500 字符。
func Validate(content string) string {
	if strings.TrimSpace(content) == "" {
		return "content is required"
	}
	if len(content) > 500 {
		return "content too long (max 500)"
	}
	return ""
}
