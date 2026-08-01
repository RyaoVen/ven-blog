/** 用户/作者主页共享类型（与 Go 侧 build/interfaces UserView、ProfileStats 的 JSON 同形） */

import type { Post } from "../posts/types";

/** 用户公开信息 */
export interface ProfileUser {
    username: string;
    role: string;
    bio: string;
    avatarUrl: string;
    createdAt: string;
}

/** 个人页作品统计 */
export interface ProfileStats {
    posts: number;
    comments: number;
}

/** /users/:name 的 initialState（favorites 仅 viewer 为本人时下发） */
export interface UserProfileState {
    user: ProfileUser;
    stats: ProfileStats;
    isAuthor: boolean;
    /** 邮箱（仅 viewer 为本人时下发） */
    email?: string;
    favorites?: Post[];
}

/** /author/:name 的 initialState */
export interface AuthorProfileState {
    author: ProfileUser;
    posts: Post[];
}
