package moderationapp

import (
	"context"
	"errors"
	"testing"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/build/domain/guestbook"
	"ven_hybird/build/domain/moderation"
)

// fakeModerator 预置 Verdict/error 序列（按调用顺序出结果；耗尽回退 pending）。
type fakeModerator struct {
	verdicts []moderation.Verdict
	errs     []error
	calls    int
}

func (f *fakeModerator) Review(_ context.Context, _ moderation.Request) (moderation.Verdict, error) {
	i := f.calls
	f.calls++
	if i < len(f.errs) && f.errs[i] != nil {
		return moderation.Verdict{}, f.errs[i]
	}
	if i < len(f.verdicts) {
		return f.verdicts[i], nil
	}
	return moderation.Verdict{Action: moderation.ActionPending}, nil
}

// fakeCommentRepo 内存实现 comment.Repository（与 commentapp 测试同款；ListPending 正序）。
type fakeCommentRepo struct {
	byID map[int64]*comment.Comment
	next int64
}

func newFakeCommentRepo() *fakeCommentRepo {
	return &fakeCommentRepo{byID: map[int64]*comment.Comment{}, next: 1}
}

func (f *fakeCommentRepo) add(c *comment.Comment) *comment.Comment {
	c.ID = f.next
	f.next++
	f.byID[c.ID] = c
	return c
}

