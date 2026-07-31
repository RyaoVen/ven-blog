/** 动态相关共享类型（与 Go 侧 build/interfaces MomentView 的 JSON 同形） */

/** 一条动态 */
export interface Moment {
    id: string;
    content: string;
    authorName: string;
    createdAt: string;
}

/** 动态页的 initialState（commentCounts 为动态 ID → 评论数映射） */
export interface MomentsState {
    moments: Moment[];
    commentCounts: Record<string, number>;
}
