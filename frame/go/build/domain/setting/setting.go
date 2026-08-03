// Package setting 站点设置聚合：键值存储与配置键定义。
package setting

// 配置键。
const (
	KeyCategories        = "categories"          // 文章分类（行列表）
	KeyCommentModeration = "comment_moderation"  // 评论审核开关（"on" 为开）
	KeyCommentsEnabled   = "comments_enabled"    // 评论总开关（"on" 为开；未设置视为开）
	KeyIntroParagraphs   = "intro_paragraphs"    // 作者介绍段落（行列表）
	KeySkills            = "skills"              // 技能标签（行：name|level）
	KeyFriendLinks       = "friend_links"        // 友链（行：name|url|desc）
	KeyQuotes            = "quotes"              // 收藏的句子（行：text|source）
	KeyProjects          = "projects"            // 维护的项目（行：name|desc|url）
	KeyShowcasePosts     = "showcase_posts"      // 展示柜文章 ID（行列表，有序，最多 4 个）
	KeySMTPHost          = "smtp_host"           // SMTP 主机
	KeySMTPPort          = "smtp_port"           // SMTP 端口（465/587/25）
	KeySMTPUser          = "smtp_user"           // SMTP 账号（即发件地址）
	KeySMTPPass          = "smtp_pass"           // SMTP 密码/授权码（接口不回传）
	KeySMTPFromName      = "smtp_from_name"      // 发件人名称
	KeyAuthorEmail       = "author_email"        // 作者个人邮箱（@ 通知收件 + author 账号绑定）
	KeySiteIcon          = "site_icon"           // 站点图标 URL（favicon + 导航品牌标）
	KeySiteURL           = "site_url"            // 站点公网地址（RSS/邮件链接拼接；空回退 env/默认）
	KeyModeratorReported = "moderator_reported"  // 审核 worker 已报告条目 ID（行：kind:id，邮件去重）
	KeyLLMBaseURL        = "llm_base_url"        // LLM OpenAI 兼容端点（空回退 env/默认）
	KeyLLMAPIKey         = "llm_api_key"         // LLM API key（接口不回传；空回退 env）
	KeyLLMModel          = "llm_model"           // LLM 模型名（空回退 env/默认）
	KeyUGCModeration     = "ugc_ai_moderation"   // AI 自动审核开关（"off" 为关；未设置视为开）
	KeyUserAuthEnabled   = "user_auth_enabled"   // 用户注册登录开关（默认 "on"；关闭后公开注册/邮箱验证码登录入口 403，作者账号登录保留）
)

// Repository 设置仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Get 读取键值，不存在返回空串与 nil 错误（调用方回退默认值）。
	Get(key string) (string, error)
	// Set 写入键值（upsert）。
	Set(key, value string) error
}
