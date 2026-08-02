/** 后台共享类型（与 Go 侧 build/interfaces admin.go 的 JSON 同形） */

import type { Moment } from "../moments/types";
import type { Post } from "../posts/types";

/** 后台统计 */
export interface AdminStats {
    posts: number;
    words: number;
    comments: number;
    likes: number;
    favorites: number;
    users: number;
    moments: number;
    subscribers: number;
}

/** 后台评论视图 */
export interface AdminComment {
    id: string;
    postId: string;
    postTitle: string;
    username: string;
    content: string;
    status: string;
    rejectedReason?: string;
    createdAt: string;
}

/** 用户增长（折线 + 增量） */
export interface UserGrowth {
    d7: { date: string; count: number }[];
    d30: { date: string; count: number }[];
    d365: { date: string; count: number }[];
    deltas: { yesterday: number; week: number; month: number };
}

/** 日历热力图日数据 */
export interface HeatDay {
    date: string;
    posts: number;
    chars: number;
}

/** 分类计数 */
export interface CategoryCount {
    category: string;
    count: number;
}

/** 数据面板 initialState */
export interface AdminDashboardState {
    stats: AdminStats;
    recentComments: AdminComment[];
    userGrowth: UserGrowth;
    heatmap: HeatDay[];
    categoryCounts: CategoryCount[];
}

/** 文章管理 initialState */
export interface AdminPostsState {
    posts: Post[];
}

/** 评论管理 initialState */
export interface AdminCommentsState {
    comments: AdminComment[];
    pending: AdminComment[];
    rejected: AdminComment[];
}

/** 后台留言管理视图（与 Go 侧 adminGuestbookView JSON 同形） */
export interface AdminGuestbookEntry {
    id: string;
    userId: string;
    username: string;
    content: string;
    status: string;
    rejectedReason?: string;
    createdAt: string;
}

/** 留言板管理 initialState */
export interface AdminGuestbookState {
    entries: AdminGuestbookEntry[];
    pending: AdminGuestbookEntry[];
    rejected: AdminGuestbookEntry[];
}

/** 动态管理 initialState */
export interface AdminMomentsState {
    moments: Moment[];
}
