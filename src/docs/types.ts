/** docs 插件前端类型（与 Go docView/TreeNode 契约对齐，unit-7 §6） */

/** 文档节点视图（Go docView） */
export interface DocView {
    id: string;
    parentId: string;
    slug: string;
    path: string;
    kind: "doc" | "section";
    title: string;
    summary: string;
    content?: string;
    tags: string[];
    sortOrder: number;
    status: "draft" | "published";
    createdAt: string;
    updatedAt: string;
}

/** 导航树节点（Go TreeNode） */
export interface DocsTreeNode {
    doc: {
        id: string;
        path: string;
        slug: string;
        kind: "doc" | "section";
        title: string;
        summary: string;
        sortOrder: number;
        status: "draft" | "published";
        tags: string[];
        updatedAt: string;
    };
    children: DocsTreeNode[];
}

/** 上一页/下一页链接 */
export interface DocLink {
    path: string;
    title: string;
}

/** 书架条目（Go BookShelfItem） */
export interface BookShelfItem {
    path: string;
    slug: string;
    title: string;
    summary: string;
    kind: "doc" | "section";
    tags: string[];
    chapters: number;
    updatedAt: string;
}

/** 文档首页（书架）initialState（Go /docs handler） */
export interface DocsHomeState {
    mode: "bookshelf";
    books: BookShelfItem[];
    tree: DocsTreeNode[];
}

/** 歌单式头部统计（Go book handler stats） */
export interface BookStats {
    chapters: number;
    totalChars: number;
    lastUpdated: string;
}

/** 文档页 initialState（Go 文档页 handler，mode=book|chapter） */
export interface DocPageState {
    mode: "book" | "chapter";
    /** 所属顶层书 */
    book: DocView | null;
    /** 书内章节目录（侧栏数据源） */
    chapters: DocView[];
    doc: DocView | null;
    /** 当前节点的直接子节点（深层小节兜底） */
    children: DocView[];
    prev: DocLink | null;
    next: DocLink | null;
    /** 书页统计（歌单式头部） */
    stats?: BookStats;
}
