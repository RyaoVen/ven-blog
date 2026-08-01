// Package commentapp 评论用例服务。
package commentapp

import (
	"ven_hybird/build/domain/comment"
)

// Service 评论用例服务。
type Service struct {
	repo       comment.Repository
	moderation func() bool // 评论审核开关（true 时新评论待审核）
}

// NewService 构造评论用例服务；moderation 为审核开关解析函数（组装根注入，随设置实时生效）。
func NewService(repo comment.Repository, moderation func() bool) *Service {
	return &Service{repo: repo, moderation: moderation}
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

// Approve 审核通过评论；返回宿主（失效声明用）。
func (s *Service) Approve(commentID int64) (comment.Target, error) {
	c, err := s.repo.Get(commentID)
	if err != nil {
		return comment.Target{}, err
	}
	if err := s.repo.SetStatus(commentID, comment.StatusApproved); err != nil {
		return comment.Target{}, err
	}
	return comment.Target{PostID: c.PostID, MomentID: c.MomentID}, nil
}

// ListPending 待审核评论（后台管理用）。
func (s *Service) ListPending() ([]*comment.Comment, error) {
	return s.repo.ListPending()
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
	status := comment.StatusApproved
	if s.moderation != nil && s.moderation() {
		status = comment.StatusPending
	}
	c := &comment.Comment{
		UserID:   userID,
		PostID:   target.PostID,
		MomentID: target.MomentID,
		Content:  content,
		ReplyTo:  replyTo,
		Status:   status,
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
