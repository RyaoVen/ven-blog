// Package emailcode 邮箱验证码聚合。
package emailcode

import "time"

// Entry 一条验证码记录（哈希存储）。
type Entry struct {
	ID        int64
	Email     string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

// Repository 验证码仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Create 新建验证码（同邮箱先清旧码）。
	Create(email, codeHash string, expiresAt time.Time) error
	// Latest 取邮箱最新一条验证码，不存在返回 nil（不报错）。
	Latest(email string) (*Entry, error)
	// IncrAttempts 尝试次数 +1。
	IncrAttempts(id int64) error
	// Delete 删除验证码（验证成功或作废）。
	Delete(id int64) error
}
