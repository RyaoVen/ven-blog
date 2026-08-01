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

/** 设置页 initialState */
export interface AdminSettingsState {
    content: SettingsContent;
    moderation: boolean;
    categories: string[];
    profile: SettingsProfile;
}
