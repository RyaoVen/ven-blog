package subscriber

import "errors"

// ErrAlreadySubscribed 邮箱已订阅（幂等语义由应用层转达）。
var ErrAlreadySubscribed = errors.New("already subscribed")

// Repository 订阅者仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Create 记录订阅；邮箱已存在返回 ErrAlreadySubscribed。
	Create(s *Subscriber) error
	// Count 返回订阅者总数（后台统计）。
	Count() (int, error)
}
