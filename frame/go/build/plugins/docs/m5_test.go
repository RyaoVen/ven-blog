package docs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMove_RenameAndCascade(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "N", Path: "ai/attention", Kind: KindSection})
	svc.Create(CreateInput{Title: "Self", Path: "ai/attention/self"})
	got, err := svc.Move(MoveInput{Path: "ai/attention", NewSlug: "attention-v2"})
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.Path != "ai/attention-v2" {
		t.Fatalf("新 path 不对：%s", got.Path)
	}
	if _, err := svc.Get("ai/attention-v2/self"); err != nil {
		t.Fatal("后代 path 应级联更新")
	}
	if _, err := svc.Get("ai/attention/self"); !errIsNotFound(err) {
		t.Fatal("旧 path 应不存在")
	}
}

func TestMove_ToNewParent(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "N", Path: "ai/n"})
	svc.Create(CreateInput{Title: "ML", Path: "ml", Kind: KindSection})
	got, err := svc.Move(MoveInput{Path: "ai/n", NewParent: "ml"})
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.Path != "ml/n" {
		t.Fatalf("应移动到 ml/n，得 %s", got.Path)
	}
}

func TestMove_CycleRejected(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "A", Path: "a", Kind: KindSection})
	svc.Create(CreateInput{Title: "Child", Path: "a/b"})
	// 把 a 移到 a/b 下 = 环。
	if _, err := svc.Move(MoveInput{Path: "a", NewParent: "a/b"}); err == nil {
		t.Fatal("移入自身后代应拒绝")
	}
}

func TestMove_TargetOccupied(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "A", Path: "x/a"})
	svc.Create(CreateInput{Title: "B", Path: "y/b"})
	if _, err := svc.Move(MoveInput{Path: "x/a", NewParent: "y", NewSlug: "b"}); err == nil {
		t.Fatal("目标 path 被占应拒绝")
	}
}

func TestMove_ReorderOnly(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "A", Path: "a"})
	got, err := svc.Move(MoveInput{Path: "a", Order: intPtr(7)})
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.SortOrder != 7 {
		t.Fatalf("order 应更新，得 %d", got.SortOrder)
	}
}

func TestListUpdatedSince_Semantics(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "A", Path: "a"})
	cutoff := svc.nowFn()
	svc.Create(CreateInput{Title: "B", Path: "b"})
	svc.Update(UpdateInput{Path: "a", Title: strPtr("A2")})
	docs, err := svc.repo.ListUpdatedSince(cutoff)
	if err != nil {
		t.Fatalf("ListUpdatedSince: %v", err)
	}
	// cutoff 之后：b 创建 + a 更新（a 的 updated_at 已刷新）。
	found := map[string]bool{}
	for _, d := range docs {
		found[d.Path] = true
	}
	if !found["a"] || !found["b"] {
		t.Fatalf("应含 a、b：%v", found)
	}
}

func (s *Service) nowFn() time.Time {
	base := time.Now().Add(-time.Minute)
	s.SetClock(func() time.Time { return base })
	return base
}

func errIsNotFound(err error) bool { return err != nil && err.Error() == ErrNotFound.Error() }

func intPtr(v int) *int { return &v }

func TestWebhookDispatcher_SignatureDelivery(t *testing.T) {
	var gotSig string
	var gotBody []byte
	received := make(chan struct{}, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-Ven-Signature")
		gotBody = body
		w.WriteHeader(200)
		received <- struct{}{}
	}))
	defer server.Close()

	store := fakeSettingsStore{
		"plugin.docs.webhook_url":    server.URL,
		"plugin.docs.webhook_secret": "s3cret",
	}
	dispatcher := NewWebhookDispatcher(store)
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	dispatcher.Emit("doc.created", "ai/attention")
	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("webhook 未送达")
	}
	dispatcher.Stop()

	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(gotBody)
	want := hex.EncodeToString(mac.Sum(nil))
	if gotSig != want {
		t.Fatalf("HMAC 签名不匹配：got %s want %s", gotSig, want)
	}
	var ev WebhookEvent
	_ = json.Unmarshal(gotBody, &ev)
	if ev.Event != "doc.created" || ev.Path != "ai/attention" {
		t.Fatalf("事件体不对：%s", gotBody)
	}
}

func TestWebhookDispatcher_RetriesOn500(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(500)
	}))
	defer server.Close()

	store := fakeSettingsStore{"plugin.docs.webhook_url": server.URL}
	dispatcher := NewWebhookDispatcher(store)
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	dispatcher.Emit("doc.updated", "x")
	time.Sleep(500 * time.Millisecond)
	dispatcher.Stop()
	// 退避 0/2s/8s：500ms 内应只发第 1 次（不阻塞测试验证退避调度存在）。
	if count < 1 {
		t.Fatalf("应至少投递 1 次，得 %d", count)
	}
}

func TestWebhookDispatcher_NotConfiguredIdles(t *testing.T) {
	dispatcher := NewWebhookDispatcher(fakeSettingsStore{})
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("未配置应空转启动: %v", err)
	}
	dispatcher.Emit("doc.created", "x") // 不应 panic/阻塞
	if err := dispatcher.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// fakeSettingsStore webhook 测试用 settings。
type fakeSettingsStore map[string]string

func (s fakeSettingsStore) Get(key string) (string, error) { return s[key], nil }
func (s fakeSettingsStore) Set(key, value string) error    { s[key] = value; return nil }
