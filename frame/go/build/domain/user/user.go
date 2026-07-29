// Package user 用户聚合：实体、角色值对象与领域规则。
package user

import "time"

// Role 是用户角色（与框架鉴权角色同名对应：reader/author）。
type Role string

const (
	RoleReader Role = "reader"
	RoleAuthor Role = "author"
)

// String 返回角色名字符串。
func (r Role) String() string { return string(r) }

// User 用户实体。
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         Role
	Bio          string
	AvatarURL    string
	CreatedAt    time.Time
}

// ValidateCredentials 校验注册凭据，返回错误消息（空串表示通过）。
// 领域规则：用户名 2-32 字符；密码至少 6 位。
func ValidateCredentials(username, password string) string {
	if len(username) < 2 || len(username) > 32 {
		return "username must be 2-32 characters"
	}
	if len(password) < 6 {
		return "password must be at least 6 characters"
	}
	return ""
}
