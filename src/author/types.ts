/** 作者主页数据类型（与 Go 侧 build/interfaces profiles.go /author/:name 的 JSON 同形） */

import type { Post } from "../posts/types";

/** 技术栈标签（level：deep 深入 / solid 熟练 / know 了解） */
export interface AuthorSkill {
    name: string;
    level: string;
}

/** 个人介绍配置 */
export interface AuthorIntro {
    paragraphs: string[];
    skills: AuthorSkill[];
}

/** 展示柜项目卡 */
export interface ShowcaseProject {
    name: string;
    desc: string;
    url: string;
}

/** 展示柜数据 */
export interface Showcase {
    projects: ShowcaseProject[];
    articles: Post[];
}

/** 友链卡片 */
export interface FriendLink {
    name: string;
    url: string;
    desc: string;
}

/** 一条留言 */
export interface GuestbookEntry {
    id: string;
    userId: string;
    username: string;
    content: string;
    status?: string;
    createdAt: string;
}

/** 作者主页 initialState */
export interface AuthorHomeState {
    author: {
        username: string;
        role: string;
        bio: string;
        avatarUrl: string;
        createdAt: string;
    };
    intro: AuthorIntro;
    showcase: Showcase;
    friendLinks: FriendLink[];
    guestbook: GuestbookEntry[];
}
