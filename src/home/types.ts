/** 首页数据类型（与 Go 侧 build/interfaces home.go 的 JSON 同形） */

import type { Post } from "../posts/types";

/** 首页动态短列表项 */
export interface HomeMoment {
    id: string;
    content: string;
    createdAt: string;
}

/** 站点统计 */
export interface HomeStats {
    posts: number;
    words: number;
    /** 运营天数（最早文章起算） */
    days: number;
    /** 最新文章发布日期（YYYY-MM-DD，无文章为空串） */
    latestPost: string;
}

/** 作者维护的项目 */
export interface HomeProject {
    name: string;
    desc: string;
    url: string;
}

/** 作者收藏的句子 */
export interface HomeQuote {
    text: string;
    source: string;
}

/** 文章时间线条目 */
export interface HomeTimelineItem {
    id: string;
    title: string;
    createdAt: string;
}

/** hero 作者卡 */
export interface HomeAuthor {
    username: string;
    bio: string;
    avatarUrl: string;
    github: string;
}

/** 首页 initialState */
export interface HomeState {
    recentPosts: Post[];
    recentMoments: HomeMoment[];
    stats: HomeStats;
    projects: HomeProject[];
    quotes: HomeQuote[];
    timeline: HomeTimelineItem[];
    author: HomeAuthor;
}
