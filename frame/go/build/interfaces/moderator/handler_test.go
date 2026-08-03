package moderator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/application/moderationapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/build/domain/guestbook"
	"ven_hybird/build/domain/moderation"
)

// fakeMailer 记录发送调用，可注入发送失败。
type fakeMailer struct {
	to      string
	subject string
	text    string
	sent    bool
	err     error
}

func (f *fakeMailer) Send(to, subject, text string) error {
	f.to, f.subject, f.text = to, subject, text
	f.sent = true
	return f.err
}

func (f *fakeMailer) SendHTML(to, subject, html string) error {
	f.to, f.subject, f.text = to, subject, html
	f.sent = true
	return f.err
}

// fakeModerator 预置判定序列；耗尽回退 pending。
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

// fakeSettingsRepo 内存设置仓储。
type fakeSettingsRepo struct {
	kv map[string]string
}

func (f *fakeSettingsRepo) Get(key string) (string, error) { return f.kv[key], nil }
func (f *fakeSettingsRepo) Set(key, value string) error {
	f.kv[key] = value
	return nil
}

// fakeCommentRepo 评论仓储（带待审队列错误注入；reviewed 模拟 ai_reviewed_at）。
type fakeCommentRepo struct {
	byID     map[int64]*comment.Comment
	reviewed map[int64]bool
	next     int64
	listErr  error
}

func newFakeCommentRepo() *fakeCommentRepo {
	return &fakeCommentRepo{byID: map[int64]*comment.Comment{}, reviewed: map[int64]bool{}, next: 1}
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
func (f *fakeCommentRepo) ListAll(limit int) ([]*comment.Comment, error) { return nil, nil }
func (f *fakeCommentRepo) ListPending() ([]*comment.Comment, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*comment.Comment, 0, len(f.byID))
	for _, c := range f.byID {
		if c.Status == comment.StatusPending {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeCommentRepo) ListUnreviewedPending() ([]*comment.Comment, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*comment.Comment, 0, len(f.byID))
	for _, c := range f.byID {
		if c.Status == comment.StatusPending && !f.reviewed[c.ID] {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeCommentRepo) MarkAIReviewed(id int64) error {
	// 与真仓储一致：仅 pending 行打标，幂等，不存在也不报错。
	if c, ok := f.byID[id]; ok && c.Status == comment.StatusPending {
		f.reviewed[id] = true
	}
	return nil
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

// fakeGuestbookRepo 留言板仓储（带待审队列错误注入；reviewed 模拟 ai_reviewed_at）。
type fakeGuestbookRepo struct {
	byID     map[int64]*guestbook.Entry
	reviewed map[int64]bool
	next     int64
	listErr  error
}

func newFakeGuestbookRepo() *fakeGuestbookRepo {
	return &fakeGuestbookRepo{byID: map[int64]*guestbook.Entry{}, reviewed: map[int64]bool{}, next: 1}
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
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*guestbook.Entry, 0, len(f.byID))
	for _, e := range f.byID {
		if e.Status == guestbook.StatusPending {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeGuestbookRepo) ListUnreviewedPending() ([]*guestbook.Entry, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*guestbook.Entry, 0, len(f.byID))
	for _, e := range f.byID {
		if e.Status == guestbook.StatusPending && !f.reviewed[e.ID] {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeGuestbookRepo) MarkAIReviewed(id int64) error {
	// 与真仓储一致：仅 pending 行打标，幂等，不存在也不报错。
	if e, ok := f.byID[id]; ok && e.Status == guestbook.StatusPending {
		f.reviewed[id] = true
	}
	return nil
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

// newHandler 组装真实 AutoReview + fake 依赖（结果完全由 fake Moderator 控制）。
func newHandler(cr *fakeCommentRepo, gr *fakeGuestbookRepo, m *fakeModerator, mail *fakeMailer, enabled func() bool) (*Handler, *fakeSettingsRepo) {
	settings := &fakeSettingsRepo{kv: map[string]string{"author_email": "author@example.com"}}
	svc := moderationapp.NewService(
		commentapp.NewService(cr, nil),
		guestbookapp.NewService(gr, nil),
		m,
	)
	h := NewHandler(svc, settingsapp.NewService(settings), mail, &fakeInvalidator{},
		authorNameFn, "https://blog.example.com", Options{Batch: 20, Enabled: enabled})
	return h, settings
}

func TestRunOnceSendsSummaryMail(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "spammer", Content: "加微信领红包", Status: comment.StatusPending})
	cr.add(&comment.Comment{PostID: 1, UserID: 2, Username: "reader", Content: "有疑问……", Status: comment.StatusPending})
	cr.add(&comment.Comment{PostID: 1, UserID: 3, Username: "user2", Content: "hi", Status: comment.StatusPending})
	gr := newFakeGuestbookRepo()
	gr.add(&guestbook.Entry{UserID: 4, Username: "visitor", Content: "写得好", Status: guestbook.StatusPending})
	m := &fakeModerator{
		verdicts: []moderation.Verdict{
			{Action: moderation.ActionReject, Reason: "包含广告引流链接"}, // 评论1
			{Action: moderation.ActionPending},                          // 评论2
			{Action: moderation.ActionApprove},                          // 评论3
		},
		errs: []error{nil, nil, nil, errors.New("timeout"), errors.New("timeout")}, // 留言板两次都失败 → 保持 pending 记 failed
	}
	mail := &fakeMailer{}
	h, _ := newHandler(cr, gr, m, mail, func() bool { return true })

	h.RunOnce(context.Background())

	if !mail.sent {
		t.Fatal("summary mail should be sent when rejected/uncertain exist")
	}
	if mail.to != "author@example.com" {
		t.Fatalf("to = %q, want author@example.com", mail.to)
	}
	if !strings.HasPrefix(mail.subject, "ven-blog 内容审核摘要（") {
		t.Fatalf("subject = %q", mail.subject)
	}
	for _, want := range []string{"自动驳回", "包含广告引流链接", "需人工复核", "判定失败",
		`href="https://blog.example.com/admin/comments"`, "前往管理面板"} {
		if !strings.Contains(mail.text, want) {
			t.Fatalf("mail text missing %q:\n%s", want, mail.text)
		}
	}
}

func TestRunOnceAllApprovedNoMail(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "正常交流", Status: comment.StatusPending})
	m := &fakeModerator{verdicts: []moderation.Verdict{{Action: moderation.ActionApprove}}}
	mail := &fakeMailer{}
	h, _ := newHandler(cr, newFakeGuestbookRepo(), m, mail, func() bool { return true })

	h.RunOnce(context.Background())

	if mail.sent {
		t.Fatal("all-approved round should not send mail")
	}
	if m.calls != 1 {
		t.Fatalf("moderator calls = %d, want 1", m.calls)
	}
}

func TestRunOnceNoAuthorEmailSkipsMail(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "垃圾广告", Status: comment.StatusPending})
	m := &fakeModerator{verdicts: []moderation.Verdict{{Action: moderation.ActionReject, Reason: "spam"}}}
	mail := &fakeMailer{}
	h, settings := newHandler(cr, newFakeGuestbookRepo(), m, mail, func() bool { return true })
	settings.kv["author_email"] = ""

	h.RunOnce(context.Background())

	if mail.sent {
		t.Fatal("no mail when author email empty")
	}
}

func TestRunOnceDisabledDoesNothing(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "x", Status: comment.StatusPending})
	m := &fakeModerator{}
	mail := &fakeMailer{}
	h, _ := newHandler(cr, newFakeGuestbookRepo(), m, mail, func() bool { return false })

	h.RunOnce(context.Background())

	if m.calls != 0 {
		t.Fatalf("moderator calls = %d, want 0（开关关闭不执行 AutoReview）", m.calls)
	}
	if mail.sent {
		t.Fatal("no mail when disabled")
	}
}

