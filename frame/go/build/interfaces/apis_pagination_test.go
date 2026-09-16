package interfaces

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ven_hybird/build/domain/post"
)

// TestAPIPostsPagination 分页与列表裁剪（issue #14：测评报告 #8）。
func TestAPIPostsPagination(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	// 种 7 篇（fakePostRepo 直接插 map；Create 语义不适用——直接构造）。
	for i := int64(1); i <= 7; i++ {
		env.postRepo.posts[i] = &post.Post{
			ID:       i,
			Title:    "T",
			Content:  longContent,
			Category: "技术",
			AuthorID: 1,
		}
	}
	if err := env.app.RegisterRole("reader", nil); err != nil {
		t.Fatalf("RegisterRole reader: %v", err)
	}
	if err := env.app.RegisterRole("author", nil); err != nil {
		t.Fatalf("RegisterRole author: %v", err)
	}
	if err := RegisterAPIs(env.app, env.posts, nil, env.authorNameFn); err != nil {
		t.Fatalf("RegisterAPIs: %v", err)
	}

	// 默认 page=1 size=10。
	resp, body := env.getApp(t, "/api/posts")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default list: %d %s", resp.StatusCode, body)
	}
	// size=2&page=2：返回 2 条 + total=7。
	resp, body = env.getApp(t, "/api/posts?page=2&size=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("paged list: %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"total":7`) {
		t.Fatalf("total 应为 7: %s", body)
	}
	// 列表项不含 content 字段（裁剪硬断言）。
	if strings.Contains(body, `"content":`) {
		t.Fatalf("列表视图不应含正文: %s", body)
	}
	// 非法参数 400。
	for _, q := range []string{"?page=abc", "?page=0", "?size=999", "?size=-1"} {
		resp, _ = env.getApp(t, "/api/posts"+q)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s 应 400，得 %d", q, resp.StatusCode)
		}
	}
}

// getApp 发起 GET 请求到 fiber app（无鉴权头）。
func (e *mcpTestEnv) getApp(t *testing.T, target string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	resp, err := e.server.App().Test(req)
	if err != nil {
		t.Fatalf("get %s failed: %v", target, err)
	}
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

const longContent = "# 长正文\n\n用于断言列表裁剪——正文字段不应出现在列表响应。"
