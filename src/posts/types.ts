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

/** /posts 列表页（标签筛选 + 分页）的 initialState */
export interface PagedPostsState {
    posts: Post[];
    total: number;
    page: number;
    pageSize: number;
    /** 当前筛选标签（空串表示全部） */
    tag: string;
    /** 全量标签（筛选条用） */
    tags: string[];
}

/** 详情/编辑类页面的 initialState */
export interface PostState {
    post: Post | null;
}
