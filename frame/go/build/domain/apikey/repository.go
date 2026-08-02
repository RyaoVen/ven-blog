package apikey

import "time"

// Repository API 密钥仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Create 新建密钥（user_id/name/key_hash/prefix），回填 ID 与 CreatedAt。
	Create(k *ApiKey) error
	// FindByHash 按 hash 精确查找（唯一索引），不存在返回 ErrNotFound。
	FindByHash(hash string) (*ApiKey, error)
	// ListByUser 返回某用户全部密钥（创建时间倒序，含已吊销）。
	ListByUser(userID int64) ([]*ApiKey, error)
	// Revoke 吊销（写 revoked_at）：仅限本人（user_id 匹配且未吊销）。
	// 不存在 / 已吊销 / 非本人统一返回 ErrNotFound（不泄露 key 存在性）。
	Revoke(userID, id int64) error
	// UpdateLastUsedAt 写最后使用时间（鉴权成功后调用）。
	UpdateLastUsedAt(id int64, t time.Time) error
}
