package image

import "errors"

// ErrNotFound 图片不存在。
var ErrNotFound = errors.New("image not found")

// Repository 图片仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Create 新建图片并回填 ID 与时间戳。
	Create(img *Image) error
	// Get 按 ID 取图片（含二进制数据），不存在返回 ErrNotFound。
	Get(id int64) (*Image, error)
}
