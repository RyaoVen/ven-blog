// Package guestbookapp 留言板用例服务。
package guestbookapp

import (
	"ven_hybird/build/domain/guestbook"
)

// Service 留言板用例服务。
type Service struct {
	repo guestbook.Repository
}

// NewService 构造留言板用例服务。
func NewService(repo guestbook.Repository) *Service {
	return &Service{repo: repo}
}

// List 留言列表（最近 limit 条）。
func (s *Service) List(limit int) ([]*guestbook.Entry, error) {
	return s.repo.List(limit)
}

// Create 发表留言。
func (s *Service) Create(userID int64, content string) (*guestbook.Entry, error) {
	if msg := guestbook.Validate(content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	e := &guestbook.Entry{UserID: userID, Content: content}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Delete 删除留言：本人或 author 角色可删，其余 ErrForbidden。
func (s *Service) Delete(userID int64, role string, id int64) error {
	e, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	if e.UserID != userID && role != "author" {
		return guestbook.ErrForbidden
	}
	return s.repo.Delete(id)
}

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
