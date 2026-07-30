// 搜索页注册：按查询参数 q 检索文章（标题/正文）。
package interfaces

import (
	"strings"

	"ven_hybird/build/application/postapp"
	"ven_hybird/hybrid"
)

// RegisterSearch 注册搜索页（公开动态页；空关键词由应用层归一为空结果）。
func RegisterSearch(a *hybrid.App, posts *postapp.Service) error {
	return a.Page("/search", nil, func(c *hybrid.PageCtx) error {
		q := strings.TrimSpace(c.Query("q"))
		results, err := posts.Search(q)
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"q": q, "results": toPostViews(results)})
	})
}
