package guestbook

import "errors"

// 领域错误。
var (
	ErrNotFound  = errors.New("guestbook entry not found")
	ErrForbidden = errors.New("forbidden")
)

// Repository 留言板仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// List 返回留言（创建时间倒序，含用户名），limit 上限。
	List(limit int) ([]*Entry, error)
	// Get 按 ID 取留言，不存在返回 ErrNotFound（删除前归属校验用）。
	Get(id int64) (*Entry, error)
	// Create 创建留言并回填 ID 与时间戳。
	Create(e *Entry) error
	// Delete 删除留言，不存在返回 ErrNotFound。
	Delete(id int64) error
}
