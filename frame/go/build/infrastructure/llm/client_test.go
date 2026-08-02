package llm

import (
	"context"
	"encoding/json"
	"errors"
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
	return NewClient(func() (Config, error) {
		return Config{BaseURL: baseURL, APIKey: "test-key", Model: "test-model"}, nil
	})
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
		configFn: func() (Config, error) { return Config{BaseURL: srv.URL, APIKey: "k", Model: "m"}, nil },
		http:     &http.Client{Timeout: 50 * time.Millisecond},
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

func TestReviewMissingAPIKeyIsError(t *testing.T) {
	// API key 未配置：直接报错不发请求（worker 视为判定失败，绝不放行/驳回）
	c := NewClient(func() (Config, error) { return Config{BaseURL: "http://127.0.0.1:1/v1"}, nil })
	_, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"})
	if err == nil || !strings.Contains(err.Error(), "api key is not configured") {
		t.Fatalf("want api-key-not-configured error, got %v", err)
	}
}

func TestReviewConfigErrorIsError(t *testing.T) {
	// configFn 出错（如 settings 读取失败）→ 无法判定 → error
	c := NewClient(func() (Config, error) { return Config{}, errors.New("settings down") })
	_, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"})
	if err == nil || !strings.Contains(err.Error(), "load config") {
		t.Fatalf("want load-config error, got %v", err)
	}
}

// roundTripFunc 拦截传输层（验证默认端点用，不走真实网络）。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReviewEmptyBaseURLAndModelFallBackToDefaults(t *testing.T) {
	// 空 BaseURL/Model 时默认值在 Review 内回退：请求发往默认端点、使用默认模型
	var gotURL string
	var gotBody chatRequest
	c := &Client{
		configFn: func() (Config, error) { return Config{APIKey: "k"}, nil },
		http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotURL = r.URL.String()
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{\"verdict\":\"approve\",\"reason\":\"\"}"}}]}`)),
				Header:     make(http.Header),
			}, nil
		})},
	}
	if _, err := c.Review(context.Background(), moderation.Request{Host: moderation.HostComment, HostTitle: "t", Content: "c"}); err != nil {
		t.Fatalf("Review: %v", err)
	}
	if gotURL != defaultBaseURL+"/chat/completions" {
		t.Fatalf("url = %q, want %q", gotURL, defaultBaseURL+"/chat/completions")
	}
	if gotBody.Model != defaultModel {
		t.Fatalf("model = %q, want %q", gotBody.Model, defaultModel)
	}
}
