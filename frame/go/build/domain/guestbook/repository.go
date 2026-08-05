package guestbook

import "errors"

// 领域错误。
var (
	ErrNotFound  = errors.New("guestbook entry not found")
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidState Recover 前置校验用（非 rejected 状态不可恢复）。
	ErrInvalidState = errors.New("guestbook entry not in rejected state")
)

// Repository 留言板仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// List 返回公开留言（仅 approved，创建时间倒序，含用户名），limit 上限。
	List(limit int) ([]*Entry, error)
	// ListAll 返回全量留言（含全部状态，创建时间倒序），limit 上限（后台管理用）。
	ListAll(limit int) ([]*Entry, error)
	// ListPending 返回待审核留言（创建时间正序，先审先到的）。
	ListPending() ([]*Entry, error)
	// ListUnreviewedPending 返回 AI 未判的待审留言（worker 队列；AI 判过 uncertain 的不在内）。
	ListUnreviewedPending() ([]*Entry, error)
	// ClaimAIReview 原子抢占 AI 审核权（仅 pending 且 AI 未审行；返回是否抢到）。
	// 多实例 worker 并发时同一条只有一方抢到（其余 false 跳过），杜绝重复审核。
	ClaimAIReview(id int64) (bool, error)
	// UnclaimAIReview 回滚抢占（LLM 判定/写库失败后释放，保持"失败下轮重审"）。
	UnclaimAIReview(id int64) error
	// ListRejected 返回被驳回留言（创建时间正序）。
	ListRejected() ([]*Entry, error)
	// Get 按 ID 取留言，不存在返回 ErrNotFound（删除前归属校验用）。
	Get(id int64) (*Entry, error)
	// Create 创建留言并回填 ID 与时间戳。
	Create(e *Entry) error
	// Delete 删除留言，不存在返回 ErrNotFound。
	Delete(id int64) error
	// SetStatus 更新留言状态（审核通过/打回），任何写入同时清空驳回原因
	// （保持"仅 rejected 有 reason"不变量）。
	SetStatus(id int64, status string) error
	// SetRejected 驳回留言并记录驳回原因。
	SetRejected(id int64, reason string) error
}
