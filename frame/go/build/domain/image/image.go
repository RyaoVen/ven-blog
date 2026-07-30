// Package image 图片聚合：实体与领域规则（MIME 白名单、大小上限）。
package image

import "time"

// MaxSize 是上传图片的字节数上限（5MB）。
const MaxSize = 5 << 20

// Image 图片实体。MySQL BLOB 后端下 Data 存二进制（images.data LONGBLOB）。
type Image struct {
	ID         int64
	UploaderID int64
	Filename   string
	Mime       string
	Data       []byte
	CreatedAt  time.Time
}

// AllowedMime 报告 mime 是否在允许上传的图片类型白名单内。
func AllowedMime(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	}
	return false
}
