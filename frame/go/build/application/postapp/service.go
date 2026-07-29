// Package postapp 文章用例服务：列表/详情/创建/编辑/删除。
// 应用层只做用例编排与领域规则校验，不依赖 hybrid/框架（失效声明由接口层协调）。
package postapp

import (
	"ven_hybird/build/domain/post"
)

// Service 文章用例服务。
type Service struct {
	repo post.Repository
}

// NewService 构造文章用例服务。
func NewService(repo post.Repository) *Service {
	return &Service{repo: repo}
}

// ListRecent 最近文章（首页用），limit <= 0 表示全部。
func (s *Service) ListRecent(limit int) ([]*post.Post, error) {
	posts, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
}

// Get 文章详情。
func (s *Service) Get(id int64) (*post.Post, error) {
	return s.repo.Get(id)
}

// Create 发文：领域校验 + 落库。
func (s *Service) Create(authorID int64, title, content string) (*post.Post, error) {
	if msg := post.Validate(title, content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	p := &post.Post{AuthorID: authorID}
	p.Apply(title, content)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Update 编辑文章。
func (s *Service) Update(id int64, title, content string) (*post.Post, error) {
	if msg := post.Validate(title, content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	p := &post.Post{ID: id}
	p.Apply(title, content)
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	return s.repo.Get(id)
}

// Delete 删除文章。
func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
