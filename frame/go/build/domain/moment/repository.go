package moment

import "errors"

// ErrNotFound 动态不存在。
var ErrNotFound = errors.New("moment not found")

// Repository 动态仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// List 返回最近动态（创建时间倒序，含作者名），limit <= 0 表示全部。
	List(limit int) ([]*Moment, error)
	// Create 新建动态并回填 ID 与时间戳。
	Create(m *Moment) error
	// Delete 删除动态，不存在返回 ErrNotFound。
	Delete(id int64) error
}
