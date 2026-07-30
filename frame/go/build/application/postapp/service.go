// Package postapp 文章用例服务：列表/详情/创建/编辑/删除。
// 应用层只做用例编排与领域规则校验，不依赖 hybrid/框架（失效声明由接口层协调）。
package postapp

import (
	"strings"

	"ven_hybird/build/domain/post"
)

// defaultPageSize 是列表分页的默认每页条数。
const defaultPageSize = 10

// Service 文章用例服务。
type Service struct {
	repo post.Repository
}

// NewService 构造文章用例服务。
func NewService(repo post.Repository) *Service {
	return &Service{repo: repo}
}

// PostInput 是创建/更新文章的用例入参。
type PostInput struct {
	Title    string
	Content  string
	Summary  string
	CoverURL string
	Tags     []string
}

// ListFilter 是列表筛选条件：标签 + 分页。
type ListFilter struct {
	Tag      string
	Page     int
	PageSize int
}

// Paged 是一页文章与分页信息。
type Paged struct {
	Posts    []*post.Post
	Total    int
	Page     int
	PageSize int
}

// ListRecent 最近文章（首页用），limit <= 0 表示全部。
func (s *Service) ListRecent(limit int) ([]*post.Post, error) {
	posts, _, err := s.repo.ListPaged("", 1, limit)
	return posts, err
}

// List 分页列表：Page 归一为 >= 1，PageSize <= 0 时默认 10，Tag 去首尾空白。
func (s *Service) List(filter ListFilter) (*Paged, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	posts, total, err := s.repo.ListPaged(strings.TrimSpace(filter.Tag), page, pageSize)
	if err != nil {
		return nil, err
	}
	return &Paged{Posts: posts, Total: total, Page: page, PageSize: pageSize}, nil
}

// AllTags 全量标签（列表页筛选条用）。
func (s *Service) AllTags() ([]string, error) {
	return s.repo.AllTags()
}

// Get 文章详情。
func (s *Service) Get(id int64) (*post.Post, error) {
	return s.repo.Get(id)
}

// Create 发文：领域校验 + 落库。
func (s *Service) Create(authorID int64, in PostInput) (*post.Post, error) {
	if msg := post.Validate(in.Title, in.Content, in.Summary, in.CoverURL); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	p := &post.Post{AuthorID: authorID}
	p.Apply(in.Title, in.Content, in.Summary, in.CoverURL, in.Tags)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Update 编辑文章。
func (s *Service) Update(id int64, in PostInput) (*post.Post, error) {
	if msg := post.Validate(in.Title, in.Content, in.Summary, in.CoverURL); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	p := &post.Post{ID: id}
	p.Apply(in.Title, in.Content, in.Summary, in.CoverURL, in.Tags)
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
