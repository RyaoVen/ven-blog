package docs

import (
	"strings"

	"ven_hybird/build/plugin"
	"ven_hybird/hybrid"
)

// InvalidateFunc 写操作后失效 /docs 静态页（三条固定深度模式全局失效 + SSE 联动）。
type InvalidateFunc func()

// newInvalidate 构造失效回调：DataChange 仅在接口层调用（治理规则）；
// params 留空 = 该模式全局失效（Phase 1 简化，子树精确失效随 M5 Move 一并评估）。
func newInvalidate(rt *plugin.Runtime) InvalidateFunc {
	patterns := []string{"/docs/:a", "/docs/:a/:b", "/docs/:a/:b/:c"}
	return func() {
		if rt.DataChange == nil {
			return
		}
		for _, pattern := range patterns {
			// StaticPage 未声明（页面注册失败路径）时 DataChange 报错——忽略，缓存由 LRU 兜底。
			_ = rt.DataChange(pattern)
		}
	}
}

// registerPages 注册前端页面（unit-7 §6）。
// 路由契约（上游需求 7 未合入前的固定深度过渡方案，树深上限 3）：
//
//	/docs            首页（动态页：目录树视图）
//	/docs/:a         一段文档/目录
//	/docs/:a/:b      两段
//	/docs/:a/:b/:c   三段
//
// pattern 与 src/docs/**/page.tsx 推导严格一致（不一致启动即失败）。
func registerPages(rt *plugin.Runtime, svc *Service) error {
	app := rt.App
	// 首页：公开动态页，initialState 为 published 导航树。
	if err := app.Page("/docs", nil, func(c *hybrid.PageCtx) error {
		tree, err := svc.Tree(true)
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"tree": tree})
	}); err != nil {
		return err
	}

	// 文档页：三条固定深度 ISR 声明共享同一 handler。
	docHandler := func(c *hybrid.PageCtx) error {
		parts := make([]string, 0, 3)
		for _, key := range []string{"a", "b", "c"} {
			if v := c.Param(key); v != "" {
				parts = append(parts, v)
			}
		}
		res, err := svc.Get(strings.Join(parts, "/"))
		if err != nil {
			return c.NotFound()
		}
		if res.Doc.Status != StatusPublished {
			// draft 不对公开页面曝光（admin 编辑走 #9 的 API 视图）。
			return c.NotFound()
		}
		tree, err := svc.Tree(true)
		if err != nil {
			return err
		}
		published, err := svc.publishedPaths()
		if err != nil {
			return err
		}
		prev, next := neighbors(published, res.Doc.Path)
		kids := make([]docView, 0, len(res.Children))
		for _, ch := range res.Children {
			if ch.Status == StatusPublished {
				kids = append(kids, toView(ch, false))
			}
		}
		return c.JSON(map[string]any{
			"doc":      toView(res.Doc, true),
			"children": kids,
			"tree":     tree,
			"prev":     prev,
			"next":     next,
		})
	}
	for _, pattern := range []string{"/docs/:a", "/docs/:a/:b", "/docs/:a/:b/:c"} {
		// maxPages=2000、smartLoad 开启：全局更新按热度预重渲染，超出上限懒回源。
		if err := app.StaticPage(pattern, 2000, true, nil, docHandler); err != nil {
			return err
		}
	}
	return nil
}

// neighbors 返回扁平发布序（DFS：sort_order → slug）中 path 的前驱/后继链接（path+title）。
func neighbors(flat []*Doc, path string) (prev, next *linkItem) {
	for i, d := range flat {
		if d.Path != path {
			continue
		}
		if i > 0 {
			prev = &linkItem{Path: flat[i-1].Path, Title: flat[i-1].Title}
		}
		if i < len(flat)-1 {
			next = &linkItem{Path: flat[i+1].Path, Title: flat[i+1].Title}
		}
		break
	}
	return prev, next
}

// linkItem 上一页/下一页链接（initialState.prev/next）。
type linkItem struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}
