// Package postapp 文章用例服务：列表/详情/创建/编辑/删除。
// 应用层只做用例编排与领域规则校验，不依赖 hybrid/框架（失效声明由接口层协调）。
package postapp

import (
	"regexp"
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
	Category string
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

// Search 按关键词搜索文章；trim 后为空直接返回空切片（不打数据库）。
func (s *Service) Search(q string) ([]*post.Post, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []*post.Post{}, nil
	}
	return s.repo.Search(q, 50)
}

// Get 文章详情。
func (s *Service) Get(id int64) (*post.Post, error) {
	return s.repo.Get(id)
}

// ListByAuthor 指定作者的文章（作者主页用，创建时间倒序）。
func (s *Service) ListByAuthor(authorID int64) ([]*post.Post, error) {
	return s.repo.ListByAuthor(authorID)
}

// firstImagePattern 提取正文 markdown 第一张图片的 URL（封面留空兜底用）。
var firstImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)

// firstImageOf 取正文第一张图片 URL（无图返回空串）。
func firstImageOf(content string) string {
	m := firstImagePattern.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// Create 发文：领域校验 + 封面兜底（留空取正文首图）+ 落库。
func (s *Service) Create(authorID int64, in PostInput) (*post.Post, error) {
	if msg := post.Validate(in.Title, in.Category, in.Content, in.Summary, in.CoverURL); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	if in.CoverURL == "" {
		in.CoverURL = firstImageOf(in.Content)
	}
	p := &post.Post{AuthorID: authorID}
	p.Apply(in.Title, in.Category, in.Content, in.Summary, in.CoverURL, in.Tags)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Update 编辑文章（封面留空同样兜底首图）。
func (s *Service) Update(id int64, in PostInput) (*post.Post, error) {
	if msg := post.Validate(in.Title, in.Category, in.Content, in.Summary, in.CoverURL); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	if in.CoverURL == "" {
		in.CoverURL = firstImageOf(in.Content)
	}
	p := &post.Post{ID: id}
	p.Apply(in.Title, in.Category, in.Content, in.Summary, in.CoverURL, in.Tags)
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

// SiteStats 站点统计（首页仪表盘用）。
func (s *Service) SiteStats() (posts int, totalChars int, err error) {
	return s.repo.Stats()
}

// ListFavorites 用户收藏的文章（个人页本人可见）。
func (s *Service) ListFavorites(userID int64) ([]*post.Post, error) {
	return s.repo.ListFavorites(userID)
}

// DailyPublication 近 days 天每日发布篇数与字数（日历热力图用）。
func (s *Service) DailyPublication(days int) ([]post.DayPublication, error) {
	return s.repo.DailyPublication(days)
}

// CategoryCounts 各分类文章数（分类雷达图用）。
func (s *Service) CategoryCounts() ([]post.CategoryCount, error) {
	return s.repo.CategoryCounts()
}

// CategoryCount 某分类文章数（删除分类前检查）。
func (s *Service) CategoryCount(category string) (int, error) {
	return s.repo.CountByCategory(category)
}

// MigrateCategory 把某分类全部文章迁移到目标分类（删除分类用）。
func (s *Service) MigrateCategory(from, to string) error {
	return s.repo.UpdateCategory(from, to)
}
