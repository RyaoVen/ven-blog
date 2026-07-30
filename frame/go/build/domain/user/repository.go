package user

import "errors"

// 领域错误。
var (
	ErrNotFound      = errors.New("user not found")
	ErrUsernameTaken = errors.New("username taken")
)

// Repository 用户仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// FindByUsername 按用户名查找，不存在返回 ErrNotFound。
	FindByUsername(username string) (*User, error)
	// FindByID 按 ID 查找，不存在返回 ErrNotFound。
	FindByID(id int64) (*User, error)
	// Create 创建用户；用户名冲突返回 ErrUsernameTaken。
	Create(u *User) error
	// Count 返回用户总数（种子判定用）。
	Count() (int, error)
	// CountPosts 返回用户发布的文章数。
	CountPosts(userID int64) (int, error)
	// CountComments 返回用户发表的评论数。
	CountComments(userID int64) (int, error)
}
