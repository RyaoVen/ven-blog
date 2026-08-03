// 摘要邮件正文构建（纯函数，可单测）。
package moderator

import (
	"time"

	"ven_hybird/build/application/emailhtml"
	"ven_hybird/build/application/moderationapp"
)

// buildSummaryEmail 构建摘要邮件（subject, text）。纯函数，可单测。
// 主题：ven-blog 内容审核摘要（MM-DD HH:mm）；
// 正文为 HTML：驳回/需人工复核/判定失败三段明细列表 + "前往管理面板"按钮；
// 全正常或空结果返回占位文案（防御：RunOnce 仅在存在异常项时才调用）。
// siteURL 为站点对外 URL（siteURLFromEnv()，用于模板站点信息与面板按钮链接）。
func buildSummaryEmail(r *moderationapp.Result, siteURL string) (subject, text string) {
	subject = "ven-blog 内容审核摘要（" + time.Now().Format("01-02 15:04") + "）"
	text = emailhtml.RenderModerationSummary("ven-blog", siteURL, r)
	return subject, text
}
