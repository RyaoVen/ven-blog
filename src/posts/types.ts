/** 文章相关共享类型（与 Go 侧 build/interfaces 的 JSON 同形） */

/** 一篇博客文章 */
export interface Post {
    id: string;
    title: string;
    category: string;
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

/** /posts 列表页（分类筛选 + 分页）的 initialState */
export interface PagedPostsState {
    posts: Post[];
    total: number;
    page: number;
    pageSize: number;
    /** 当前筛选分类（空串表示全部） */
    category: string;
    /** 设置中的分类列表（筛选框用） */
    categories: string[];
}

/** 编辑类页面的 initialState（categories 为编辑器分类下拉的列表） */
export interface PostState {
    post: Post | null;
    categories: string[];
}

/** 详情页的 initialState（公开数据；viewer 个性化状态走 /api 互动接口） */
export interface PostDetailState {
    post: Post | null;
    likeCount: number;
    favoriteCount: number;
    comments: Comment[];
}
