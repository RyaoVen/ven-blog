package guestbookapp

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/guestbook"
)

// fakeRepo 内存实现 guestbook.Repository（测试用）。
// List 语义与真实仓储一致：仅返回 approved（公开列表）。
// reviewed 模拟 ai_reviewed_at：MarkAIReviewed 打标，ListUnreviewedPending 排除已打标。
type fakeRepo struct {
	byID     map[int64]*guestbook.Entry
	reviewed map[int64]bool
	next     int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*guestbook.Entry{}, reviewed: map[int64]bool{}, next: 1}
}

func (f *fakeRepo) add(e *guestbook.Entry) *guestbook.Entry {
	e.ID = f.next
	f.next++
	f.byID[e.ID] = e
	return e
}

func (f *fakeRepo) List(limit int) ([]*guestbook.Entry, error) {
	out := make([]*guestbook.Entry, 0)
	for _, e := range f.byID {
		if e.Status == guestbook.StatusApproved {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListAll(limit int) ([]*guestbook.Entry, error) {
	out := make([]*guestbook.Entry, 0, len(f.byID))
	for _, e := range f.byID {
		out = append(out, e)
	}
	return out, nil
}

func (f *fakeRepo) ListPending() ([]*guestbook.Entry, error) {
	out := make([]*guestbook.Entry, 0)
	for _, e := range f.byID {
		if e.Status == guestbook.StatusPending {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListUnreviewedPending() ([]*guestbook.Entry, error) {
	out := make([]*guestbook.Entry, 0)
	for _, e := range f.byID {
		if e.Status == guestbook.StatusPending && !f.reviewed[e.ID] {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRepo) MarkAIReviewed(id int64) error {
	// 与真仓储一致：仅 pending 行打标，幂等，不存在也不报错。
	if e, ok := f.byID[id]; ok && e.Status == guestbook.StatusPending {
		f.reviewed[id] = true
	}
	return nil
}

func (f *fakeRepo) ListRejected() ([]*guestbook.Entry, error) {
	out := make([]*guestbook.Entry, 0)
	for _, e := range f.byID {
		if e.Status == guestbook.StatusRejected {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRepo) Get(id int64) (*guestbook.Entry, error) {
	e, ok := f.byID[id]
	if !ok {
		return nil, guestbook.ErrNotFound
	}
	return e, nil
}

func (f *fakeRepo) Create(e *guestbook.Entry) error {
	f.add(e)
	return nil
}

func (f *fakeRepo) Delete(id int64) error {
	if _, ok := f.byID[id]; !ok {
		return guestbook.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeRepo) SetStatus(id int64, status string) error {
	e, ok := f.byID[id]
	if !ok {
		return guestbook.ErrNotFound
	}
	e.Status = status
	e.RejectedReason = ""
	return nil
}

func (f *fakeRepo) SetRejected(id int64, reason string) error {
	e, ok := f.byID[id]
	if !ok {
		return guestbook.ErrNotFound
	}
	e.Status = guestbook.StatusRejected
	e.RejectedReason = reason
	return nil
}

func moderation(on bool) func() bool {
	return func() bool { return on }
}

func TestCreateModeration(t *testing.T) {
	t.Run("moderation on starts pending", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, moderation(true))
		e, err := svc.Create(1, "hello")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if e.Status != guestbook.StatusPending {
			t.Fatalf("status = %q, want pending", e.Status)
		}
	})

	t.Run("moderation off starts approved", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, moderation(false))
		e, err := svc.Create(1, "hello")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if e.Status != guestbook.StatusApproved {
			t.Fatalf("status = %q, want approved", e.Status)
		}
	})
}

func TestApprove(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, moderation(true))
	pending := repo.add(&guestbook.Entry{UserID: 1, Content: "hi", Status: guestbook.StatusPending})
	if err := svc.Approve(pending.ID); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	got, _ := repo.Get(pending.ID)
	if got.Status != guestbook.StatusApproved {
		t.Fatalf("status = %q, want approved", got.Status)
	}
}

func TestReject(t *testing.T) {
	t.Run("empty reason is validation error", func(t *testing.T) {
		svc := NewService(newFakeRepo(), nil)
		err := svc.Reject(1, "")
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("want ValidationError, got %v", err)
		}
	})

	t.Run("reject records reason", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		approved := repo.add(&guestbook.Entry{UserID: 1, Content: "hi", Status: guestbook.StatusApproved})
		if err := svc.Reject(approved.ID, "spam"); err != nil {
			t.Fatalf("Reject: %v", err)
		}
		got, _ := repo.Get(approved.ID)
		if got.Status != guestbook.StatusRejected || got.RejectedReason != "spam" {
			t.Fatalf("after reject: status=%q reason=%q", got.Status, got.RejectedReason)
		}
	})

	t.Run("not found passes through", func(t *testing.T) {
		svc := NewService(newFakeRepo(), nil)
		if err := svc.Reject(999, "spam"); !errors.Is(err, guestbook.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}

func TestRecover(t *testing.T) {
	t.Run("non-rejected returns ErrInvalidState", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		pending := repo.add(&guestbook.Entry{UserID: 1, Content: "hi", Status: guestbook.StatusPending})
		if err := svc.Recover(pending.ID); !errors.Is(err, guestbook.ErrInvalidState) {
			t.Fatalf("want ErrInvalidState, got %v", err)
		}
	})

	t.Run("rejected becomes approved with reason cleared", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		rejected := repo.add(&guestbook.Entry{UserID: 1, Content: "hi", Status: guestbook.StatusRejected, RejectedReason: "AI miss"})
		if err := svc.Recover(rejected.ID); err != nil {
			t.Fatalf("Recover: %v", err)
		}
		got, _ := repo.Get(rejected.ID)
		if got.Status != guestbook.StatusApproved || got.RejectedReason != "" {
			t.Fatalf("after recover: status=%q reason=%q", got.Status, got.RejectedReason)
		}
	})
}

func TestListPublicOnlyApproved(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil)
	repo.add(&guestbook.Entry{UserID: 1, Content: "a", Status: guestbook.StatusApproved})
	repo.add(&guestbook.Entry{UserID: 1, Content: "b", Status: guestbook.StatusPending})
	repo.add(&guestbook.Entry{UserID: 1, Content: "c", Status: guestbook.StatusRejected, RejectedReason: "spam"})
	list, err := svc.List(50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Content != "a" {
		t.Fatalf("public List must only contain approved, got %d entries", len(list))
	}
}

func TestListPendingAndRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil)
	repo.add(&guestbook.Entry{UserID: 1, Content: "a", Status: guestbook.StatusApproved})
	repo.add(&guestbook.Entry{UserID: 1, Content: "b", Status: guestbook.StatusPending})
	repo.add(&guestbook.Entry{UserID: 1, Content: "c", Status: guestbook.StatusRejected, RejectedReason: "spam"})
	pending, err := svc.ListPending()
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 1 || pending[0].Content != "b" {
		t.Fatalf("pending = %d entries, want 1", len(pending))
	}
	rejected, err := svc.ListRejected()
	if err != nil {
		t.Fatalf("ListRejected: %v", err)
	}
	if len(rejected) != 1 || rejected[0].Content != "c" {
		t.Fatalf("rejected = %d entries, want 1", len(rejected))
	}
	all, err := svc.ListAll(200)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListAll = %d entries, want 3", len(all))
	}
}
