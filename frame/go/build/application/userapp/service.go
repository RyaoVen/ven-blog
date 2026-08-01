// Package userapp 用户用例服务：注册与登录认证（bcrypt）。
package userapp

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ven_hybird/build/domain/user"
)

// Service 用户用例服务。
type Service struct {
	repo user.Repository
}

// NewService 构造用户用例服务。
func NewService(repo user.Repository) *Service {
	return &Service{repo: repo}
}

// Register 注册新用户（角色 reader，邮箱必填）：领域校验 → bcrypt 哈希 → 落库。
func (s *Service) Register(username, password, email string) (*user.User, error) {
	if msg := user.ValidateCredentials(username, password, email); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &user.User{Username: username, PasswordHash: string(hash), Role: user.RoleReader, Email: email}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateEmail 绑定/修改邮箱（唯一性冲突返回 user.ErrEmailTaken）。
func (s *Service) UpdateEmail(userID int64, email string) error {
	return s.repo.UpdateEmail(userID, email)
}

// Authenticate 登录认证：查用户 + bcrypt 比对；失败统一返回 ErrInvalidCredentials（不泄露是哪一环错）。
func (s *Service) Authenticate(username, password string) (*user.User, error) {
	u, err := s.repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// Profile 用户主页数据：用户实体 + 作品统计。
type Profile struct {
	User     *user.User
	Posts    int
	Comments int
}

// GetProfile 用户主页用例：按用户名取用户并聚合文章/评论统计；不存在返回 user.ErrNotFound。
func (s *Service) GetProfile(username string) (*Profile, error) {
	u, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	posts, err := s.repo.CountPosts(u.ID)
	if err != nil {
		return nil, err
	}
	comments, err := s.repo.CountComments(u.ID)
	if err != nil {
		return nil, err
	}
	return &Profile{User: u, Posts: posts, Comments: comments}, nil
}

// ErrInvalidCredentials 用户名或密码错误。
var ErrInvalidCredentials = errors.New("invalid credentials")

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Count 用户总数（后台统计）。
func (s *Service) Count() (int, error) {
	return s.repo.Count()
}

// ChangePassword 修改密码：先校验旧密码，再写入新哈希。
func (s *Service) ChangePassword(userID int64, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return &ValidationError{Message: "password must be at least 6 characters"}
	}
	u, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)) != nil {
		return &ValidationError{Message: "old password incorrect"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(userID, string(hash))
}

// UpdateProfile 更新简介与头像。
func (s *Service) UpdateProfile(userID int64, bio, avatarURL string) error {
	if len([]rune(bio)) > 200 {
		return &ValidationError{Message: "bio too long (max 200)"}
	}
	return s.repo.UpdateProfile(userID, bio, avatarURL)
}

// UpdateUsername 修改用户名（2-32 字符；冲突返回 user.ErrUsernameTaken）。
func (s *Service) UpdateUsername(userID int64, username string) error {
	if len(username) < 2 || len(username) > 32 {
		return &ValidationError{Message: "username must be 2-32 characters"}
	}
	return s.repo.UpdateUsername(userID, username)
}

// FindByID 按 ID 取用户。
func (s *Service) FindByID(userID int64) (*user.User, error) {
	return s.repo.FindByID(userID)
}

// DailyRegistrations 近 days 天每日注册数（仪表盘用户增长折线用）。
func (s *Service) DailyRegistrations(days int) ([]user.DayCount, error) {
	return s.repo.DailyRegistrations(days)
}

// CountSince 某时刻之后注册的用户数（增量对比用）。
func (s *Service) CountSince(t time.Time) (int, error) {
	return s.repo.CountSince(t)
}
