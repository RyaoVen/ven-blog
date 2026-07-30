package post

import "errors"

// ErrNotFound 文章不存在。
var ErrNotFound = errors.New("post not found")

// Repository 文章仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// ListPaged 分页返回文章（创建时间倒序，含作者名与标签），并返回符合过滤条件的总数；
	// tag 非空时按标签过滤，pageSize <= 0 表示不分页（返回全部）。
	ListPaged(tag string, page, pageSize int) ([]*Post, int, error)
	// Get 按 ID 取文章（含标签），不存在返回 ErrNotFound。
	Get(id int64) (*Post, error)
	// Search 按关键词匹配标题或正文（创建时间倒序，含作者名）；limit 为返回上限（实现归一到 [1, 50]）。
	Search(query string, limit int) ([]*Post, error)
	// Create 新建文章（含标签）并回填 ID 与时间戳。
	Create(p *Post) error
	// Update 更新标题/摘要/正文/封面与标签，不存在返回 ErrNotFound。
	Update(p *Post) error
	// Delete 删除文章，不存在返回 ErrNotFound。
	Delete(id int64) error
	// AllTags 返回全部标签名（字典序）。
	AllTags() ([]string, error)
}
