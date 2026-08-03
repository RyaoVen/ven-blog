// 链接预览 SSRF 防护测试：连接层 IP 校验（回环/私网/链路本地含云元数据/组播/未指定）、
// DNS 解析到受限地址、重定向逐跳校验（目标 IP/scheme/端口/跳数）、端口白名单，
// 以及 /api/admin/linkpreview handler 的错误映射与公网正常通过。
package interfaces

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"ven_hybird/hybrid"
	"ven_hybird/internal/auth"
	"ven_hybird/internal/config"
	"ven_hybird/internal/httpserver"
	"ven_hybird/internal/pagepattern"
	"ven_hybird/internal/ssr"

	"github.com/gofiber/fiber/v2"
)

// fetchPreview 用真实抓取客户端（含 SSRF 拦截传输层）请求 rawURL。
func fetchPreview(t *testing.T, rawURL string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return previewHTTPClient.Do(req)
}

// closeResp 兜底关闭响应体（校验拒绝分支的 resp 可能非 nil，如重定向被拒时）。
func closeResp(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// withFakeDNS 注入假 DNS 解析（t.Cleanup 自动恢复）。
func withFakeDNS(t *testing.T, lookup func(ctx context.Context, host string) ([]net.IPAddr, error)) {
	t.Helper()
	old := previewLookupIP
	previewLookupIP = lookup
	t.Cleanup(func() { previewLookupIP = old })
}

// allowTestAddr 放行指定 host:port（测试用 httptest 监听 127.0.0.1:随机端口）：
// 同时放行连接层校验与 handler 端口白名单，其余地址/端口走真实校验。
func allowTestAddr(t *testing.T, addrs ...string) {
	t.Helper()
	oldCheck := previewCheckHost
	oldPort := previewPortAllowed
	allowed := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		allowed[a] = true
	}
	previewCheckHost = func(ctx context.Context, host, port string) ([]net.IPAddr, error) {
		if allowed[net.JoinHostPort(host, port)] {
			if ip := net.ParseIP(host); ip != nil {
				return []net.IPAddr{{IP: ip}}, nil
			}
		}
		return oldCheck(ctx, host, port)
	}
	previewPortAllowed = func(u *url.URL) bool {
		if allowed[u.Host] {
			return true
		}
		return oldPort(u)
	}
	t.Cleanup(func() {
		previewCheckHost = oldCheck
		previewPortAllowed = oldPort
	})
}

// hostPortOf 取 httptest 服务器的 host:port。
func hostPortOf(s *httptest.Server) string {
	u, _ := url.Parse(s.URL)
	return u.Host
}

// TestLinkPreview_RejectsRestrictedAddr 受限地址直连：连接层校验在 dial 前拒绝，不产生任何网络连接。
func TestLinkPreview_RejectsRestrictedAddr(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://172.16.0.1/",
		"http://172.31.255.254/",
		"http://192.168.1.1/",
		"http://169.254.169.254/latest/meta-data/", // 云元数据地址
		"http://0.0.0.0/",
		"http://224.0.0.1/",
		"http://[::1]/",
		"http://[fe80::1]/",
	} {
		resp, err := fetchPreview(t, raw)
		if err == nil {
			closeResp(resp)
			t.Fatalf("url %s: expected rejection, got response %d", raw, resp.StatusCode)
		}
		if !strings.Contains(err.Error(), "not allowed") {
			t.Fatalf("url %s: expected address-rejection error, got %v", raw, err)
		}
	}
}

// TestLinkPreview_RejectsDomainResolvingRestricted 域名解析到受限地址（含元数据）、
// 多 IP 混入受限地址（DNS 重绑定）、解析失败与无地址，均拒绝。
func TestLinkPreview_RejectsDomainResolvingRestricted(t *testing.T) {
	withFakeDNS(t, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		switch host {
		case "private.test":
			return []net.IPAddr{{IP: net.ParseIP("10.0.0.5")}}, nil
		case "metadata.test":
			return []net.IPAddr{{IP: net.ParseIP("169.254.169.254")}}, nil
		case "mixed.test": // 多 IP 含受限地址 → 整体拒绝
			return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("127.0.0.1")}}, nil
		case "dnsfail.test":
			return nil, errors.New("dns boom")
		case "noaddr.test":
			return []net.IPAddr{}, nil
		}
		return nil, errors.New("unexpected host " + host)
	})

	for _, raw := range []string{
		"http://private.test/",
		"http://metadata.test/",
		"http://mixed.test/",
		"http://dnsfail.test/",
		"http://noaddr.test/",
	} {
		resp, err := fetchPreview(t, raw)
		if err == nil {
			closeResp(resp)
			t.Fatalf("url %s: expected rejection, got response %d", raw, resp.StatusCode)
		}
		if !strings.Contains(err.Error(), "linkpreview:") {
			t.Fatalf("url %s: expected linkpreview rejection, got %v", raw, err)
		}
	}
}

