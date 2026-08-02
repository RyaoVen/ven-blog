/** 首页数据类型（与 Go 侧 build/interfaces home.go 的 JSON 同形） */

import type { Post } from "../posts/types";

/** 首页动态短列表项 */
export interface HomeMoment {
    id: string;
    content: string;
    /** 置顶标记（列表排序置顶优先，前端只做展示） */
    pinned: boolean;
    createdAt: string;
}

/** 站点统计 */
export interface HomeStats {
    posts: number;
    words: number;
    /** 运营天数（最早文章起算，SSR 展示用） */
    days: number;
    /** 运营起点（最早文章时间，RFC3339；客户端滚动计时用） */
    launchAt: string;
    /** 最新文章 ID（卡片跳转用） */
    latestID: string;
    /** 最新文章"几天前"文案（服务端计算，防 hydration 时钟偏差） */
    latestAgo: string;
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
