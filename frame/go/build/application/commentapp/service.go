// Package commentapp 评论用例服务。
package commentapp

import (
	"ven_hybird/build/domain/comment"
)

// Service 评论用例服务。
type Service struct {
	repo comment.Repository
}

// NewService 构造评论用例服务。
func NewService(repo comment.Repository) *Service {
	return &Service{repo: repo}
}

// ListForPost 文章评论列表。
func (s *Service) ListForPost(postID int64) ([]*comment.Comment, error) {
	return s.repo.ListByPost(postID)
}

// Create 发表评论。
func (s *Service) Create(userID, postID int64, content string) (*comment.Comment, error) {
	if msg := comment.Validate(content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	c := &comment.Comment{UserID: userID, PostID: postID, Content: content}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Delete 删除评论：本人或 author 角色可删，其余 ErrForbidden；返回被删评论的文章 ID（失效声明用）。
func (s *Service) Delete(userID int64, role string, commentID int64) (int64, error) {
	c, err := s.repo.Get(commentID)
	if err != nil {
		return 0, err
	}
	if c.UserID != userID && role != "author" {
		return 0, comment.ErrForbidden
	}
	if err := s.repo.Delete(commentID); err != nil {
		return 0, err
	}
	return c.PostID, nil
}

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
