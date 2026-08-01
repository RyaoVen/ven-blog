// Package build 业务组装根（composition root）：
// 构造基础设施 → 注入应用服务 → 注册接口层。
// 分层约定（DDD）见根目录 AGENTS.md「业务分层」一节。
package build

import (
	"fmt"
	"os"

	"ven_hybird/build/domain/user"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/emailauth"
	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/subscribeapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/infrastructure/mailer"
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
	guestbookRepo := persistence.NewGuestbookRepository(db)
	settingsRepo := persistence.NewSettingsRepository(db)
	emailCodeRepo := persistence.NewEmailCodeRepository(db)
	if err := persistence.SeedUsers(userRepo); err != nil {
		return fmt.Errorf("build: seed users: %w", err)
	}

	// 作者资料每次请求现取（设置页改头像/简介后立即生效）
	authorFn := func() (*user.User, error) {
		return userRepo.FindByUsername(persistence.AuthorUsernameFromEnv())
	}
	if _, err := authorFn(); err != nil {
		return fmt.Errorf("build: find author: %w", err)
	}

	// 应用服务
	posts := postapp.NewService(postRepo)
	users := userapp.NewService(userRepo)
	settings := settingsapp.NewService(settingsRepo)
	mail := mailer.NewSMTPMailer(func() (mailer.Config, error) {
		host, port, user, password, fromName, err := settings.SMTPConfig()
		return mailer.Config{Host: host, Port: port, User: user, Password: password, FromName: fromName}, err
	})
	emailAuth := emailauth.NewService(emailCodeRepo, userRepo, mail)
	comments := commentapp.NewService(commentRepo, func() bool {
		on, err := settings.Moderation()
		return err == nil && on
	})
	interactions := interactionapp.NewService(interactionRepo)
	moments := momentapp.NewService(momentRepo)
	subscribe := subscribeapp.NewService(subscriberRepo)
	guestbook := guestbookapp.NewService(guestbookRepo)

	// 接口层注册（发文归属经 c.User() 取调用者，框架会话已携带用户身份）
	interfaces.RegisterAuth(a, users)
	interfaces.RegisterImages(a, imageRepo)
	if err := interfaces.RegisterHome(a, posts, moments, authorFn, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterSiteInfo(a, authorFn); err != nil {
		return err
	}
	if err := interfaces.RegisterSubscribe(a, subscribe, posts, siteURLFromEnv()); err != nil {
		return err
	}
	if err := interfaces.RegisterPages(a, posts, comments, interactions); err != nil {
		return err
	}
	if err := interfaces.RegisterInteractions(a, comments, interactions, emailAuth, siteURLFromEnv()); err != nil {
		return err
	}
	if err := interfaces.RegisterSearch(a, posts); err != nil {
		return err
	}
	if err := interfaces.RegisterProfiles(a, users, posts, guestbook, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterGuestbookAPI(a, guestbook, persistence.AuthorUsernameFromEnv()); err != nil {
		return err
	}
	if err := interfaces.RegisterAPIs(a, posts); err != nil {
		return err
	}
	if err := interfaces.RegisterSettings(a, settings, users); err != nil {
		return err
	}
	if err := interfaces.RegisterCategories(a, posts, settings); err != nil {
		return err
	}
	interfaces.RegisterEmailAuth(a, emailAuth, users)
	if err := interfaces.RegisterMeEmail(a, users); err != nil {
		return err
	}
	if err := interfaces.RegisterMomentComments(a, comments, emailAuth, siteURLFromEnv()); err != nil {
		return err
	}
	if err := interfaces.RegisterMomentLikes(a, interactions); err != nil {
		return err
	}
	if err := interfaces.RegisterMe(a); err != nil {
		return err
	}
	if err := interfaces.RegisterMoments(a, moments, comments, interactions); err != nil {
		return err
	}
	return interfaces.RegisterAdmin(a, posts, comments, interactions, moments, subscribe, users, settings)
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
