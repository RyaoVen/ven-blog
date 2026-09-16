package interfaces

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"ven_hybird/build/domain/post"
)

// TestSEO_Basics favicon/robots/sitemap/manifest 四资源（issue #15）。
func TestSEO_Basics(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	mod, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00Z")
	env.postRepo.posts[7] = &post.Post{ID: 7, Title: "测评文", UpdatedAt: mod, Category: "技术", AuthorID: 1}
	siteURL := "https://blog.example.com"
	if err := RegisterSEO(env.app, env.posts, siteURL); err != nil {
		t.Fatalf("RegisterSEO: %v", err)
	}

	// robots.txt：允许全站 + sitemap 指向 + 后台/API 禁爬。
	resp, body := env.getApp(t, "/robots.txt")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Sitemap: "+siteURL+"/sitemap.xml") ||
		!strings.Contains(body, "Disallow: /admin") {
		t.Fatalf("robots.txt 不对：%d %s", resp.StatusCode, body)
	}

	// sitemap：含静态页 + posts lastmod。
	resp, body = env.getApp(t, "/sitemap.xml")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "<loc>"+siteURL+"/posts/7</loc>") ||
		!strings.Contains(body, "<lastmod>2026-09-01</lastmod>") || !strings.Contains(body, "<loc>"+siteURL+"/docs</loc>") {
		t.Fatalf("sitemap 不对：%d %s", resp.StatusCode, body)
	}

	// favicon：非空二进制（ICO 头 00 00 01 00 01 00）。
	resp, body = env.getApp(t, "/favicon.ico")
	if resp.StatusCode != http.StatusOK || len(body) < 100 ||
		!strings.Contains(body[:6], string([]byte{0, 0, 1, 0, 1, 0})) {
		t.Fatalf("favicon 不对：%d len=%d", resp.StatusCode, len(body))
	}

	// manifest：JSON。
	resp, body = env.getApp(t, "/manifest.json")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `#0d9488`) {
		t.Fatalf("manifest 不对：%d %s", resp.StatusCode, body)
	}
}
