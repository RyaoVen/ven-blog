// Package build 业务组装根（composition root）：
// 构造基础设施 → 注入应用服务 → 注册接口层。
// 分层约定（DDD）见根目录 AGENTS.md「业务分层」一节。
package build

import (
	"fmt"
	"os"
	"time"

	"ven_hybird/build/domain/user"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/apikeyapp"
	"ven_hybird/build/application/emailauth"
	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/application/interactionapp"
	"ven_hybird/build/application/moderationapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/ratelimit"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/subscribeapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/application/visitapp"
	"ven_hybird/build/infrastructure/llm"
	"ven_hybird/build/infrastructure/mailer"
	"ven_hybird/build/infrastructure/persistence"
	"ven_hybird/build/interfaces"
	"ven_hybird/build/interfaces/moderator"
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
	apiKeyRepo := persistence.NewApiKeyRepository(db)
	visitRepo := persistence.NewVisitRepository(db)
	if err := persistence.SeedUsers(userRepo); err != nil {
		return fmt.Errorf("build: seed users: %w", err)
	}

	// 作者资料每次请求现取（设置页改头像/简介/用户名后立即生效；按角色定位，不受改名影响）
	authorFn := func() (*user.User, error) {
		return userRepo.FindByRole(user.RoleAuthor)
	}
	if _, err := authorFn(); err != nil {
		return fmt.Errorf("build: find author: %w", err)
	}
	// 需要作者主页路径的失效场景现取当前用户名（改名后旧路径不再有效）
	authorNameFn := func() string {
		u, err := authorFn()
		if err != nil {
			return ""
		}
		return u.Username
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
	// 审核开关（comment_moderation 设置）同时管评论与留言：开时新留言待审核
	guestbook := guestbookapp.NewService(guestbookRepo, func() bool {
		on, err := settings.Moderation()
		return err == nil && on
	})
	apiKeys := apikeyapp.NewService(apiKeyRepo)
	visits := visitapp.NewService(visitRepo)

	// 埋点 ① Go 网关中间件：最外层 Use（ISR 直发也计数），panic 由框架兜底——
	// 页面响应不受埋点影响；SPA data-only 取数已在中间件层跳过，由前端导航上报兜底。
	a.SetVisitRecorder(func(path string) {
		_ = visits.Record(time.Now(), path)
	})

	// 接口层注册（发文归属经 c.User() 取调用者，框架会话已携带用户身份）
	// 认证限速器（内存实现，进程内生效；参数可调）：
	// 登录失败 5 次（用户名+IP 维度）锁 15 分钟；发码每邮箱 1 次/分钟、每 IP 每日 50 次。
	loginLimiter := ratelimit.New(loginFailThreshold, loginLockWindow)
	codeEmailLimiter := ratelimit.New(1, time.Minute)
	codeIPLimiter := ratelimit.New(emailCodePerIPPerDay, 24*time.Hour)
	interfaces.RegisterAuth(a, users, settings, loginLimiter)
	interfaces.RegisterImages(a, imageRepo)
	if err := interfaces.RegisterHome(a, posts, moments, authorFn, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterSiteInfo(a, authorFn, settings, func() string { return siteURLOf(settings) }); err != nil {
		return err
	}
	if err := interfaces.RegisterSubscribe(a, subscribe, posts, siteURLOf(settings)); err != nil {
		return err
	}
	if err := interfaces.RegisterPages(a, posts, comments, interactions, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterInteractions(a, comments, interactions, emailAuth, users, settings, siteURLOf(settings)); err != nil {
		return err
	}
	if err := interfaces.RegisterSearch(a, posts); err != nil {
		return err
	}
	if err := interfaces.RegisterProfiles(a, users, posts, guestbook, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterGuestbookAPI(a, guestbook, authorNameFn); err != nil {
		return err
	}
	if err := interfaces.RegisterGuestbookAdmin(a, guestbook, authorNameFn); err != nil {
		return err
	}
	// 订阅通知器：新文章发布 → 异步邮件通知全部订阅者（goroutine 内发信，不阻塞发布响应；
	// siteURL 每次通知现取——设置页改站点地址即时生效；SMTP 未配置时 mailer 降级日志输出）
	newPostNotify := interfaces.NewPostNotifier(subscribe.Subscribers, mail, func() string { return siteURLOf(settings) })
	if err := interfaces.RegisterAPIs(a, posts, newPostNotify, authorNameFn); err != nil {
		return err
	}
	if err := interfaces.RegisterSettings(a, settings, users); err != nil {
		return err
	}
	if err := interfaces.RegisterCategories(a, posts, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterAuthorAdmin(a, settings, posts, authorNameFn); err != nil {
		return err
	}
	interfaces.RegisterEmailAuth(a, emailAuth, users, settings, siteURLOf(settings), codeEmailLimiter, codeIPLimiter)
	if err := interfaces.RegisterMeEmail(a, users); err != nil {
		return err
	}
	if err := interfaces.RegisterMomentComments(a, comments, emailAuth, users, settings, siteURLOf(settings)); err != nil {
		return err
	}
	if err := interfaces.RegisterMomentLikes(a, interactions); err != nil {
		return err
	}
	// 埋点 ② SPA 导航上报接口（公开；30s 同 path 节流在服务内）
	if err := interfaces.RegisterVisitAPI(a, visits); err != nil {
		return err
	}
	if err := interfaces.RegisterMe(a, users); err != nil {
		return err
	}
	if err := interfaces.RegisterLinkPreview(a); err != nil {
		return err
	}
	if err := interfaces.RegisterMoments(a, moments, comments, interactions, settings); err != nil {
		return err
	}
	if err := interfaces.RegisterPins(a, posts, moments); err != nil {
		return err
	}
	if err := interfaces.RegisterAdmin(a, posts, comments, interactions, moments, subscribe, users, settings, visits); err != nil {
		return err
	}
	if err := interfaces.RegisterKeysAdmin(a, apiKeys); err != nil {
		return err
	}
	// /api/mcp 网关（agent 统一入口）：纯原生 fiber 路由，只认 key 不认 cookie，
	// 与页面注册顺序无关，放链尾最稳；apiKeys 天然满足 interfaces.KeyAuthenticator。
	if err := interfaces.RegisterMCP(a, apiKeys, posts, moments, comments, settings, users, authorFn, authorNameFn); err != nil {
		return err
	}
	// Unit 4：AI 自动审核 worker（BLOG_LLM_API_KEY 未配置则不启动）
	return registerModerator(a, comments, guestbook, settings, mail, authorNameFn)
}

// registerModerator 组装自动审核 worker：构造 llm 客户端 → moderationapp → handler → 启动 ticker。
// LLM 配置 settings 键优先、env（BLOG_LLM_*）兜底，每次判定现取（设置页改动即时生效）；
// worker 常启动，每 tick 现查 AI 开关与 API key 是否就绪，未就绪则停手。
func registerModerator(a *hybrid.App, comments *commentapp.Service, gb *guestbookapp.Service,
	settings *settingsapp.Service, mail mailer.Mailer, authorNameFn func() string) error {
	llmConfigFn := func() (llm.Config, error) {
		baseURL, apiKey, model, err := settings.LLMConfig()
		if err != nil {
			return llm.Config{}, err
		}
		if baseURL == "" {
			baseURL = os.Getenv("BLOG_LLM_BASE_URL")
		}
		if apiKey == "" {
			apiKey = os.Getenv("BLOG_LLM_API_KEY")
		}
		if model == "" {
			model = os.Getenv("BLOG_LLM_MODEL")
		}
		return llm.Config{BaseURL: baseURL, APIKey: apiKey, Model: model}, nil
	}
	llmClient := llm.NewClient(llmConfigFn)
	svc := moderationapp.NewService(comments, gb, llmClient)
	handler := moderator.NewHandler(svc, settings, mail, a, authorNameFn, siteURLOf(settings), moderator.Options{
		Interval: moderator.IntervalFromEnv(), // BLOG_MODERATOR_INTERVAL，默认 5m
		Batch:    moderator.BatchFromEnv(),    // BLOG_MODERATOR_BATCH，默认 20
		Enabled: func() bool {
			on, err := settings.AIModeration()
			if err != nil || !on {
				return false
			}
			cfg, err := llmConfigFn()
			return err == nil && cfg.APIKey != ""
		},
	})
	handler.Start() // 内部 go 协程，不阻塞注册与启动
	return nil
}

// defaultSiteURL 本地开发默认地址（设置键与 env 均未配置时兜底）。
const defaultSiteURL = "http://127.0.0.1:8080"

// 认证限速参数（内存实现；可调）：
// loginFailThreshold 登录失败阈值（用户名+IP 维度），loginLockWindow 锁定窗口——失败计数在窗口内累计，
// 达到阈值后窗口内后续登录 429，窗口过期自动解锁；emailCodePerIPPerDay 每 IP 每日发码上限。
const (
	loginFailThreshold   = 5
	loginLockWindow      = 15 * time.Minute
	emailCodePerIPPerDay = 50
)

// siteURLOf 返回站点对外 URL（RSS/邮件链接拼接用）：
// 设置键 site_url 优先，env BLOG_SITE_URL 兜底，最后回退本地开发默认地址。
func siteURLOf(settings *settingsapp.Service) string {
	if u, err := settings.SiteURL(); err == nil && u != "" {
		return u
	}
	if u := os.Getenv("BLOG_SITE_URL"); u != "" {
		return u
	}
	return defaultSiteURL
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
