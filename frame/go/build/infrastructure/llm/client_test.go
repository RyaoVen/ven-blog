package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ven_hybird/build/domain/moderation"
)

// recordServer 假 OpenAI 兼容端点：记录最近一次请求并返回预置响应。
func recordServer(t *testing.T, status int, content string) (*httptest.Server, *chatRequest, *http.Header) {
	t.Helper()
	var reqBody chatRequest
	var reqHeader http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqHeader = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &reqBody)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": content}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv, &reqBody, &reqHeader
}

func newTestClient(baseURL string) *Client {
	return &Client{
		cfg:  Config{BaseURL: baseURL, APIKey: "test-key", Model: "test-model"},
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func TestReviewRequestShape(t *testing.T) {
	srv, req, hdr := recordServer(t, http.StatusOK, `{"verdict":"approve","reason":""}`)
	c := newTestClient(srv.URL)

	verdict, err := c.Review(context.Background(), moderation.Request{
		Host:      moderation.HostComment,
		HostTitle: "用 Go 写一个博客",
		Content:   "写得好",
		ReplyTo:   "alice",
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if verdict.Action != moderation.ActionApprove {
		t.Fatalf("action = %q, want approve", verdict.Action)
	}
	// 请求体：model / temperature=0 / JSON 输出 / 单条 user 消息
	if req.Model != "test-model" {
		t.Fatalf("model = %q, want test-model", req.Model)
	}
	if req.Temperature != 0 {
		t.Fatalf("temperature = %d, want 0", req.Temperature)
	}
	if req.ResponseFormat["type"] != "json_object" {
		t.Fatalf("response_format = %v, want json_object", req.ResponseFormat)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v, want single user message", req.Messages)
	}
	msg := req.Messages[0].Content
	for _, want := range []string{"审核规则", "用 Go 写一个博客", "@alice", "写得好", "approve", "reject", "pending"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
	// 请求头
	if got := hdr.Get("Authorization"); got != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want Bearer test-key", got)
	}
	if got := hdr.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestReviewGuestbookPromptKind(t *testing.T) {
	srv, req, _ := recordServer(t, http.StatusOK, `{"verdict":"pending","reason":""}`)
	c := newTestClient(srv.URL)
	if _, err := c.Review(context.Background(), moderation.Request{
		Host:      moderation.HostGuestbook,
		HostTitle: "作者主页",
		Content:   "你好呀",
	}); err != nil {
		t.Fatalf("Review: %v", err)
	}
	msg := req.Messages[0].Content
	if !strings.Contains(msg, "guestbook（留言板留言）") || !strings.Contains(msg, "（无）") {
		t.Fatalf("prompt should use guestbook kind and 无 replyTo: %q", msg)
	}
}

func TestReviewRejectWithReason(t *testing.T) {
	srv, _, _ := recordServer(t, http.StatusOK, `{"verdict":"reject","reason":"包含广告引流链接"}`)
	c := newTestClient(srv.URL)
	v, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if v.Action != moderation.ActionReject || v.Reason != "包含广告引流链接" {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestReviewFencedJSONStillParses(t *testing.T) {
	// 模型偶尔用 Markdown 围栏包裹 JSON，宽松解析应剥掉围栏
	srv, _, _ := recordServer(t, http.StatusOK, "```json\n{\"verdict\":\"approve\",\"reason\":\"\"}\n```")
	c := newTestClient(srv.URL)
	v, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if v.Action != moderation.ActionApprove {
		t.Fatalf("action = %q, want approve", v.Action)
	}
}

func TestReviewHTTPError(t *testing.T) {
	srv, _, _ := recordServer(t, http.StatusInternalServerError, "")
	c := newTestClient(srv.URL)
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error on non-2xx, got nil")
	}
}

func TestReviewTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	c := &Client{
		cfg:  Config{BaseURL: srv.URL, APIKey: "k", Model: "m"},
		http: &http.Client{Timeout: 50 * time.Millisecond},
	}
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error on timeout, got nil")
	}
}

func TestReviewContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.Review(ctx, moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error on ctx cancel, got nil")
	}
}

func TestReviewInvalidVerdictIsError(t *testing.T) {
	// 非法值（如 "approved"）必须视为无法判定 → error，绝不按非法值放行/驳回
	srv, _, _ := recordServer(t, http.StatusOK, `{"verdict":"approved","reason":""}`)
	c := newTestClient(srv.URL)
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error on invalid verdict, got nil")
	}
}

func TestReviewRejectWithoutReasonIsError(t *testing.T) {
	srv, _, _ := recordServer(t, http.StatusOK, `{"verdict":"reject","reason":""}`)
	c := newTestClient(srv.URL)
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error on reject without reason, got nil")
	}
}

func TestReviewMissingContentIsError(t *testing.T) {
	srv, _, _ := recordServer(t, http.StatusOK, `{"choices":[]}`)
	c := newTestClient(srv.URL)
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error when choices empty, got nil")
	}
}

func TestReviewNetworkError(t *testing.T) {
	c := newTestClient("http://127.0.0.1:1/v1") // 不可达
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err == nil {
		t.Fatal("want error on network failure, got nil")
	}
}

func TestNewClientEnv(t *testing.T) {
	t.Setenv("BLOG_LLM_API_KEY", "")
	if _, err := NewClient(); err == nil {
		t.Fatal("NewClient without API key should error")
	}
	t.Setenv("BLOG_LLM_API_KEY", "k")
	t.Setenv("BLOG_LLM_BASE_URL", "")
	t.Setenv("BLOG_LLM_MODEL", "")
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.cfg.BaseURL != "https://api.deepseek.com/v1" || c.cfg.Model != "deepseek-chat" {
		t.Fatalf("defaults: BaseURL=%q Model=%q", c.cfg.BaseURL, c.cfg.Model)
	}
	t.Setenv("BLOG_LLM_BASE_URL", "http://localhost:9999/v1")
	t.Setenv("BLOG_LLM_MODEL", "gpt-test")
	c, err = NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.cfg.BaseURL != "http://localhost:9999/v1" || c.cfg.Model != "gpt-test" {
		t.Fatalf("env override: BaseURL=%q Model=%q", c.cfg.BaseURL, c.cfg.Model)
	}
}
