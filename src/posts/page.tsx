/** 文章列表页（动态页：左侧分类框标签筛选 + 分页；数据变更由框架缓存失效 + SSE 推送刷新，页面零接入） */

import type { PageAppProps } from "../app/pageApp";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";
import { PostList } from "./list";
import type { PagedPostsState } from "./types";

const fallback: PagedPostsState = { posts: [], total: 0, page: 1, pageSize: 10, tag: "", tags: [] };

/** 标签筛选链接（"全部"不带 query） */
function tagHref(tag: string): string {
    return tag ? `/posts?tag=${encodeURIComponent(tag)}` : "/posts";
}

/** 分页链接：保留 tag 参数，第 1 页省略 page */
function pageHref(tag: string, page: number): string {
    const params = new URLSearchParams();
    if (tag) {
        params.set("tag", tag);
    }
    if (page > 1) {
        params.set("page", String(page));
    }
    const qs = params.toString();
    return qs ? `/posts?${qs}` : "/posts";
}

export default function PostsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? fallback) as PagedPostsState;
    const totalPages = Math.max(1, Math.ceil(state.total / state.pageSize));
    return (
        <Layout>
            <div className="ven-posts-grid" style={{ display: "grid", gridTemplateColumns: "200px 1fr", gap: 36, alignItems: "start" }}>
                <aside className="ven-card ven-tagbox" style={{ padding: "16px 14px", position: "sticky", top: 84 }}>
                    <p className="ven-meta" style={{ margin: "0 0 10px" }}>
                        分类
                    </p>
                    <nav style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                        <TagRow label="全部" href="/posts" active={state.tag === ""} />
                        {state.tags.map((t) => (
                            <TagRow key={t} label={t} href={tagHref(t)} active={state.tag === t} />
                        ))}
                    </nav>
                </aside>
                <div>
                    <header style={{ marginBottom: 24 }}>
                        <h1 style={{ fontSize: 28 }}>文章</h1>
                        <p style={{ color: v.textSecondary, margin: 0 }}>
                            {state.tag ? `标签「${state.tag}」 · 共 ${state.total} 篇` : `共 ${state.total} 篇`}
                        </p>
                    </header>
                    <PostList posts={state.posts} />
                    {totalPages > 1 && (
                        <nav style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: 28 }}>
                            {state.page > 1 ? (
                                <a className="ven-btn" href={pageHref(state.tag, state.page - 1)}>
                                    ← 上一页
                                </a>
                            ) : (
                                <span />
                            )}
                            <span className="ven-meta">
                                {state.page} / {totalPages}
                            </span>
                            {state.page < totalPages ? (
                                <a className="ven-btn" href={pageHref(state.tag, state.page + 1)}>
                                    下一页 →
                                </a>
                            ) : (
                                <span />
                            )}
                        </nav>
                    )}
                </div>
            </div>
        </Layout>
    );
}

/** 分类框行（激活态玉青加粗） */
function TagRow({ label, href, active }: { label: string; href: string; active: boolean }) {
    return (
        <a
            href={href}
            className="ven-accent-item"
            style={{
                display: "block",
                padding: "6px 0 6px 12px",
                fontSize: 14,
                textDecoration: "none",
                color: active ? v.accent : v.textSecondary,
                fontWeight: active ? 650 : 400,
            }}
        >
            {label}
        </a>
    );
}