func (f *fakeCommentRepo) ListByPost(postID int64) ([]*comment.Comment, error) { return nil, nil }
func (f *fakeCommentRepo) ListByMoment(momentID int64) ([]*comment.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) MomentCommentCounts() (map[int64]int, error) { return nil, nil }
func (f *fakeCommentRepo) ListAll(limit int) ([]*comment.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) ListPending() ([]*comment.Comment, error) {
	out := make([]*comment.Comment, 0, len(f.byID))
	for _, c := range f.byID {
		if c.Status == comment.StatusPending {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeCommentRepo) ListRejected() ([]*comment.Comment, error) { return nil, nil }
func (f *fakeCommentRepo) SetStatus(id int64, status string) error {
	c, ok := f.byID[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Status = status
	c.RejectedReason = ""
	return nil
}
func (f *fakeCommentRepo) SetRejected(id int64, reason string) error {
	c, ok := f.byID[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Status = comment.StatusRejected
	c.RejectedReason = reason
	return nil
}
func (f *fakeCommentRepo) Count() (int, error) { return len(f.byID), nil }
func (f *fakeCommentRepo) Get(id int64) (*comment.Comment, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, comment.ErrNotFound
	}
	return c, nil
}
func (f *fakeCommentRepo) Create(c *comment.Comment) error {
	f.add(c)
	return nil
}
func (f *fakeCommentRepo) Delete(id int64) error { return nil }

// fakeGuestbookRepo 内存实现 guestbook.Repository（与 guestbookapp 测试同款）。
type fakeGuestbookRepo struct {
	byID map[int64]*guestbook.Entry
	next int64
}

func newFakeGuestbookRepo() *fakeGuestbookRepo {
	return &fakeGuestbookRepo{byID: map[int64]*guestbook.Entry{}, next: 1}
}

func (f *fakeGuestbookRepo) add(e *guestbook.Entry) *guestbook.Entry {
	e.ID = f.next
	f.next++
	f.byID[e.ID] = e
	return e
}

func (f *fakeGuestbookRepo) List(limit int) ([]*guestbook.Entry, error) { return nil, nil }
func (f *fakeGuestbookRepo) ListAll(limit int) ([]*guestbook.Entry, error) {
	return nil, nil
}
func (f *fakeGuestbookRepo) ListPending() ([]*guestbook.Entry, error) {
	out := make([]*guestbook.Entry, 0, len(f.byID))
	for _, e := range f.byID {
		if e.Status == guestbook.StatusPending {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeGuestbookRepo) ListRejected() ([]*guestbook.Entry, error) { return nil, nil }
func (f *fakeGuestbookRepo) Get(id int64) (*guestbook.Entry, error) {
	e, ok := f.byID[id]
	if !ok {
		return nil, guestbook.ErrNotFound
	}
	return e, nil
}
func (f *fakeGuestbookRepo) Create(e *guestbook.Entry) error {
	f.add(e)
	return nil
}
func (f *fakeGuestbookRepo) Delete(id int64) error { return nil }
func (f *fakeGuestbookRepo) SetStatus(id int64, status string) error {
	e, ok := f.byID[id]
	if !ok {
		return guestbook.ErrNotFound
	}
	e.Status = status
	e.RejectedReason = ""
	return nil
}
func (f *fakeGuestbookRepo) SetRejected(id int64, reason string) error {
	e, ok := f.byID[id]
	if !ok {
		return guestbook.ErrNotFound
	}
	e.Status = guestbook.StatusRejected
	e.RejectedReason = reason
	return nil
}

func newService(cr *fakeCommentRepo, gr *fakeGuestbookRepo, m *fakeModerator) *Service {
	return NewService(
		commentapp.NewService(cr, nil),
		guestbookapp.NewService(gr, nil),
		m,
	)
}

func TestAutoReviewApprove(t *testing.T) {
	cr := newFakeCommentRepo()
	gr := newFakeGuestbookRepo()
	m := &fakeModerator{verdicts: []moderation.Verdict{
		{Action: moderation.ActionApprove},
		{Action: moderation.ActionApprove},
		{Action: moderation.ActionApprove},
	}}
	postC := cr.add(&comment.Comment{PostID: 3, UserID: 1, Username: "alice", Content: "写得好，学到了", Status: comment.StatusPending})
	momentC := cr.add(&comment.Comment{MomentID: 7, UserID: 2, Username: "bob", Content: "哈哈哈", Status: comment.StatusPending})
	e := gr.add(&guestbook.Entry{UserID: 1, Username: "carol", Content: "感谢分享！", Status: guestbook.StatusPending})

	svc := newService(cr, gr, m)
	result, err := svc.AutoReview(context.Background(), 20)
	if err != nil {
		t.Fatalf("AutoReview: %v", err)
	}
	if result.Processed != 3 || result.Approved != 3 || result.Rejected != 0 ||
		result.Uncertain != 0 || result.Failed != 0 {
		t.Fatalf("result = %+v, want processed=3 approved=3", result)
	}
	// 评论宿主信息携带（PostID/MomentID/宿主标题兜底）
	if got := result.ApprovedItems[0]; got.PostID != 3 || got.MomentID != 0 || got.HostTitle != "文章 #3" {
		t.Fatalf("post comment item = %+v, want PostID=3 HostTitle=文章 #3", got)
	}
	if got := result.ApprovedItems[1]; got.MomentID != 7 || got.HostTitle != "动态 #7" {
		t.Fatalf("moment comment item = %+v, want MomentID=7 HostTitle=动态 #7", got)
	}
	if got := result.ApprovedItems[2]; got.Kind != KindGuestbook || got.HostTitle != "作者主页" {
		t.Fatalf("guestbook item = %+v, want Kind=guestbook HostTitle=作者主页", got)
	}
	// 仓储状态落库
	if got, _ := cr.Get(postC.ID); got.Status != comment.StatusApproved {
		t.Fatalf("post comment status = %q, want approved", got.Status)
	}
	if got, _ := cr.Get(momentC.ID); got.Status != comment.StatusApproved {
		t.Fatalf("moment comment status = %q, want approved", got.Status)
	}
	if got, _ := gr.Get(e.ID); got.Status != guestbook.StatusApproved {
		t.Fatalf("guestbook status = %q, want approved", got.Status)
	}
}

func TestAutoReviewPostTitleUsed(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.add(&comment.Comment{PostID: 9, UserID: 1, Username: "alice", Content: "hi", PostTitle: "用 Go 写一个博客", Status: comment.StatusPending})
	m := &fakeModerator{verdicts: []moderation.Verdict{{Action: moderation.ActionApprove}}}
	result, err := newService(cr, newFakeGuestbookRepo(), m).AutoReview(context.Background(), 20)
	if err != nil {
		t.Fatalf("AutoReview: %v", err)
	}
	if got := result.ApprovedItems[0].HostTitle; got != "用 Go 写一个博客" {
		t.Fatalf("HostTitle = %q, want 用 Go 写一个博客", got)
	}
}

func TestAutoReviewReject(t *testing.T) {
	cr := newFakeCommentRepo()
	gr := newFakeGuestbookRepo()
	m := &fakeModerator{verdicts: []moderation.Verdict{
		{Action: moderation.ActionReject, Reason: "包含广告引流链接"},
		{Action: moderation.ActionReject, Reason: "无意义重复灌水"},
	}}
	c := cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "spammer", Content: "加微信领红包 https://evil.example", Status: comment.StatusPending})
	e := gr.add(&guestbook.Entry{UserID: 2, Username: "visitor", Content: "好 好 好 好 好", Status: guestbook.StatusPending})

	result, err := newService(cr, gr, m).AutoReview(context.Background(), 20)
	if err != nil {
		t.Fatalf("AutoReview: %v", err)
	}
	if result.Rejected != 2 || len(result.RejectedItems) != 2 {
		t.Fatalf("result = %+v, want rejected=2", result)
	}
	if got := result.RejectedItems[0]; got.ID != c.ID || got.Reason != "包含广告引流链接" || got.Username != "spammer" {
		t.Fatalf("rejected item = %+v", got)
	}
	got, _ := cr.Get(c.ID)
	if got.Status != comment.StatusRejected || got.RejectedReason != "包含广告引流链接" {
		t.Fatalf("comment after reject: status=%q reason=%q", got.Status, got.RejectedReason)
	}
	ge, _ := gr.Get(e.ID)
	if ge.Status != guestbook.StatusRejected || ge.RejectedReason != "无意义重复灌水" {
		t.Fatalf("guestbook after reject: status=%q reason=%q", ge.Status, ge.RejectedReason)
	}
}

func TestAutoReviewRejectClipsReason(t *testing.T) {
	cr := newFakeCommentRepo()
	long := "很长的原因"
	for len(long) <= 200 {
		long += "很长的原因"
	}
	m := &fakeModerator{verdicts: []moderation.Verdict{{Action: moderation.ActionReject, Reason: long}}}
	c := cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "x", Status: comment.StatusPending})
	result, err := newService(cr, newFakeGuestbookRepo(), m).AutoReview(context.Background(), 20)
	if err != nil {
		t.Fatalf("AutoReview: %v", err)
	}
	if result.Rejected != 1 {
		t.Fatalf("rejected = %d, want 1（超长 reason 应截断后落库而非校验失败）", result.Rejected)
	}
	got, _ := cr.Get(c.ID)
	if len(got.RejectedReason) > 200 {
		t.Fatalf("stored reason length = %d, want ≤200", len(got.RejectedReason))
	}
}

func TestAutoReviewPending(t *testing.T) {
	cr := newFakeCommentRepo()
	m := &fakeModerator{verdicts: []moderation.Verdict{{Action: moderation.ActionPending}}}
	c := cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "这个说法我觉得有问题", Status: comment.StatusPending})
	result, err := newService(cr, newFakeGuestbookRepo(), m).AutoReview(context.Background(), 20)
	if err != nil {
		t.Fatalf("AutoReview: %v", err)
	}
	if result.Uncertain != 1 || len(result.UncertainItems) != 1 {
		t.Fatalf("result = %+v, want uncertain=1", result)
	}
	// 不写库：仍 pending
	got, _ := cr.Get(c.ID)
	if got.Status != comment.StatusPending {
		t.Fatalf("status = %q, want pending（不确定内容必须留在待审队列）", got.Status)
	}
}

