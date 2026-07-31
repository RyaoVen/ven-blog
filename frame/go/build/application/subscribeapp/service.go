// Package subscribeapp 订阅用例服务。
package subscribeapp

import (
	"strings"

	"ven_hybird/build/domain/subscriber"
)

// Service 订阅用例服务。
type Service struct {
	repo subscriber.Repository
}

// NewService 构造订阅用例服务。
func NewService(repo subscriber.Repository) *Service {
	return &Service{repo: repo}
}

// Subscribe 记录邮箱订阅；重复订阅返回 already=true（幂等，不报错）。
func (s *Service) Subscribe(email string) (already bool, err error) {
	email = strings.TrimSpace(email)
	if msg := subscriber.ValidateEmail(email); msg != "" {
		return false, &ValidationError{Message: msg}
	}
	err = s.repo.Create(&subscriber.Subscriber{Email: email})
	if err == subscriber.ErrAlreadySubscribed {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
