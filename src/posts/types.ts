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
    /** 回复目标用户名（@ 形式平铺展示，空串表示非回复） */
    replyTo: string;
    /** 审核状态（approved | pending；pending 仅发表者提交瞬间本地展示） */
    status: string;
    createdAt: string;
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