func TestRunOnceSendFailureOnlyLogs(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.add(&comment.Comment{PostID: 1, UserID: 1, Username: "u", Content: "垃圾广告", Status: comment.StatusPending})
	m := &fakeModerator{verdicts: []moderation.Verdict{{Action: moderation.ActionReject, Reason: "spam"}}}
	mail := &fakeMailer{err: errors.New("smtp down")}
	h, _ := newHandler(cr, newFakeGuestbookRepo(), m, mail, func() bool { return true })

	h.RunOnce(context.Background()) // 不应 panic；Send 失败只记日志
}

func TestRunOnceAutoReviewErrorOnlyLogs(t *testing.T) {
	cr := newFakeCommentRepo()
	cr.listErr = errors.New("db down")
	m := &fakeModerator{}
	mail := &fakeMailer{}
	h, _ := newHandler(cr, newFakeGuestbookRepo(), m, mail, func() bool { return true })

	h.RunOnce(context.Background()) // 不应 panic；本轮整体失败只记日志

	if m.calls != 0 || mail.sent {
		t.Fatal("failed round should not review or send mail")
	}
}

func TestIntervalFromEnv(t *testing.T) {
	t.Setenv("BLOG_MODERATOR_INTERVAL", "1m")
	if got := IntervalFromEnv(); got != time.Minute {
		t.Fatalf("1m → %v", got)
	}
	t.Setenv("BLOG_MODERATOR_INTERVAL", "bad")
	if got := IntervalFromEnv(); got != 5*time.Minute {
		t.Fatalf("bad → %v, want 5m", got)
	}
	t.Setenv("BLOG_MODERATOR_INTERVAL", "0s")
	if got := IntervalFromEnv(); got != 5*time.Minute {
		t.Fatalf("0s → %v, want 5m", got)
	}
	t.Setenv("BLOG_MODERATOR_INTERVAL", "")
	if got := IntervalFromEnv(); got != 5*time.Minute {
		t.Fatalf("empty → %v, want 5m", got)
	}
}

func TestBatchFromEnv(t *testing.T) {
	t.Setenv("BLOG_MODERATOR_BATCH", "5")
	if got := BatchFromEnv(); got != 5 {
		t.Fatalf("5 → %d", got)
	}
	t.Setenv("BLOG_MODERATOR_BATCH", "0")
	if got := BatchFromEnv(); got != 20 {
		t.Fatalf("0 → %d, want 20", got)
	}
	t.Setenv("BLOG_MODERATOR_BATCH", "-3")
	if got := BatchFromEnv(); got != 20 {
		t.Fatalf("-3 → %d, want 20", got)
	}
	t.Setenv("BLOG_MODERATOR_BATCH", "bad")
	if got := BatchFromEnv(); got != 20 {
		t.Fatalf("bad → %d, want 20", got)
	}
	t.Setenv("BLOG_MODERATOR_BATCH", "")
	if got := BatchFromEnv(); got != 20 {
		t.Fatalf("empty → %d, want 20", got)
	}
}
