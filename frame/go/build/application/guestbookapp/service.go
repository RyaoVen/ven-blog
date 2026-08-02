// Package guestbookapp 留言板用例服务。
package guestbookapp

import (
	"ven_hybird/build/domain/guestbook"
)

// Service 留言板用例服务。
type Service struct {
	repo       guestbook.Repository
	moderation func() bool // 审核开关（true 时新留言待审核，与评论共用同一开关）
}

// NewService 构造留言板用例服务；moderation 为审核开关解析函数（组装根注入，随设置实时生效）。
func NewService(repo guestbook.Repository, moderation func() bool) *Service {
	return &Service{repo: repo, moderation: moderation}
}

// List 公开留言列表（最近 limit 条，仅 approved）。
func (s *Service) List(limit int) ([]*guestbook.Entry, error) {
	return s.repo.List(limit)
}

// Create 发表留言：审核开启时初始状态 pending，否则直接 approved。
func (s *Service) Create(userID int64, content string) (*guestbook.Entry, error) {
	if msg := guestbook.Validate(content); msg != "" {
		return nil, &ValidationError{Message: msg}
	}
	status := guestbook.StatusApproved
	if s.moderation != nil && s.moderation() {
		status = guestbook.StatusPending
	}
	e := &guestbook.Entry{UserID: userID, Content: content, Status: status}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Approve 审核通过留言（对 rejected 调用等价 recover，reason 随 SetStatus 清空）。
func (s *Service) Approve(id int64) error {
	if _, err := s.repo.Get(id); err != nil {
		return err
	}
	return s.repo.SetStatus(id, guestbook.StatusApproved)
}

// Reject 驳回留言：reason 必填且 ≤200，任意状态可驳回。
func (s *Service) Reject(id int64, reason string) error {
	if msg := guestbook.ValidateRejectedReason(reason); msg != "" {
		return &ValidationError{Message: msg}
	}
	if _, err := s.repo.Get(id); err != nil {
		return err
	}
	return s.repo.SetRejected(id, reason)
}

// Recover 恢复被驳回的留言：仅 rejected → approved（reason 随 SetStatus 清空）；其余状态返回 ErrInvalidState。
func (s *Service) Recover(id int64) error {
	e, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	if e.Status != guestbook.StatusRejected {
		return guestbook.ErrInvalidState
	}
	return s.repo.SetStatus(id, guestbook.StatusApproved)
}

// ListPending 待审核留言（后台管理用）。
func (s *Service) ListPending() ([]*guestbook.Entry, error) {
	return s.repo.ListPending()
}

// ListRejected 被驳回留言（后台管理用）。
func (s *Service) ListRejected() ([]*guestbook.Entry, error) {
	return s.repo.ListRejected()
}

// ListAll 全量留言（后台管理用，含全部状态）。
func (s *Service) ListAll(limit int) ([]*guestbook.Entry, error) {
	return s.repo.ListAll(limit)
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
