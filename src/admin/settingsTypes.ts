/** 后台-设置页共享类型（与 Go 侧 build/interfaces settings.go 的 JSON 同形） */

/** 技能标签 */
export interface SettingsSkill {
    name: string;
    level: string;
}

/** 友链 */
export interface SettingsFriend {
    name: string;
    url: string;
    desc: string;
}

/** 收藏的句子 */
export interface SettingsQuote {
    text: string;
    source: string;
}

/** 项目 */
export interface SettingsProject {
    name: string;
    desc: string;
    url: string;
}

/** 站点内容配置 */
export interface SettingsContent {
    paragraphs: string[];
    skills: SettingsSkill[];
    friends: SettingsFriend[];
    quotes: SettingsQuote[];
    projects: SettingsProject[];
    github: string;
}

/** 当前作者资料 */
export interface SettingsProfile {
    username: string;
    bio: string;
    avatarUrl: string;
}

/** 邮箱配置（SMTP + 作者个人邮箱） */
export interface EmailConfig {
    host: string;
    port: string;
    user: string;
    fromName: string;
    passwordSet: boolean;
    authorEmail: string;
}

/** LLM 配置（AI 审核；apiKey 不下发，只给 keySet） */
export interface LLMConfig {
    baseUrl: string;
    model: string;
    keySet: boolean;
}

/** 设置页 initialState */
export interface AdminSettingsState {
    content: SettingsContent;
    moderation: boolean;
    /** 评论总开关（一键关闭全站评论区；false 时读者侧隐藏评论区、接口拒绝新评论） */
    commentsEnabled: boolean;
    aiModeration: boolean;
    authEnabled: boolean;
    categories: string[];
    profile: SettingsProfile;
    email: EmailConfig;
    llm: LLMConfig;
    siteIcon: string;
    /** 站点公网地址（设置键原始值；未设置时空串——RSS/邮件链接回退 env/默认） */
    siteUrl: string;
}

/** API 访问密钥（与 Go 侧 apikeyapp.KeyView JSON 同形；永不含明文） */
export interface ApiKeyView {
    id: string;
    name: string;
    prefix: string;
    createdAt: string;
    lastUsedAt: string | null;
    revokedAt: string | null;
}
