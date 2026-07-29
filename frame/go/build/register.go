// Package build 业务层：角色、页面与 API 的注册入口。
// 在这里编写业务注册（框架不附带 demo）；hybrid API 用法见根目录 README.md 与 PROMPT.md。
package build

import "ven_hybird/hybrid"

// Register 注册业务角色、页面与 API。
// 页面 pattern 必须与 src/**/page.tsx 推导出的路由一致，否则启动即失败。
func Register(a *hybrid.App) error {
	if err := registerRoles(a); err != nil {
		return err
	}
	store := newBlogStore()
	registerAuth(a, store)
	if err := registerPostPages(a, store); err != nil {
		return err
	}
	return registerPostAPIs(a, store)
}

// registerRoles 注册博客角色（须在页面注册前完成）。
// 阅读是公开行为（页面 roles 为 nil），reader/author 之间无需继承，扁平注册即可：
// 框架的页面鉴权会把声明角色连带父级等级一起命中，若让 author 继承 reader，
// 声明 "author" 的页面/API 会连带放行 reader，表达不了"仅 author"。
func registerRoles(a *hybrid.App) error {
	if err := a.RegisterRole("reader", nil); err != nil {
		return err
	}
	return a.RegisterRole("author", nil)
}
