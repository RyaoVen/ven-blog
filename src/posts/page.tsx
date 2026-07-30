/** 文章列表页（动态页：标签筛选 + 分页；数据变更由框架缓存失效 + SSE 推送刷新，页面零接入） */

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

const chipLink = { textDecoration: "none" } as const;

/** 激活态标签：黑底反色 */
const chipActive = { background: v.text, borderColor: v.text, color: v.bg } as const;

export default function PostsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? fallback) as PagedPostsState;
    const totalPages = Math.max(1, Math.ceil(state.total / state.pageSize));
    return (
        <Layout>
            <header style={{ marginBottom: 24 }}>
                <h1 style={{ fontSize: 28 }}>文章</h1>
                <p style={{ color: v.textSecondary, margin: 0 }}>
                    {state.tag ? `标签「${state.tag}」 · 共 ${state.total} 篇` : `共 ${state.total} 篇`}
                </p>
            </header>
            {state.tags.length > 0 && (
                <nav style={{ display: "flex", flexWrap: "wrap", gap: 8, marginBottom: 24 }}>
                    <a href="/posts" className="ven-chip" style={{ ...chipLink, ...(state.tag === "" ? chipActive : null) }}>
                        全部
                    </a>
                    {state.tags.map((t) => (
                        <a
                            key={t}
                            href={tagHref(t)}
                            className="ven-chip"
                            style={{ ...chipLink, ...(state.tag === t ? chipActive : null) }}
                        >
                            {t}
                        </a>
                    ))}
                </nav>
            )}
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
        </Layout>
    );
}
