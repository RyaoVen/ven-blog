// 摘要邮件正文构建（纯函数，可单测）。
package moderator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ven_hybird/build/application/moderationapp"
)

// buildSummaryEmail 构建摘要邮件（subject, text）。纯函数，可单测。
// 主题：ven-blog 内容审核摘要（MM-DD HH:mm）；
// 正文含驳回/需人工复核/判定失败三段明细与后台面板链接；
// 全正常或空结果返回占位文案（防御：RunOnce 仅在存在异常项时才调用）。
func buildSummaryEmail(r *moderationapp.Result, siteURL string) (subject, text string) {
	subject = "ven-blog 内容审核摘要（" + time.Now().Format("01-02 15:04") + "）"
	if r == nil {
		r = &moderationapp.Result{}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("本轮共审核 %d 条：自动放行 %d 条，自动驳回 %d 条，需人工复核 %d 条，判定失败 %d 条（已保持待审）。",
		r.Processed, r.Approved, r.Rejected, r.Uncertain, r.Failed))
	if r.Rejected+r.Uncertain+r.Failed == 0 {
		b.WriteString("\n\n本轮无异常项，无需人工处理。")
	} else {
		if r.Rejected > 0 {
			b.WriteString("\n\n【自动驳回】" + strconv.Itoa(r.Rejected) + " 条")
			for i, it := range r.RejectedItems {
				b.WriteString("\n" + rejectedLine(i+1, it))
			}
		}
		if r.Uncertain > 0 {
			b.WriteString("\n\n【需人工复核】" + strconv.Itoa(r.Uncertain) + " 条（AI 无法确定是否违规，请人工判断）")
			for i, it := range r.UncertainItems {
				b.WriteString("\n" + reviewLine(i+1, it))
			}
		}
		if r.Failed > 0 {
			b.WriteString("\n\n【判定失败】" + strconv.Itoa(r.Failed) + " 条（LLM 调用失败，已保持待审，下轮自动重试）")
			for i, it := range r.FailedItems {
				b.WriteString("\n" + failedLine(i+1, it))
			}
		}
	}
	b.WriteString("\n\n处理入口：" + siteURL + "/admin/comments")
	return subject, b.String()
}

// rejectedLine 驳回明细行：编号 + 宿主 + 用户 + 原因 + 内容摘录。
func rejectedLine(n int, it moderationapp.Item) string {
	line := fmt.Sprintf("%d. [%s #%d] 用户：%s | 宿主：《%s》", n, kindLabel(it.Kind), it.ID, it.Username, it.HostTitle)
	if it.Reason != "" {
		line += " | 原因：" + it.Reason
	}
	return line + "\n   内容：" + excerpt(it.Content)
}

// reviewLine 需人工复核明细行：编号 + 宿主 + 用户 + 内容摘录。
func reviewLine(n int, it moderationapp.Item) string {
	return fmt.Sprintf("%d. [%s #%d] 用户：%s | 宿主：《%s》\n   内容：%s",
		n, kindLabel(it.Kind), it.ID, it.Username, it.HostTitle, excerpt(it.Content))
}

// failedLine 判定失败明细行（无内容展示，仅标识）。
func failedLine(n int, it moderationapp.Item) string {
	return fmt.Sprintf("%d. [%s #%d] 用户：%s | 宿主：《%s》",
		n, kindLabel(it.Kind), it.ID, it.Username, it.HostTitle)
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
