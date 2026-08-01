package comment

import "errors"

// 领域错误。
var (
	ErrNotFound = errors.New("comment not found")
	// ErrForbidden 无权删除他人评论。
	ErrForbidden = errors.New("forbidden")
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
	// SetStatus 更新评论状态（审核通过/打回）。
	SetStatus(id int64, status string) error
	// Count 返回评论总数。
	Count() (int, error)
	// Get 按 ID 取评论，不存在返回 ErrNotFound（删除前归属校验用）。
	Get(id int64) (*Comment, error)
	// Create 创建评论并回填 ID 与时间戳。
	Create(c *Comment) error
	// Delete 删除评论，不存在返回 ErrNotFound。
	Delete(id int64) error
}