// TestLinkPreview_RejectsNonWhitelistedPort 非白名单端口在连接层拒绝（不产生网络连接）。
func TestLinkPreview_RejectsNonWhitelistedPort(t *testing.T) {
	for _, raw := range []string{"http://example.com:8080/", "http://example.com:22/"} {
		resp, err := fetchPreview(t, raw)
		if err == nil {
			closeResp(resp)
			t.Fatalf("url %s: expected rejection, got response %d", raw, resp.StatusCode)
		}
		if !strings.Contains(err.Error(), "port") {
			t.Fatalf("url %s: expected port rejection, got %v", raw, err)
		}
	}
}

// TestLinkPreview_FollowsPublicRedirect 公网重定向正常跟随，最终 URL 为落地页。
func TestLinkPreview_FollowsPublicRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<html><head><title>落地页</title></head><body>ok</body></html>`)
	}))
	defer final.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/landing", http.StatusFound)
	}))
	defer first.Close()
	allowTestAddr(t, hostPortOf(first), hostPortOf(final))

	resp, err := fetchPreview(t, first.URL)
	if err != nil {
		t.Fatalf("public redirect should be followed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if want := final.URL + "/landing"; resp.Request.URL.String() != want {
		t.Fatalf("expected final url %s, got %s", want, resp.Request.URL)
	}
}

// TestLinkPreview_BlocksRedirectToRestricted 重定向到内网：首跳放行，第二跳连接前被拒，
// 且内网地址未被实际请求（首跳服务器只收到 1 次请求）。
func TestLinkPreview_BlocksRedirectToRestricted(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, "http://10.0.0.1/", http.StatusFound)
	}))
	defer server.Close()
	allowTestAddr(t, hostPortOf(server))

	resp, err := fetchPreview(t, server.URL)
	if err == nil {
		closeResp(resp)
		t.Fatal("redirect to private address must be rejected")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected address rejection on redirect hop, got %v", err)
	}
	if hits != 1 {
		t.Fatalf("first-hop server hits = %d, want 1 (redirect target must not be requested)", hits)
	}
}

// TestLinkPreview_BlocksUnsafeRedirectTarget 重定向到非 http(s) scheme 与非白名单端口均拒绝
// （端口在连接层拦截：非白名单端口根本不产生连接）。
func TestLinkPreview_BlocksUnsafeRedirectTarget(t *testing.T) {
	for _, tc := range []struct {
		location string
		wantErr  string
	}{
		{"file:///etc/passwd", "redirect scheme"},
		{"http://127.0.0.1:9999/", "port"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", tc.location)
			w.WriteHeader(http.StatusFound)
		}))
		allowTestAddr(t, hostPortOf(server))

		resp, err := fetchPreview(t, server.URL)
		server.Close()
		if err == nil {
			closeResp(resp)
			t.Fatalf("location %s: expected rejection, got response %d", tc.location, resp.StatusCode)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("location %s: expected error containing %q, got %v", tc.location, tc.wantErr, err)
		}
	}
}

// TestLinkPreview_StopsAfter10Redirects 重定向循环超过 10 跳被截断。
func TestLinkPreview_StopsAfter10Redirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer server.Close()
	allowTestAddr(t, hostPortOf(server))

	resp, err := fetchPreview(t, server.URL)
	if err == nil {
		closeResp(resp)
		t.Fatal("redirect loop must be stopped")
	}
	if !strings.Contains(err.Error(), "10 redirects") {
		t.Fatalf("expected hop limit error, got %v", err)
	}
}

/* ===== /api/admin/linkpreview handler 全链路 ===== */

// newLinkPreviewApp 构造注册了 /api/admin/linkpreview 的应用（author 守卫 + 测试登录端点）。
func newLinkPreviewApp(t *testing.T) (*hybrid.App, *httpserver.Server) {
	t.Helper()
	cfg := config.Config{
		NodeSubmitTimeout: 5 * time.Second,
		RenderTimeout:     10 * time.Second,
	}
	cfg.IsrDir = t.TempDir()
	cfg.IsrEnabled = true
	client := &fakeSSRClient{submitted: make(chan ssr.RenderTask, 1)}
	pending := ssr.NewPendingRegistry(10)
	patterns := pagepattern.NewValidator(nil)
	server := httpserver.New(cfg, client, pending, fakeHookIDs{}, patterns)
	app := hybrid.New(server)
	if err := app.RegisterRole("author", nil); err != nil {
		t.Fatalf("register author role: %v", err)
	}
	if err := RegisterLinkPreview(app); err != nil {
		t.Fatalf("RegisterLinkPreview: %v", err)
	}
	server.App().Post("/test-login-author", func(ctx *fiber.Ctx) error {
		return server.GrantAuth(ctx, "author")
	})
	return app, server
}

// loginAuthor 登录 author 角色，返回会话 cookie。
func loginAuthor(t *testing.T, server *httpserver.Server) *http.Cookie {
	t.Helper()
	resp, err := server.App().Test(httptest.NewRequest(http.MethodPost, "/test-login-author", nil))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == auth.AuthCookieName {
			return c
		}
	}
	t.Fatal("no auth cookie issued")
	return nil
}

// callLinkPreview 请求 /api/admin/linkpreview（cookie 为 nil 表示未登录）。
func callLinkPreview(t *testing.T, server *httpserver.Server, cookie *http.Cookie, rawURL string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/linkpreview?url="+url.QueryEscape(rawURL), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("linkpreview request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

// TestLinkPreviewHandler handler 全链路：鉴权、解析、SSRF 拒绝与错误映射。
func TestLinkPreviewHandler(t *testing.T) {
	og := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<!doctype html><html><head>`+
			`<meta property="og:title" content="链接标题">`+
			`<meta name="description" content="链接简介">`+
			`<meta property="og:image" content="/cover.png">`+
			`<title>回退标题</title>`+
			`<link rel="icon" href="/favicon.ico">`+
			`</head><body>正文</body></html>`)
	}))
	defer og.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://10.0.0.1/", http.StatusFound)
	}))
	defer redirector.Close()
	allowTestAddr(t, hostPortOf(og), hostPortOf(redirector))

	_, server := newLinkPreviewApp(t)
	cookie := loginAuthor(t, server)

	t.Run("unauthenticated is 401", func(t *testing.T) {
		resp, _ := callLinkPreview(t, server, nil, og.URL)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("public site parsed", func(t *testing.T) {
		resp, body := callLinkPreview(t, server, cookie, og.URL)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
		}
		for _, want := range []string{`"title":"链接标题"`, `"desc":"链接简介"`} {
			if !strings.Contains(body, want) {
				t.Fatalf("body missing %s:\n%s", want, body)
			}
		}
	})

	t.Run("redirect to private is 502", func(t *testing.T) {
		resp, body := callLinkPreview(t, server, cookie, redirector.URL)
		if resp.StatusCode != http.StatusBadGateway {
			t.Fatalf("expected 502, got %d (%s)", resp.StatusCode, body)
		}
	})

	t.Run("metadata address is 502", func(t *testing.T) {
		resp, body := callLinkPreview(t, server, cookie, "http://169.254.169.254/latest/meta-data/")
		if resp.StatusCode != http.StatusBadGateway {
			t.Fatalf("expected 502, got %d (%s)", resp.StatusCode, body)
		}
	})

	t.Run("non-whitelist port is 400", func(t *testing.T) {
		resp, body := callLinkPreview(t, server, cookie, "http://example.com:8080/")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d (%s)", resp.StatusCode, body)
		}
	})

	t.Run("bad scheme is 400", func(t *testing.T) {
		resp, body := callLinkPreview(t, server, cookie, "ftp://example.com/")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d (%s)", resp.StatusCode, body)
		}
	})
}
