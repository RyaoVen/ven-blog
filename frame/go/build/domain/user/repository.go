package user

import (
	"errors"
	"time"
)

// DayCount 某日计数（注册/发布聚合通用）。
type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// 领域错误。
var (
	ErrNotFound      = errors.New("user not found")
	ErrUsernameTaken = errors.New("username taken")
	ErrEmailTaken    = errors.New("email taken")
)

// Repository 用户仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// FindByUsername 按用户名查找，不存在返回 ErrNotFound。
	FindByUsername(username string) (*User, error)
	// FindByID 按 ID 查找，不存在返回 ErrNotFound。
	FindByID(id int64) (*User, error)
	// FindByRole 按角色取第一个用户（定位 author 用，不受改名影响），不存在返回 ErrNotFound。
	FindByRole(role Role) (*User, error)
	// UpdateUsername 修改用户名；唯一键冲突返回 ErrUsernameTaken。
	UpdateUsername(userID int64, username string) error
	// FindByEmail 按邮箱查找（验证码登录），不存在返回 ErrNotFound。
	FindByEmail(email string) (*User, error)
	// UpdateEmail 绑定/修改邮箱；唯一键冲突返回 ErrEmailTaken。
	UpdateEmail(userID int64, email string) error
	// DailyRegistrations 近 days 天每日注册数（不足补零，日期升序）。
	DailyRegistrations(days int) ([]DayCount, error)
	// CountSince 某时刻之后注册的用户数（增量对比用）。
	CountSince(t time.Time) (int, error)
	// Create 创建用户；用户名冲突返回 ErrUsernameTaken。
	Create(u *User) error
	// Count 返回用户总数（种子判定用）。
	Count() (int, error)
	// UpdatePassword 更新密码哈希（设置页改密码）。
	UpdatePassword(userID int64, passwordHash string) error
	// UpdateProfile 更新简介与头像（设置页资料编辑）。
	UpdateProfile(userID int64, bio, avatarURL string) error
	// CountPosts 返回用户发布的文章数。
	CountPosts(userID int64) (int, error)
	// CountComments 返回用户发表的评论数。
	CountComments(userID int64) (int, error)
}
