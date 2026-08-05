package comment

import "errors"

// 领域错误。
var (
	ErrNotFound = errors.New("comment not found")
	// ErrForbidden 无权删除他人评论。
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidState Recover 前置校验用（非 rejected 状态不可恢复）。
	ErrInvalidState = errors.New("comment not in rejected state")
)

// Repository 评论仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// ListByPost 返回文章下的评论（创建时间倒序，含用户名）。
	ListByPost(postID int64) ([]*Comment, error)
	// ListByMoment 返回动态下的评论（创建时间倒序，含用户名）。
	ListByMoment(momentID int64) ([]*Comment, error)
	// MomentCommentCounts 动态评论数分组统计（/moments 页展示用）。
	MomentCommentCounts() (map[int64]int, error)
	// ListAll 返回全站评论（创建时间倒序，含用户名与所属文章标题），limit 上限。
	ListAll(limit int) ([]*Comment, error)
	// ListPending 返回待审核评论（创建时间正序，先审先到的）。
	ListPending() ([]*Comment, error)
	// ListUnreviewedPending 返回 AI 未判的待审评论（worker 队列；AI 判过 uncertain 的不在内）。
	ListUnreviewedPending() ([]*Comment, error)
	// ClaimAIReview 原子抢占 AI 审核权（仅 pending 且 AI 未审行；返回是否抢到）。
	// 多实例 worker 并发时同一条只有一方抢到（其余 false 跳过），杜绝重复审核。
	ClaimAIReview(id int64) (bool, error)
	// UnclaimAIReview 回滚抢占（LLM 判定/写库失败后释放，保持"失败下轮重审"）。
	UnclaimAIReview(id int64) error
	// ListRejected 返回被驳回评论（创建时间正序）。
	ListRejected() ([]*Comment, error)
	// SetStatus 更新评论状态（审核通过/打回），任何写入同时清空驳回原因
	// （保持"仅 rejected 有 reason"不变量）。
	SetStatus(id int64, status string) error
	// SetRejected 驳回评论并记录驳回原因。
	SetRejected(id int64, reason string) error
	// Count 返回评论总数。
	Count() (int, error)
	// Get 按 ID 取评论，不存在返回 ErrNotFound（删除前归属校验用）。
	Get(id int64) (*Comment, error)
	// Create 创建评论并回填 ID 与时间戳。
	Create(c *Comment) error
	// Delete 删除评论，不存在返回 ErrNotFound。
	Delete(id int64) error
}
