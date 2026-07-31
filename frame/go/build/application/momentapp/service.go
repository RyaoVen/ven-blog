// Package momentapp 动态用例服务：列表/发布/删除。
// 应用层只做用例编排与领域规则校验，不依赖 hybrid/框架（失效声明由接口层协调）。
package momentapp

import (
	"ven_hybird/build/domain/moment"
)

// ListLimit 动态列表条数上限（时间线只展示最近一屏）。
const ListLimit = 50

// Service 动态用例服务。
type Service struct {
	repo moment.Repository
}

// NewService 构造动态用例服务。
func NewService(repo moment.Repository) *Service {
	return &Service{repo: repo}
}

// List 最近动态（创建时间倒序，最多 ListLimit 条）。
func (s *Service) List() ([]*moment.Moment, error) {
	return s.repo.List(ListLimit)
}

// Create 发布动态：领域校验 + 落库。
func (s *Service) Create(authorID int64, content string) (*moment.Moment, error) {
	if msg := moment.Validate(content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	m := &moment.Moment{AuthorID: authorID}
	m.Apply(content)
	if err := s.repo.Create(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete 删除动态。
func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Count 动态总数（后台统计）。
func (s *Service) Count() (int, error) {
	return s.repo.Count()
}
