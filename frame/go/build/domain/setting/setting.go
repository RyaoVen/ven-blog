// Package setting 站点设置聚合：键值存储与配置键定义。
package setting

// 配置键。
const (
	KeyCategories        = "categories"          // 文章分类（行列表）
	KeyCommentModeration = "comment_moderation"  // 评论审核开关（"on" 为开）
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
)

// Repository 设置仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// Get 读取键值，不存在返回空串与 nil 错误（调用方回退默认值）。
	Get(key string) (string, error)
	// Set 写入键值（upsert）。
	Set(key, value string) error
}
