// 按审核结果做失效声明（纯函数，可单测）。
// 失效只在接口层声明（AGENTS.md 红线）：AutoReview 用例只返回统计与明细。
package moderator

import (
	"strconv"

	"ven_hybird/build/application/moderationapp"
)

// applyInvalidations 按审核结果逐条失效（只处理产生读者可见性变化的条目）：
//   - 评论被 AI 放行：宿主=文章 → /posts + / + /posts/:id（同 declarePostsChanged，interfaces/apis.go:116）；
//     宿主=动态 → /moments（interactions.go:154 先例）
//   - 评论被 AI 驳回：无需失效（pending/rejected 从未在公开页展示）
//   - 留言板留言被 AI 放行/驳回：InvalidatePage("/author/" + 当前作者用户名)
//     （留言板公开页在作者主页；设计文档 §8 规则表）
func applyInvalidations(inv Invalidator, authorNameFn func() string, r *moderationapp.Result) {
	if inv == nil || r == nil {
		return
	}
	authorPath := "/author/" + authorName(authorNameFn)
	for _, it := range r.ApprovedItems {
		switch it.Kind {
		case moderationapp.KindComment:
			if it.MomentID > 0 {
				_ = inv.DataChange("/moments")
			} else {
				inv.InvalidatePage("/posts")
				inv.InvalidatePage("/")
				if it.PostID > 0 {
					_ = inv.DataChange("/posts/:id", strconv.FormatInt(it.PostID, 10))
				}
			}
		case moderationapp.KindGuestbook:
			inv.InvalidatePage(authorPath)
		}
	}
	for _, it := range r.RejectedItems {
		if it.Kind == moderationapp.KindGuestbook {
			inv.InvalidatePage(authorPath)
		}
	}
}

// authorName 现取作者用户名（nil 兜底空串，避免 panic）。
func authorName(fn func() string) string {
	if fn == nil {
		return ""
	}
	return fn()
}
