package moderator

import (
	"strings"
	"testing"

	"ven_hybird/build/application/moderationapp"
)

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

func TestBuildSummaryEmail(t *testing.T) {
	subject, html := buildSummaryEmail(sampleResult(), "https://blog.example.com")
	if !strings.HasPrefix(subject, "ven-blog 内容审核摘要（") {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{
		"本轮共审核 5 条：自动放行 1 条，自动驳回 1 条，需人工复核 2 条，判定失败 1 条（已保持待审）",
		"自动驳回",
		"1. [评论 #12] 用户：someone | 宿主：《用 Go 写一个博客》 | 原因：包含广告引流链接",
		"点击链接领取红包 https://evil.example",
		"2. [留言板 #3] 用户：visitor | 宿主：《作者主页》 | 原因：无意义重复灌水",
		"需人工复核",
		"（AI 无法确定是否违规，请人工判断）",
		"1. [评论 #15] 用户：reader01 | 宿主：《关于 Node SSR 的思考》",
		"判定失败",
		"（LLM 调用失败，已保持待审，下轮自动重试）",
		"1. [评论 #17] 用户：user2 | 宿主：《动态 #8》",
		`href="https://blog.example.com/admin/comments"`,
		"前往管理面板",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("email html missing %q:\n%s", want, html)
		}
	}
}

func TestBuildSummaryEmailContentExcerpted(t *testing.T) {
	long := strings.Repeat("长", 200)
	r := &moderationapp.Result{
		Processed: 1,
		Rejected:  1,
		RejectedItems: []moderationapp.Item{
			{Kind: moderationapp.KindComment, ID: 1, Username: "u", Content: long, HostTitle: "t", Reason: "r"},
		},
	}
	_, html := buildSummaryEmail(r, "https://x.example")
	if strings.Contains(html, strings.Repeat("长", 200)) {
		t.Fatal("email should excerpt content to 80 chars")
	}
	if !strings.Contains(html, strings.Repeat("长", 80)+"…") {
		t.Fatal("email should contain 80-char excerpt with ellipsis")
	}
}

func TestBuildSummaryEmailEmptyResult(t *testing.T) {
	subject, html := buildSummaryEmail(nil, "https://blog.example.com")
	if !strings.HasPrefix(subject, "ven-blog 内容审核摘要（") {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(html, "本轮无异常项") {
		t.Fatalf("empty result should render placeholder html:\n%s", html)
	}
	if strings.Contains(html, "前往管理面板") {
		t.Fatalf("empty result should not render admin button:\n%s", html)
	}
}
