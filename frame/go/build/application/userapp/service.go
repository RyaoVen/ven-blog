// Package userapp 用户用例服务：注册与登录认证（bcrypt）。
package userapp

import (
	"errors"

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

// Register 注册新用户（角色 reader）：领域校验 → 查重 → bcrypt 哈希 → 落库。
func (s *Service) Register(username, password string) (*user.User, error) {
	if msg := user.ValidateCredentials(username, password); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &user.User{Username: username, PasswordHash: string(hash), Role: user.RoleReader}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	return u, nil
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
