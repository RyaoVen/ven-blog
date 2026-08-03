package emailhtml

import (
	"fmt"
	"html/template"
	"strings"

	"ven_hybird/build/application/moderationapp"
)

// ===== 验证码邮件 =====

// codeBodyData 验证码正文模板数据。
type codeBodyData struct {
	Code         string
	ValidityHint string
}

// RenderVerificationCode 登录验证码邮件：大号等宽 code 展示 + 有效期提示。
func RenderVerificationCode(siteName, siteURL, code, validityHint string) string {
	body := renderBody(codeBodyTmpl, codeBodyData{Code: code, ValidityHint: validityHint})
	return RenderLayout(siteName, siteURL, "登录验证码", body)
}

var codeBodyTmpl = template.Must(template.New("code").Parse(`
<p style="margin:0 0 12px;color:` + colorSecondary + `;">你的登录验证码是：</p>
<div style="margin:16px 0;padding:18px 24px;border:1px solid ` + colorBorderStr + `;border-radius:3px;background-color:` + colorBGSubtle + `;text-align:center;">
<span style="font-family:` + fontMono + `;font-size:34px;font-weight:600;letter-spacing:0.18em;color:` + colorAccent + `;">{{.Code}}</span>
</div>
<p style="margin:0;color:` + colorSecondary + `;font-size:13px;">{{.ValidityHint}}</p>
`))

// ===== @提及邮件 =====

// mentionBodyData @提及正文模板数据。
type mentionBodyData struct {
	Excerpt string
	Link    string
}

// RenderMention @提及通知邮件：原文摘录 + "查看原文"按钮（链接 = siteURL + path）。
func RenderMention(siteName, siteURL, excerpt, path string) string {
	body := renderBody(mentionBodyTmpl, mentionBodyData{Excerpt: excerpt, Link: joinURL(siteURL, path)})
	return RenderLayout(siteName, siteURL, "有人在评论中提到了你", body)
}

var mentionBodyTmpl = template.Must(template.New("mention").Parse(`
<div style="margin:0 0 20px;padding:12px 16px;border-left:3px solid ` + colorAccent + `;background-color:` + colorBGSubtle + `;color:` + colorSecondary + `;font-size:14px;line-height:1.7;">「{{.Excerpt}}」</div>
<p style="margin:0 0 20px;">点击下方按钮查看原文：</p>
<p style="margin:0;"><a href="{{.Link}}" style="` + actionButtonStyle + `">查看原文</a></p>
`))

// ===== 订阅新文章邮件 =====

// newArticleBodyData 订阅新文章正文模板数据。
type newArticleBodyData struct {
	ArticleTitle string
	Summary      string
	Link         string
}

// RenderNewArticle 订阅新文章通知邮件：文章标题 + 摘要 + "查看原文"按钮（链接 = siteURL + path）。
// 当前业务尚无订阅通知发送点（订阅仅记录邮箱），本函数为通知功能就绪的渲染器，可直接接线。
func RenderNewArticle(siteName, siteURL, articleTitle, summary, path string) string {
	body := renderBody(newArticleBodyTmpl, newArticleBodyData{
		ArticleTitle: articleTitle,
		Summary:      summary,
		Link:         joinURL(siteURL, path),
	})
	return RenderLayout(siteName, siteURL, "新文章发布", body)
}

var newArticleBodyTmpl = template.Must(template.New("new_article").Parse(`
<p style="margin:0 0 16px;font-size:16px;font-weight:600;color:` + colorText + `;">{{.ArticleTitle}}</p>
<p style="margin:0 0 20px;color:` + colorSecondary + `;">{{.Summary}}</p>
<p style="margin:0;"><a href="{{.Link}}" style="` + actionButtonStyle + `">查看原文</a></p>
`))

// ===== 审核摘要邮件 =====

// summarySectionData 明细段（如"自动驳回"）。
type summarySectionData struct {
	Label string // 段标题（如 自动驳回）
	Note  string // 补充说明（如 AI 无法确定是否违规，请人工判断）
	Items []summaryItemData
}

// summaryItemData 单条明细：Heading 为编号+宿主+用户+原因，Content 为内容摘录（可为空）。
type summaryItemData struct {
	Heading string
	Content string
}

// summaryBodyData 摘要正文模板数据。
type summaryBodyData struct {
	StatLine   string
	HasAbnormal bool
	Sections   []summarySectionData
	AdminLink  string
}

