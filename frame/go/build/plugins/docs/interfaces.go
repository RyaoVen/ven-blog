package docs

import (
	"errors"
	"fmt"
	"strconv"
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

// registerAdminAPI 注册后台管理 API（cookie 会话 + author 角色）。
// 列表数据由管理页面 /admin/docs 的 initialState 直接灌入（posts 管理页先例），
// data-only 取数走同一 handler，无需独立 GET API——避免与页面路由撞车。
func registerAdminAPI(rt *plugin.Runtime, svc *Service, invalidate InvalidateFunc) error {
	app := rt.App
	admin := []string{"author"}
	// 新建（path 全路径，隐式补父与 MCP 同语义）。
	if err := app.Post("/admin/docs", admin, func(c *hybrid.ApiCtx) error {
		var in struct {
			Title   string   `json:"title"`
			Path    string   `json:"path"`
			Kind    Kind     `json:"kind"`
			Content *string  `json:"content"`
			Summary *string  `json:"summary"`
			Tags    []string `json:"tags"`
			Order   *int     `json:"order"`
			Status  Status   `json:"status"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "invalid payload")
		}
		doc, err := svc.Create(CreateInput{
			Title: in.Title, Path: in.Path, Kind: in.Kind, Content: in.Content,
			Summary: in.Summary, Tags: in.Tags, Order: in.Order, Status: in.Status,
		})
		if err != nil {
			return adminErr(c, err)
		}
		invalidate()
		return c.JSON(200, map[string]any{"doc": toView(doc, true)})
	}); err != nil {
		return err
	}
	// 按 ID 部分更新。
	if err := app.Put("/admin/docs/:id", admin, func(c *hybrid.ApiCtx) error {
		id, err := parseID(c.Param("id"))
		if err != nil {
			return c.Error(400, "invalid id")
		}
		var in struct {
			Title   *string  `json:"title"`
			Content *string  `json:"content"`
			Summary *string  `json:"summary"`
			Tags    []string `json:"tags"`
			Order   *int     `json:"order"`
			Status  *Status  `json:"status"`
		}
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "invalid payload")
		}
		doc, err := svc.UpdateByID(id, UpdateInput{
			Title: in.Title, Content: in.Content, Summary: in.Summary,
			Tags: in.Tags, Order: in.Order, Status: in.Status,
		})
		if err != nil {
			return adminErr(c, err)
		}
		invalidate()
		return c.JSON(200, map[string]any{"doc": toView(doc, true)})
	}); err != nil {
		return err
	}
	// 删除（?recursive=true 级联）。
	return app.Delete("/admin/docs/:id", admin, func(c *hybrid.ApiCtx) error {
		id, err := parseID(c.Param("id"))
		if err != nil {
			return c.Error(400, "invalid id")
		}
		if err := svc.DeleteByID(id, c.Query("recursive") == "true"); err != nil {
			return adminErr(c, err)
		}
		invalidate()
		return c.JSON(200, map[string]any{"deleted": true})
	})
}

// adminErr 用例错误 → HTTP 状态（admin 接口用消息可读格式）。
func adminErr(c *hybrid.ApiCtx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return c.Error(404, "文档不存在")
	case errors.Is(err, ErrDuplicatePath):
		return c.Error(400, "路径已存在")
	case errors.Is(err, ErrHasChildren):
		return c.Error(400, "存在子节点，删除需 recursive")
	default:
		return c.Error(400, err.Error())
	}
}

// registerAdminPages 注册后台页面（与 src/admin/docs/** 推导路由严格一致）。
func registerAdminPages(rt *plugin.Runtime, svc *Service) error {
	app := rt.App
	admin := []string{"author"}
	allDocs := func() ([]docView, error) {
		docs, _, err := svc.List(ListInput{Recursive: true, Limit: 10000})
		if err != nil {
			return nil, err
		}
		views := make([]docView, 0, len(docs))
		for _, d := range docs {
			views = append(views, toView(d, false))
		}
		return views, nil
	}
	// 管理列表页。
	if err := app.Page("/admin/docs", admin, func(c *hybrid.PageCtx) error {
		views, err := allDocs()
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{"docs": views})
	}); err != nil {
		return err
	}
	// 新建页（空数据页：c.JSON(nil)，pages.go 惯例）。
	if err := app.Page("/admin/docs/new", admin, func(c *hybrid.PageCtx) error {
		return c.JSON(nil)
	}); err != nil {
		return err
	}
	// 编辑页。
	return app.Page("/admin/docs/:id/edit", admin, func(c *hybrid.PageCtx) error {
		id, err := parseID(c.Param("id"))
		if err != nil {
			return c.JSON(map[string]any{"doc": nil})
		}
		doc, err := svc.GetByID(id)
		if err != nil {
			return c.JSON(map[string]any{"doc": nil})
		}
		return c.JSON(map[string]any{"doc": toView(doc, true)})
	})
}

// parseID 解析路径参数 ID。
func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("bad id")
	}
	return id, nil
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
