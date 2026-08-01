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
    createdAt: string;
}

/** 数据面板 initialState */
export interface AdminDashboardState {
    stats: AdminStats;
    recentComments: AdminComment[];
}

/** 文章管理 initialState */
export interface AdminPostsState {
    posts: Post[];
}

/** 评论管理 initialState */
export interface AdminCommentsState {
    comments: AdminComment[];
    pending: AdminComment[];
}

/** 动态管理 initialState */
export interface AdminMomentsState {
    moments: Moment[];
}