// RenderModerationSummary 内容审核摘要邮件：统计行 + 驳回/需人工复核/判定失败三段明细列表 +
// "前往管理面板"按钮（siteURL + /admin/comments）。nil 结果按空处理（防御）。
func RenderModerationSummary(siteName, siteURL string, r *moderationapp.Result) string {
	if r == nil {
		r = &moderationapp.Result{}
	}
	data := summaryBodyData{
		StatLine: fmt.Sprintf("本轮共审核 %d 条：自动放行 %d 条，自动驳回 %d 条，需人工复核 %d 条，判定失败 %d 条（已保持待审）。",
			r.Processed, r.Approved, r.Rejected, r.Uncertain, r.Failed),
		AdminLink: joinURL(siteURL, "/admin/comments"),
	}
	if r.Rejected > 0 {
		items := make([]summaryItemData, 0, len(r.RejectedItems))
		for i, it := range r.RejectedItems {
			h := itemHeading(i+1, it)
			if it.Reason != "" {
				h += " | 原因：" + it.Reason
			}
			items = append(items, summaryItemData{Heading: h, Content: excerpt(it.Content)})
		}
		data.Sections = append(data.Sections, summarySectionData{Label: "自动驳回", Items: items})
	}
	if r.Uncertain > 0 {
		items := make([]summaryItemData, 0, len(r.UncertainItems))
		for i, it := range r.UncertainItems {
			items = append(items, summaryItemData{Heading: itemHeading(i+1, it), Content: excerpt(it.Content)})
		}
		data.Sections = append(data.Sections, summarySectionData{
			Label: "需人工复核", Note: "（AI 无法确定是否违规，请人工判断）", Items: items,
		})
	}
	if r.Failed > 0 {
		items := make([]summaryItemData, 0, len(r.FailedItems))
		for i, it := range r.FailedItems {
			items = append(items, summaryItemData{Heading: itemHeading(i+1, it)})
		}
		data.Sections = append(data.Sections, summarySectionData{
			Label: "判定失败", Note: "（LLM 调用失败，已保持待审，下轮自动重试）", Items: items,
		})
	}
	data.HasAbnormal = len(data.Sections) > 0
	return RenderLayout(siteName, siteURL, "内容审核摘要", renderBody(summaryBodyTmpl, data))
}

var summaryBodyTmpl = template.Must(template.New("summary").Parse(`
<p style="margin:0 0 16px;color:` + colorSecondary + `;">{{.StatLine}}</p>
{{if .HasAbnormal}}
{{range .Sections}}
<div style="margin:0 0 18px;">
<div style="font-family:` + fontMono + `;font-size:13px;font-weight:600;letter-spacing:0.05em;color:` + colorText + `;">{{.Label}}{{if .Note}} {{.Note}}{{end}}</div>
{{range .Items}}
<div style="margin-top:8px;padding:10px 14px;border:1px solid ` + colorBorder + `;border-radius:3px;background-color:` + colorBGSubtle + `;">
<div style="font-size:13px;color:` + colorText + `;">{{.Heading}}</div>
{{if .Content}}<div style="margin-top:4px;font-size:13px;color:` + colorSecondary + `;">{{.Content}}</div>{{end}}
</div>
{{end}}
</div>
{{end}}
<p style="margin:0;"><a href="{{.AdminLink}}" style="` + actionButtonStyle + `">前往管理面板</a></p>
{{else}}
<p style="margin:0;color:` + colorSecondary + `;">本轮无异常项，无需人工处理。</p>
{{end}}
`))

// itemHeading 明细行：编号 + 类型 + 用户 + 宿主（与改造前纯文本行同构）。
func itemHeading(n int, it moderationapp.Item) string {
	return fmt.Sprintf("%d. [%s #%d] 用户：%s | 宿主：《%s》", n, kindLabel(it.Kind), it.ID, it.Username, it.HostTitle)
}

// kindLabel 明细类型中文标签。
func kindLabel(kind string) string {
	if kind == moderationapp.KindGuestbook {
		return "留言板"
	}
	return "评论"
}

// excerpt 内容摘录 80 字符（同 interactions.go excerptOf 先例）。
func excerpt(content string) string {
	runes := []rune(content)
	if len(runes) <= 80 {
		return content
	}
	return string(runes[:80]) + "…"
}

// renderBody 执行静态正文模板（数据字段与结构体一一对应，执行失败视为编程错误）。
func renderBody(t *template.Template, data any) string {
	var b strings.Builder
	_ = t.Execute(&b, data)
	return b.String()
}
