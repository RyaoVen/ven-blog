// 置顶管理接口：文章与动态置顶开关（全部 author 守卫，框架自动 /api 前缀）。
// 文章置顶走 declarePostsChanged（列表/首页缓存 + 单篇详情失效 + SSE）；
// 动态置顶走 DataChange("/moments")（ISR 静态页再生 + SSE）。
package interfaces

import (
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/hybrid"
)

// pinInput 置顶请求体。
type pinInput struct {
	Pinned bool `json:"pinned"`
}

// RegisterPins 注册置顶 API：POST /api/admin/posts/:id/pin 与 POST /api/admin/moments/:id/pin。
func RegisterPins(a *hybrid.App, posts *postapp.Service, moments *momentapp.Service) error {
	admin := []string{"author"}

	if err := a.Post("/admin/posts/:id/pin", admin, func(c *hybrid.ApiCtx) error {
		var in pinInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		id := mustID(c.Param("id"))
		if err := posts.SetPinned(id, in.Pinned); err != nil {
			return writePostError(c, err)
		}
		declarePostsChanged(a, id)
		return c.JSON(200, map[string]any{"ok": true, "pinned": in.Pinned})
	}); err != nil {
		return err
	}

	return a.Post("/admin/moments/:id/pin", admin, func(c *hybrid.ApiCtx) error {
		var in pinInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		id := mustID(c.Param("id"))
		if err := moments.SetPinned(id, in.Pinned); err != nil {
			return writeMomentError(c, err)
		}
		_ = a.DataChange("/moments")
		return c.JSON(200, map[string]any{"ok": true, "pinned": in.Pinned})
	})
}
