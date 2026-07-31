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

// ListForMoment 动态评论列表。
func (s *Service) ListForMoment(momentID int64) ([]*comment.Comment, error) {
	return s.repo.ListByMoment(momentID)
}

// MomentCounts 动态评论数分组统计（/moments 页展示用）。
func (s *Service) MomentCounts() (map[int64]int, error) {
	return s.repo.MomentCommentCounts()
}

// ListAll 全站评论（后台管理用，含所属文章标题）。
func (s *Service) ListAll(limit int) ([]*comment.Comment, error) {
	return s.repo.ListAll(limit)
}

// Count 评论总数（后台统计）。
func (s *Service) Count() (int, error) {
	return s.repo.Count()
}

// Create 发表评论：target 指定宿主（文章或动态，恰一个非零）；replyTo 为回复目标用户名（可为空串）。
func (s *Service) Create(userID int64, target comment.Target, content, replyTo string) (*comment.Comment, error) {
	if !target.Valid() {
		return nil, &ValidationError{Message: "comment must target exactly one of post or moment"}
	}
	if msg := comment.Validate(content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	if len(replyTo) > 64 {
		return nil, &ValidationError{Message: "replyTo too long"}
	}
	c := &comment.Comment{
		UserID:   userID,
		PostID:   target.PostID,
		MomentID: target.MomentID,
		Content:  content,
		ReplyTo:  replyTo,
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Delete 删除评论：本人或 author 角色可删，其余 ErrForbidden；返回宿主（失效声明用）。
func (s *Service) Delete(userID int64, role string, commentID int64) (comment.Target, error) {
	c, err := s.repo.Get(commentID)
	if err != nil {
		return comment.Target{}, err
	}
	if c.UserID != userID && role != "author" {
		return comment.Target{}, comment.ErrForbidden
	}
	if err := s.repo.Delete(commentID); err != nil {
		return comment.Target{}, err
	}
	return comment.Target{PostID: c.PostID, MomentID: c.MomentID}, nil
}

// ValidationError 用例入参校验失败（映射为 400）。
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
