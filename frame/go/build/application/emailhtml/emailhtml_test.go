// 邮件 HTML 模板纯函数单测：布局结构/站点信息注入/链接拼接/明细列表渲染 + HTML 完整性（标签配对）。
package emailhtml

import (
	"regexp"
	"strings"
	"testing"

	"ven_hybird/build/application/moderationapp"
)

const (
	testSiteName = "ven-blog"
	testSiteURL  = "https://blog.example.com"
)

// ===== 布局结构 =====

func TestRenderLayoutStructure(t *testing.T) {
	got := RenderLayout(testSiteName, testSiteURL, "标题测试", "<p>正文段落</p>")
	for _, want := range []string{
		"<!DOCTYPE html>",
		`<a href="https://blog.example.com"`, // 头部站点名链接
		testSiteName,                         // 站点名（头部 + 页脚）
		"标题测试",                              // 布局标题
		"<p>正文段落</p>",                       // 内容区（可信 HTML 原样注入）
		"https://blog.example.com",            // 页脚站点公网地址
		"©",                                   // 页脚版权
		"border-bottom:1px solid",             // 头部细下边框
		"border-top:1px solid",                // 页脚细上边框
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("layout missing %q:\n%s", want, got)
		}
	}
}

func TestRenderLayoutEscapesSiteInfo(t *testing.T) {
	// 站点名/标题/URL 注入点均须转义（html/template 自动处理），防注入
	got := RenderLayout(`ven<script>alert(1)</script>`, `https://x.example/" onmouseover="alert(1)`, `t<b>x</b>`, "<p>b</p>")
	if strings.Contains(got, "<script>") || strings.Contains(got, "<b>x</b>") {
		t.Fatalf("site info not escaped:\n%s", got)
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;b&gt;x&lt;/b&gt;"} {
		if !strings.Contains(got, want) {
			t.Fatalf("escaped form %q missing:\n%s", want, got)
		}
	}
}

// ===== 验证码 =====

func TestRenderVerificationCode(t *testing.T) {
	got := RenderVerificationCode(testSiteName, testSiteURL, "123456", "10 分钟内有效，请勿泄露。如果不是本人操作，请忽略本邮件。")
	for _, want := range []string{
		"123456",
		"10 分钟内有效，请勿泄露。如果不是本人操作，请忽略本邮件。",
		"font-size:34px",      // 大号等宽展示
		"登录验证码",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("code email missing %q:\n%s", want, got)
		}
	}
}

func TestRenderVerificationCodeEscapesCode(t *testing.T) {
	// 防御：code 理论上为 6 位数字，仍须转义
	got := RenderVerificationCode(testSiteName, testSiteURL, `<b>42</b>`, "hint")
	if strings.Contains(got, "<b>42</b>") || !strings.Contains(got, "&lt;b&gt;42&lt;/b&gt;") {
		t.Fatalf("code not escaped:\n%s", got)
	}
}

// ===== @提及 =====

func TestRenderMentionButtonLinkJoined(t *testing.T) {
	got := RenderMention(testSiteName, testSiteURL, "这段评论写得很精彩，学习了。", "/posts/12")
	for _, want := range []string{
		"这段评论写得很精彩，学习了。",
		`href="https://blog.example.com/posts/12"`, // 按钮链接 = siteURL + path
		"查看原文",
		"有人在评论中提到了你",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mention email missing %q:\n%s", want, got)
		}
	}
}

func TestRenderMentionEscapesExcerpt(t *testing.T) {
	// 摘录来自用户评论，含 HTML 必须转义（属性上下文下引号也转义）
	got := RenderMention(testSiteName, testSiteURL, `<img src=x onerror="alert(1)">`, "/posts/1")
	for _, bad := range []string{"<img", `onerror="`, `alert(1)"`} {
		if strings.Contains(got, bad) {
			t.Fatalf("excerpt not escaped (found %q):\n%s", bad, got)
		}
	}
	if !strings.Contains(got, "&lt;img src=x onerror=&#34;alert(1)&#34;&gt;") {
		t.Fatalf("escaped excerpt missing:\n%s", got)
	}
}

func TestRenderMentionJoinsURLWithTrailingSlash(t *testing.T) {
	got := RenderMention(testSiteName, "https://blog.example.com/", "x", "/moments")
	if !strings.Contains(got, `href="https://blog.example.com/moments"`) {
		t.Fatalf("trailing slash not trimmed:\n%s", got)
	}
}

// ===== 订阅新文章 =====

func TestRenderNewArticle(t *testing.T) {
	got := RenderNewArticle(testSiteName, testSiteURL, "用 Go 写一个博客", "从零开始搭建，包含 SSR 与自动部署。", "/posts/7")
	for _, want := range []string{
		"用 Go 写一个博客",
		"从零开始搭建，包含 SSR 与自动部署。",
		`href="https://blog.example.com/posts/7"`,
		"查看原文",
		"新文章发布",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("new article email missing %q:\n%s", want, got)
		}
	}
}

func TestRenderNewArticleEscapesUserContent(t *testing.T) {
	got := RenderNewArticle(testSiteName, testSiteURL, `t</td><script>alert(1)</script>`, `<b>s</b>`, "/posts/1")
	if strings.Contains(got, "<script>") || strings.Contains(got, "<b>s</b>") {
		t.Fatalf("article title/summary not escaped:\n%s", got)
	}
}

// ===== 审核摘要 =====

