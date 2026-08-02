// 链接预览解析：编辑器「插入链接块」用，服务端抓取目标页解析站名/简介/图标（author 守卫）。
package interfaces

import (
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"ven_hybird/hybrid"
)

// linkPreview 解析结果（编辑器预览与链接块三行数据）。
type linkPreview struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Icon  string `json:"icon"`
	URL   string `json:"url"` // 最终 URL（跟随跳转后）
}

// previewHTTPClient 抓取客户端：超时 8s，默认跟随重定向。
var previewHTTPClient = &http.Client{Timeout: 8 * time.Second}

// previewMaxBody 只读前 1MB（og 标签都在 head 内，防止超大页面拖垮网关）。
const previewMaxBody = 1 << 20

var (
	titleTagRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	metaTagRe  = regexp.MustCompile(`(?is)<meta\s[^>]*>`)
	linkTagRe  = regexp.MustCompile(`(?is)<link\s[^>]*>`)
	attrRe     = regexp.MustCompile(`([\w-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	spaceRe    = regexp.MustCompile(`\s+`)
)

// attrsOf 提取标签属性表（键小写；双/单引号均支持）。
func attrsOf(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		value := m[2]
		if value == "" {
			value = m[3]
		}
		attrs[strings.ToLower(m[1])] = html.UnescapeString(value)
	}
	return attrs
}

// cleanText 归一空白并截断（HTML 实体已解码）。
func cleanText(s string, maxRunes int) string {
	s = spaceRe.ReplaceAllString(strings.TrimSpace(html.UnescapeString(s)), " ")
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return s
}

// resolveURL 把（可能相对的）href 解析为绝对 URL。
func resolveURL(base *url.URL, href string) string {
	ref, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

// parseLinkPreview 从 HTML 解析标题/简介/图标（og 优先，回退 title/description/favicon.ico）。
func parseLinkPreview(body string, finalURL *url.URL) linkPreview {
	p := linkPreview{URL: finalURL.String()}
	var iconHref string
	for _, tag := range metaTagRe.FindAllString(body, -1) {
		attrs := attrsOf(tag)
		key := strings.ToLower(attrs["property"])
		if key == "" {
			key = strings.ToLower(attrs["name"])
		}
		content := attrs["content"]
		if content == "" {
			continue
		}
		switch key {
		case "og:title", "twitter:title":
			if p.Title == "" {
				p.Title = content
			}
		case "og:description", "description", "twitter:description":
			if p.Desc == "" {
				p.Desc = content
			}
		}
	}
	if p.Title == "" {
		if m := titleTagRe.FindStringSubmatch(body); m != nil {
			p.Title = m[1]
		}
	}
	for _, tag := range linkTagRe.FindAllString(body, -1) {
		attrs := attrsOf(tag)
		rel := strings.ToLower(attrs["rel"])
		if strings.Contains(rel, "icon") && attrs["href"] != "" {
			iconHref = attrs["href"]
			// apple-touch-icon 优先于普通 icon（分辨率更高），先到先得高分
			if strings.Contains(rel, "apple-touch-icon") {
				break
			}
		}
	}
	if iconHref == "" {
		iconHref = "/favicon.ico"
	}
	p.Title = cleanText(p.Title, 120)
	p.Desc = cleanText(p.Desc, 200)
	p.Icon = resolveURL(finalURL, iconHref)
	return p
}

// RegisterLinkPreview 注册链接预览解析接口（仅 author：编辑器插入链接块时调用）。
func RegisterLinkPreview(a *hybrid.App) error {
	return a.Get("/admin/linkpreview", []string{"author"}, func(c *hybrid.ApiCtx) error {
		raw := strings.TrimSpace(c.Query("url"))
		target, err := url.Parse(raw)
		if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
			return c.Error(400, "invalid url")
		}
		req, err := http.NewRequest(http.MethodGet, target.String(), nil)
		if err != nil {
			return c.Error(400, "invalid url")
		}
		req.Header.Set("User-Agent", "ven-blog linkpreview (+https://github.com/RyaoVen/ven-blog)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		resp, err := previewHTTPClient.Do(req)
		if err != nil {
			return c.Error(502, "fetch failed")
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			return c.Error(502, "fetch failed")
		}
		buf, err := io.ReadAll(io.LimitReader(resp.Body, previewMaxBody))
		if err != nil {
			return c.Error(502, "fetch failed")
		}
		preview := parseLinkPreview(string(buf), resp.Request.URL)
		if preview.Title == "" {
			preview.Title = resp.Request.URL.Host
		}
		return c.JSON(200, preview)
	})
}
