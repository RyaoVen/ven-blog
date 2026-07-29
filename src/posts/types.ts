/** 文章相关共享类型（与 Go 侧 build/interfaces PostView 的 JSON 同形） */

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

/** 列表类页面的 initialState */
export interface PostListState {
    posts: Post[];
}

/** 详情/编辑类页面的 initialState */
export interface PostState {
    post: Post | null;
}
