// Package guestbook 留言板聚合：作者主页留言。
package guestbook

import (
	"strings"
	"time"
)

// 留言状态。
const (
	StatusApproved = "approved"
	StatusPending  = "pending"
	StatusRejected = "rejected"
)

// Entry 一条留言。Username 是读取模型字段（仓储联表填充），写回时不持久化。
type Entry struct {
	ID             int64
	UserID         int64
	Username       string
	Content        string
	Status         string // approved | pending | rejected（开启审核后新留言为 pending）
	RejectedReason string // 驳回原因：仅 Status==rejected 时非空；其余状态由仓储保证清空
	CreatedAt      time.Time
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

// ValidateRejectedReason 校验驳回原因，返回错误消息（空串表示通过）。
// 领域规则：去空白后非空，且不超过 200 字符。
func ValidateRejectedReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "reason is required"
	}
	if len(reason) > 200 {
		return "reason too long (max 200)"
	}
	return ""
}
