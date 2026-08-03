// 链接预览解析：编辑器「插入链接块」用，服务端抓取目标页解析站名/简介/图标（author 守卫）。
package interfaces

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
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

// previewHTTPClient 抓取客户端：超时 8s；重定向逐跳校验（scheme/跳数），
// 每跳连接前的 IP/端口校验由 previewTransport 执行（SSRF 防护，见 previewDialContext）。
var previewHTTPClient = &http.Client{
	Transport:     previewTransport,
	Timeout:       8 * time.Second,
	CheckRedirect: previewCheckRedirect,
}

// previewTransport 抓取传输层：强制直连（不走环境代理——代理会把请求转发到内网，绕过地址校验），
// 每跳连接前解析 DNS 并拒绝受限地址（回环/私网/链路本地/组播/未指定）。
var previewTransport = &http.Transport{
	Proxy:               nil, // 禁止环境代理，强制直连
	DialContext:         previewDialContext,
	TLSHandshakeTimeout: 5 * time.Second,
}

// previewLookupIP 解析 host 为 IP 列表（独立成变量，测试可注入假 DNS）。
var previewLookupIP = func(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

// previewCheckHost 校验目标 host:port 是否允许连接，返回通过校验的 IP 列表（供直连）。
// 测试可整体替换（如放行 httptest 的 127.0.0.1:随机端口）。
var previewCheckHost = checkHostAllowed

// previewPortAllowed 端口白名单判定：仅 80/443（空 = 默认端口）。
// 独立成变量，测试放行 httptest 随机端口。
var previewPortAllowed = func(u *url.URL) bool {
	p := u.Port()
	return p == "" || p == "80" || p == "443"
}

// restrictedIP 是否受限地址：回环（127.0.0.0/8、::1）、私网（10/8、172.16/12、192.168/16、fc00::/7）、
// 链路本地（169.254/16 含云元数据 169.254.169.254、fe80::/10）、组播、未指定地址。
func restrictedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsMulticast() || ip.IsUnspecified()
}

// checkHostAllowed 端口白名单 + DNS 解析校验：
// 仅允许 80/443；解析失败、无地址、任一解析结果落在受限地址段
// （含"多 IP 中混有受限地址"的 DNS 重绑定场景）均拒绝。
func checkHostAllowed(ctx context.Context, host, port string) ([]net.IPAddr, error) {
	if port != "80" && port != "443" {
		return nil, fmt.Errorf("linkpreview: port %s not allowed", port)
	}
	ips, err := previewLookupIP(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("linkpreview: resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("linkpreview: %s has no addresses", host)
	}
	for _, ip := range ips {
		if restrictedIP(ip.IP) {
			return nil, fmt.Errorf("linkpreview: address %s not allowed", ip.IP)
		}
	}
	return ips, nil
}

// previewDialContext 连接拦截：校验通过后直连首个通过校验的 IP，
// 不把 hostname 再交给系统解析（消除"校验后、连接前"的 DNS 竞态）。
func previewDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("linkpreview: invalid dial address %q: %w", addr, err)
	}
	ips, err := previewCheckHost(ctx, host, port)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("linkpreview: %s has no addresses", host)
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}

// previewCheckRedirect 重定向逐跳校验：最多 10 跳；目标仅限 http/https scheme。
// 每跳目标 IP 与端口由 previewTransport 连接层校验（重定向到内网/非白名单端口同样被拒）。
func previewCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("linkpreview: stopped after 10 redirects")
	}
	if u := req.URL; u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("linkpreview: redirect scheme %q not allowed", u.Scheme)
	}
	return nil
}

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
		// 端口白名单 80/443（显式端口；连接层同样拦截非白名单端口，这里尽早 400）
		if !previewPortAllowed(target) {
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
