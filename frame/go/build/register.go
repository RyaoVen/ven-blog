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
	if err := registerPostPages(a, store); err != nil {
		return err
	}
	return registerPostAPIs(a, store)
}

// registerRoles 注册博客角色：author 继承 reader（须在页面注册前完成）。
func registerRoles(a *hybrid.App) error {
	if err := a.RegisterRole("reader", nil); err != nil {
		return err
	}
	return a.RegisterRole("author", []string{"reader"})
}
