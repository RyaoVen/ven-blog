// Package build 业务组装根（composition root）：
// 构造基础设施 → 注入应用服务 → 注册接口层。
// 分层约定（DDD）见根目录 AGENTS.md「业务分层」一节。
package build

import (
	"fmt"
	"os"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/subscribeapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/infrastructure/persistence"
	"ven_hybird/build/interfaces"
	"ven_hybird/hybrid"
)

// Register 注册业务角色、页面与 API。
// 页面 pattern 必须与 src/**/page.tsx 推导出的路由一致，否则启动即失败。
func Register(a *hybrid.App) error {
	if err := registerRoles(a); err != nil {
		return err
	}

	// 基础设施：MySQL 连接（自动建库建表）与仓储
	db, err := persistence.Open(persistence.DSNFromEnv())
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	userRepo := persistence.NewUserRepository(db)
	postRepo := persistence.NewPostRepository(db)
	commentRepo := persistence.NewCommentRepository(db)
	interactionRepo := persistence.NewInteractionRepository(db)
	momentRepo := persistence.NewMomentRepository(db)
	imageRepo := persistence.NewImageRepository(db)
	subscriberRepo := persistence.NewSubscriberRepository(db)
	if err := persistence.SeedUsers(userRepo); err != nil {
		return fmt.Errorf("build: seed users: %w", err)
	}

	// 首页 hero 作者卡需要作者资料（种子 author 即本站作者）
	author, err := userRepo.FindByUsername(persistence.AuthorUsernameFromEnv())
	if err != nil {
		return fmt.Errorf("build: find author: %w", err)
	}

	// 应用服务
	posts := postapp.NewService(postRepo)
	users := userapp.NewService(userRepo)
	comments := commentapp.NewService(commentRepo)
	interactions := interactionapp.NewService(interactionRepo)
	moments := momentapp.NewService(momentRepo)
	subscribe := subscribeapp.NewService(subscriberRepo)

	// 接口层注册（发文归属经 c.User() 取调用者，框架会话已携带用户身份）
	interfaces.RegisterAuth(a, users)
	interfaces.RegisterImages(a, imageRepo)
	if err := interfaces.RegisterHome(a, posts, moments, author); err != nil {
		return err
	}
	if err := interfaces.RegisterSiteInfo(a, author); err != nil {
		return err
	}
	if err := interfaces.RegisterSubscribe(a, subscribe, posts, siteURLFromEnv()); err != nil {
		return err
	}
	if err := interfaces.RegisterPages(a, posts, comments, interactions); err != nil {
		return err
	}
	if err := interfaces.RegisterInteractions(a, comments, interactions); err != nil {
		return err
	}
	if err := interfaces.RegisterSearch(a, posts); err != nil {
		return err
	}
	if err := interfaces.RegisterProfiles(a, users, posts); err != nil {
		return err
	}
	if err := interfaces.RegisterAPIs(a, posts); err != nil {
		return err
	}
	if err := interfaces.RegisterMomentComments(a, comments); err != nil {
		return err
	}
	if err := interfaces.RegisterMe(a); err != nil {
		return err
	}
	if err := interfaces.RegisterMoments(a, moments, comments); err != nil {
		return err
	}
	return interfaces.RegisterAdmin(a, posts, comments, interactions, moments, subscribe, users)
}

// siteURLFromEnv 返回站点对外 URL（BLOG_SITE_URL，RSS 链接拼接用；默认本地开发地址）。
func siteURLFromEnv() string {
	if u := os.Getenv("BLOG_SITE_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:8080"
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
