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

/** 文档详情页 initialState（Go 文档页 handler） */
export interface DocPageState {
    doc: DocView | null;
    children: DocView[];
    tree: DocsTreeNode[];
    prev: DocLink | null;
    next: DocLink | null;
}

/** 文档首页 initialState（Go /docs handler） */
export interface DocsHomeState {
    tree: DocsTreeNode[];
}
