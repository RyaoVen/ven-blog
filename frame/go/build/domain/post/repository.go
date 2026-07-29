package post

import "errors"

// ErrNotFound 文章不存在。
var ErrNotFound = errors.New("post not found")

// Repository 文章仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// List 返回全部文章（创建时间倒序，含作者名）。
	List() ([]*Post, error)
	// Get 按 ID 取文章，不存在返回 ErrNotFound。
	Get(id int64) (*Post, error)
	// Create 新建文章并回填 ID 与时间戳。
	Create(p *Post) error
	// Update 更新标题与正文，不存在返回 ErrNotFound。
	Update(p *Post) error
	// Delete 删除文章，不存在返回 ErrNotFound。
	Delete(id int64) error
}