func TestAutoReviewErrorKeepsPendingAndRetries(t *testing.T) {
	cases := []struct {
		name      string
		errs      []error
		verdicts  []moderation.Verdict
		wantCalls int
		wantState string
		wantFail  int
		wantAppr  int
	}{
		{name: "retry once then still failed", errs: []error{errors.New("timeout"), errors.New("timeout")},
			wantCalls: 2, wantState: comment.StatusPending, wantFail: 1},
		{name: "retry succeeds on second call", errs: []error{errors.New("timeout")},
			verdicts:  []moderation.Verdict{{Action: moderation.ActionApprove}, {Action: moderation.ActionApprove}},
			wantCalls: 2, wantState: comment.StatusApproved, wantAppr: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cr := newFakeCommentRepo()
			m := &fakeModerator{verdicts: tc.verdicts, errs: tc.errs}
			c := cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "x", Status: comment.StatusPending})
			result, err := newService(cr, newFakeGuestbookRepo(), m).AutoReview(context.Background(), 20)
			if err != nil {
				t.Fatalf("AutoReview: %v", err)
			}
			if m.calls != tc.wantCalls {
				t.Fatalf("moderator calls = %d, want %d（error 应重试 1 次）", m.calls, tc.wantCalls)
			}
			if result.Failed != tc.wantFail || result.Approved != tc.wantAppr {
				t.Fatalf("result = %+v, want failed=%d approved=%d", result, tc.wantFail, tc.wantAppr)
			}
			got, _ := cr.Get(c.ID)
			if got.Status != tc.wantState {
				t.Fatalf("status = %q, want %q（error 路径绝不误杀/放行）", got.Status, tc.wantState)
			}
		})
	}
}

func TestAutoReviewLimitTruncation(t *testing.T) {
	cr := newFakeCommentRepo()
	verdicts := make([]moderation.Verdict, 20)
	for i := range verdicts {
		verdicts[i] = moderation.Verdict{Action: moderation.ActionApprove}
	}
	m := &fakeModerator{verdicts: verdicts}
	for i := 0; i < 25; i++ {
		cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "x", Status: comment.StatusPending})
	}
	result, err := newService(cr, newFakeGuestbookRepo(), m).AutoReview(context.Background(), 20)
	if err != nil {
		t.Fatalf("AutoReview: %v", err)
	}
	if result.Processed != 20 || result.Approved != 20 {
		t.Fatalf("processed = %d approved = %d, want 20/20（库存 25 条 limit 20）", result.Processed, result.Approved)
	}
}