func sampleResult() *moderationapp.Result {
	return &moderationapp.Result{
		Processed: 5,
		Approved:  1,
		Rejected:  1,
		Uncertain: 2,
		Failed:    1,
		RejectedItems: []moderationapp.Item{
			{Kind: moderationapp.KindComment, ID: 12, Username: "someone", Content: "点击链接领取红包 https://evil.example", HostTitle: "用 Go 写一个博客", Reason: "包含广告引流链接", PostID: 5},
			{Kind: moderationapp.KindGuestbook, ID: 3, Username: "visitor", Content: "好 好 好 好 好……", HostTitle: "作者主页", Reason: "无意义重复灌水"},
		},
		UncertainItems: []moderationapp.Item{
			{Kind: moderationapp.KindComment, ID: 15, Username: "reader01", Content: "这个说法我觉得有问题，具体是……", HostTitle: "关于 Node SSR 的思考", PostID: 6},
			{Kind: moderationapp.KindGuestbook, ID: 4, Username: "guest02", Content: "嗯", HostTitle: "作者主页"},
		},
		FailedItems: []moderationapp.Item{
			{Kind: moderationapp.KindComment, ID: 17, Username: "user2", Content: "test", HostTitle: "动态 #8", MomentID: 8},
		},
	}
}

func TestRenderModerationSummary(t *testing.T) {
	got := RenderModerationSummary(testSiteName, testSiteURL, sampleResult())
	for _, want := range []string{
		"本轮共审核 5 条：自动放行 1 条，自动驳回 1 条，需人工复核 2 条，判定失败 1 条（已保持待审）。",
		"自动驳回",
		"1. [评论 #12] 用户：someone | 宿主：《用 Go 写一个博客》 | 原因：包含广告引流链接",
		"点击链接领取红包 https://evil.example", // 内容摘录
		"2. [留言板 #3] 用户：visitor | 宿主：《作者主页》 | 原因：无意义重复灌水",
		"需人工复核",
		"（AI 无法确定是否违规，请人工判断）",
		"1. [评论 #15] 用户：reader01 | 宿主：《关于 Node SSR 的思考》",
		"判定失败",
		"（LLM 调用失败，已保持待审，下轮自动重试）",
		"1. [评论 #17] 用户：user2 | 宿主：《动态 #8》",
		`href="https://blog.example.com/admin/comments"`, // 面板按钮 = siteURL + /admin/comments
		"前往管理面板",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary email missing %q:\n%s", want, got)
		}
	}
}

func TestRenderModerationSummaryEscapesItemContent(t *testing.T) {
	r := &moderationapp.Result{
		Rejected: 1,
		RejectedItems: []moderationapp.Item{
			{Kind: moderationapp.KindComment, ID: 1, Username: `<u>u</u>`, Content: `<script>alert(1)</script>`, HostTitle: `<i>t</i>`, Reason: `x" onclick="y`},
		},
	}
	got := RenderModerationSummary(testSiteName, testSiteURL, r)
	for _, bad := range []string{"<script>", "<u>u</u>", "<i>t</i>"} {
		if strings.Contains(got, bad) {
			t.Fatalf("item content not escaped (found %q):\n%s", bad, got)
		}
	}
}

func TestRenderModerationSummaryEmptyResult(t *testing.T) {
	got := RenderModerationSummary(testSiteName, testSiteURL, nil)
	if !strings.Contains(got, "本轮共审核 0 条") || !strings.Contains(got, "本轮无异常项，无需人工处理。") {
		t.Fatalf("empty result should render placeholder:\n%s", got)
	}
	if strings.Contains(got, "前往管理面板") {
		t.Fatalf("empty result should not render admin button:\n%s", got)
	}
}

// ===== HTML 完整性（标签配对） =====

func TestHTMLIntegrity(t *testing.T) {
	outputs := map[string]string{
		"layout":      RenderLayout(testSiteName, testSiteURL, "完整性", "<p>正文</p>"),
		"code":        RenderVerificationCode(testSiteName, testSiteURL, "123456", "hint"),
		"mention":     RenderMention(testSiteName, testSiteURL, "摘录", "/posts/1"),
		"newArticle":  RenderNewArticle(testSiteName, testSiteURL, "标题", "摘要", "/posts/1"),
		"summary":     RenderModerationSummary(testSiteName, testSiteURL, sampleResult()),
		"summaryEmpty": RenderModerationSummary(testSiteName, testSiteURL, nil),
	}
	for name, out := range outputs {
		assertBalancedTags(t, name, out)
	}
}

// voidTags HTML 空元素（无闭合标签）。
var voidTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// tagRe 匹配开/闭标签（忽略 DOCTYPE 与注释：< 后非字母）。
var tagRe = regexp.MustCompile(`<(/)?([a-zA-Z][a-zA-Z0-9]*)((?:\s[^<>]*)?)>`)

// assertBalancedTags 简单校验标签配对：自增栈匹配开闭标签，跳过 void 元素。
func assertBalancedTags(t *testing.T, name, html string) {
	t.Helper()
	var stack []string
	for _, m := range tagRe.FindAllStringSubmatch(html, -1) {
		tag := strings.ToLower(m[2])
		if voidTags[tag] {
			continue
		}
		if m[1] == "/" {
			if len(stack) == 0 || stack[len(stack)-1] != tag {
				t.Fatalf("%s: unbalanced closing </%s>:\n%s", name, tag, html)
			}
			stack = stack[:len(stack)-1]
		} else {
			stack = append(stack, tag)
		}
	}
	if len(stack) > 0 {
		t.Fatalf("%s: unclosed tags %v:\n%s", name, stack, html)
	}
}
