/** 文章相关共享类型（与 Go 侧 build/interfaces 的 JSON 同形） */

/** 一篇博客文章 */
export interface Post {
    id: string;
    title: string;
    summary: string;
    content: string;
    coverUrl: string;
    authorName: string;
    tags: string[];
    createdAt: string;
    updatedAt: string;
}

/** 一条评论 */
export interface Comment {
    id: string;
    userId: string;
    username: string;
    content: string;
    createdAt: string;
}

/** 列表类页面的 initialState */
export interface PostListState {
    posts: Post[];
}

/** 编辑类页面的 initialState */
export interface PostState {
    post: Post | null;
}

/** 详情页的 initialState（公开数据；viewer 个性化状态走 /api 互动接口） */
export interface PostDetailState {
    post: Post | null;
    likeCount: number;
    favoriteCount: number;
    comments: Comment[];
}
