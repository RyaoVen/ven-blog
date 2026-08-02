// Package moderation 审核语义聚合：跨评论/留言板两个宿主聚合的 AI 审核判定。
// 不依赖任何外层；实现（infrastructure/llm）由组装根注入。
package moderation

import "context"

// 审核动作。
const (
	ActionApprove = "approve" // 放行
	ActionReject  = "reject"  // 驳回（记原因）
	ActionPending = "pending" // 不确定，保持待审
)

// Verdict 审核判定。
type Verdict struct {
	Action string // 取 ActionApprove / ActionReject / ActionPending
	Reason string // reject 时的驳回原因（approve/pending 可为空）
}

// HostKind 宿主类型。
type HostKind string

const (
	HostComment   HostKind = "comment"    // 评论（宿主为文章或动态）
	HostGuestbook HostKind = "guestbook"  // 留言板留言
)

// Request 审核输入：内容 + 宿主上下文 + 回复对象。
type Request struct {
	Host      HostKind // 宿主类型（评论/留言板）
	HostTitle string   // 宿主标题：文章标题 / 动态文案摘要 / 留言板固定为作者主页
	Content   string   // 待审核内容（领域已保证 ≤2000/≤500 字符，见领域 Validate）
	ReplyTo   string   // 回复目标用户名（评论 @ 场景，可空）
}

// Moderator 审核判定器（基础设施层实现）。
// 返回 error 表示本次无法判定（网络/超时/解析失败），由应用层决定重试与挂起。
type Moderator interface {
	Review(ctx context.Context, req Request) (Verdict, error)
}
