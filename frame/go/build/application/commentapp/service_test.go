package commentapp

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/comment"
)

// fakeRepo 内存实现 comment.Repository（测试用）。
type fakeRepo struct {
	byID map[int64]*comment.Comment
	next int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*comment.Comment{}, next: 1}
}

func (f *fakeRepo) add(c *comment.Comment) *comment.Comment {
	c.ID = f.next
	f.next++
	f.byID[c.ID] = c
	return c
}

func (f *fakeRepo) ListByPost(postID int64) ([]*comment.Comment, error) {
	out := make([]*comment.Comment, 0)
	for _, c := range f.byID {
		if c.PostID == postID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByMoment(momentID int64) ([]*comment.Comment, error) {
	out := make([]*comment.Comment, 0)
	for _, c := range f.byID {
		if c.MomentID == momentID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) MomentCommentCounts() (map[int64]int, error) {
	return map[int64]int{}, nil
}

func (f *fakeRepo) ListAll(limit int) ([]*comment.Comment, error) {
	out := make([]*comment.Comment, 0, len(f.byID))
	for _, c := range f.byID {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeRepo) ListPending() ([]*comment.Comment, error) {
	out := make([]*comment.Comment, 0)
	for _, c := range f.byID {
		if c.Status == comment.StatusPending {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListRejected() ([]*comment.Comment, error) {
	out := make([]*comment.Comment, 0)
	for _, c := range f.byID {
		if c.Status == comment.StatusRejected {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) SetStatus(id int64, status string) error {
	c, ok := f.byID[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Status = status
	c.RejectedReason = ""
	return nil
}

func (f *fakeRepo) SetRejected(id int64, reason string) error {
	c, ok := f.byID[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Status = comment.StatusRejected
	c.RejectedReason = reason
	return nil
}

func (f *fakeRepo) Count() (int, error) {
	return len(f.byID), nil
}

func (f *fakeRepo) Get(id int64) (*comment.Comment, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, comment.ErrNotFound
	}
	return c, nil
}

func (f *fakeRepo) Create(c *comment.Comment) error {
	f.add(c)
	return nil
}

func (f *fakeRepo) Delete(id int64) error {
	if _, ok := f.byID[id]; !ok {
		return comment.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func moderation(on bool) func() bool {
	return func() bool { return on }
}

func TestReject(t *testing.T) {
	t.Run("empty reason is validation error", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		_, err := svc.Reject(1, "  ")
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("want ValidationError, got %v", err)
		}
	})

	t.Run("reject any status records reason and returns target", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		approved := repo.add(&comment.Comment{PostID: 7, UserID: 1, Content: "hi", Status: comment.StatusApproved})
		target, err := svc.Reject(approved.ID, "spam")
		if err != nil {
			t.Fatalf("Reject: %v", err)
		}
		if target.PostID != 7 {
			t.Fatalf("target.PostID = %d, want 7", target.PostID)
		}
		got, _ := repo.Get(approved.ID)
		if got.Status != comment.StatusRejected || got.RejectedReason != "spam" {
			t.Fatalf("after reject: status=%q reason=%q", got.Status, got.RejectedReason)
		}
	})

	t.Run("not found passes through", func(t *testing.T) {
		svc := NewService(newFakeRepo(), nil)
		if _, err := svc.Reject(999, "spam"); !errors.Is(err, comment.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}

func TestRecover(t *testing.T) {
	t.Run("non-rejected returns ErrInvalidState", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		pending := repo.add(&comment.Comment{PostID: 7, UserID: 1, Content: "hi", Status: comment.StatusPending})
		if _, err := svc.Recover(pending.ID); !errors.Is(err, comment.ErrInvalidState) {
			t.Fatalf("want ErrInvalidState, got %v", err)
		}
	})

	t.Run("rejected becomes approved with reason cleared", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, nil)
		rejected := repo.add(&comment.Comment{MomentID: 3, UserID: 1, Content: "hi", Status: comment.StatusRejected, RejectedReason: "AI miss"})
		target, err := svc.Recover(rejected.ID)
		if err != nil {
			t.Fatalf("Recover: %v", err)
		}
		if target.MomentID != 3 {
			t.Fatalf("target.MomentID = %d, want 3", target.MomentID)
		}
		got, _ := repo.Get(rejected.ID)
		if got.Status != comment.StatusApproved || got.RejectedReason != "" {
			t.Fatalf("after recover: status=%q reason=%q", got.Status, got.RejectedReason)
		}
	})

	t.Run("not found passes through", func(t *testing.T) {
		svc := NewService(newFakeRepo(), nil)
		if _, err := svc.Recover(999); !errors.Is(err, comment.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}

func TestApproveOnRejectedIsRecover(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil)
	rejected := repo.add(&comment.Comment{PostID: 7, UserID: 1, Content: "hi", Status: comment.StatusRejected, RejectedReason: "spam"})
	if _, err := svc.Approve(rejected.ID); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	got, _ := repo.Get(rejected.ID)
	if got.Status != comment.StatusApproved || got.RejectedReason != "" {
		t.Fatalf("after approve on rejected: status=%q reason=%q", got.Status, got.RejectedReason)
	}
}

func TestCreateModeration(t *testing.T) {
	t.Run("moderation on starts pending", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, moderation(true))
		c, err := svc.Create(1, comment.Target{PostID: 7}, "hello", "")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.Status != comment.StatusPending {
			t.Fatalf("status = %q, want pending", c.Status)
		}
	})

	t.Run("moderation off starts approved", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, moderation(false))
		c, err := svc.Create(1, comment.Target{PostID: 7}, "hello", "")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.Status != comment.StatusApproved {
			t.Fatalf("status = %q, want approved", c.Status)
		}
	})
}

func TestListRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil)
	repo.add(&comment.Comment{PostID: 1, UserID: 1, Content: "a", Status: comment.StatusApproved})
	repo.add(&comment.Comment{PostID: 1, UserID: 1, Content: "b", Status: comment.StatusRejected, RejectedReason: "spam"})
	repo.add(&comment.Comment{PostID: 1, UserID: 1, Content: "c", Status: comment.StatusPending})
	list, err := svc.ListRejected()
	if err != nil {
		t.Fatalf("ListRejected: %v", err)
	}
	if len(list) != 1 || list[0].Content != "b" {
		t.Fatalf("want exactly the rejected comment, got %d entries", len(list))
	}
}

func TestGet(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil)
	c := repo.add(&comment.Comment{PostID: 1, UserID: 1, Content: "a", Status: comment.StatusApproved})
	got, err := svc.Get(c.ID)
	if err != nil || got.ID != c.ID {
		t.Fatalf("Get: got %v, err %v", got, err)
	}
	if _, err := svc.Get(999); !errors.Is(err, comment.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestDeletePermission(t *testing.T) {
	cases := []struct {
		name   string
		caller int64
		role   string
		wantOK bool
	}{
		{"owner", 1, "reader", true},
		{"author", 9, "author", true},
		{"other reader", 2, "reader", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			svc := NewService(repo, nil)
			c := repo.add(&comment.Comment{PostID: 1, UserID: 1, Content: "a", Status: comment.StatusApproved})
			_, err := svc.Delete(tt.caller, tt.role, c.ID)
			if tt.wantOK && err != nil {
				t.Fatalf("Delete: %v", err)
			}
			if !tt.wantOK && !errors.Is(err, comment.ErrForbidden) {
				t.Fatalf("want ErrForbidden, got %v", err)
			}
		})
	}
}
