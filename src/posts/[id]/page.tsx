/** 文章详情页（ISR 静态页）：信息头 + Markdown 正文 + TOC 侧栏 + 互动条 + 评论区；编辑/删除仅 author 客户端可见 */

import { useMemo } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { navigate } from "../../app/router";
import { Layout } from "../../lib/layout";
import { formatDateTime } from "../../lib/format";
import { renderMarkdown } from "../../lib/markdown";
import { markdownCss } from "../../lib/markdownCss";
import { useRole } from "../../lib/role";
import { v } from "../../lib/theme";
import { CommentsSection } from "../comments";
import { InteractionBar, usePostViewer } from "../interactions";
import { Toc } from "../toc";
import type { PostDetailState } from "../types";

export default function PostDetailPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { post: null, likeCount: 0, favoriteCount: 0, comments: [] }) as PostDetailState;
    const post = state.post;
    const role = useRole();
    const rendered = useMemo(() => (post ? renderMarkdown(post.content) : null), [post]);
    const { viewer, toggle } = usePostViewer(post?.id ?? "0", state.likeCount, state.favoriteCount);

    if (!post || !rendered) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>文章不存在或已删除。</p>
            </Layout>
        );
    }

    async function onDelete() {
        if (!confirm("确定删除这篇文章吗？")) {
            return;
        }
        const resp = await fetch(`/api/posts/${post!.id}`, { method: "DELETE" });
        if (resp.ok) {
            navigate("/posts");
        }
    }

    return (
        <Layout>
            <style>{markdownCss}</style>
            <h1 style={{ fontSize: 30, marginBottom: 12 }}>{post.title}</h1>
            <div
                className="ven-meta"
                style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 12,
                    flexWrap: "wrap",
                    paddingBottom: 20,
                    marginBottom: 28,
                    borderBottom: `1px solid ${v.border}`,
                }}
            >
                <span
                    style={{
                        width: 28,
                        height: 28,
                        borderRadius: 2,
                        background: v.text,
                        display: "inline-flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: 13,
                        fontWeight: 700,
                        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
                        color: v.bg,
                    }}
                >
                    {post.authorName.slice(0, 1).toUpperCase()}
                </span>
                <a
                    href={"/author/" + post.authorName}
                    style={{ color: v.textSecondary, fontWeight: 550, textDecoration: "none" }}
                >
                    {post.authorName}
                </a>
                <span>发布于 {formatDateTime(post.createdAt)}</span>
                {post.updatedAt !== post.createdAt && <span>（更新于 {formatDateTime(post.updatedAt)}）</span>}
                {post.tags.map((t) => (
                    <span key={t} className="ven-chip">
                        {t}
                    </span>
                ))}
            </div>
            <div className="ven-post-layout">
                <article>
                    <div className="ven-prose" dangerouslySetInnerHTML={{ __html: rendered.html }} />
                    <InteractionBar viewer={viewer} onToggle={toggle} />
                    <CommentsSection postId={post.id} comments={state.comments} viewerUserId={viewer.userId} />
                    {role === "author" && (
                        <div style={{ marginTop: 36, display: "flex", gap: 12 }}>
                            <a href={`/write?id=${post.id}`} className="ven-btn">
                                编辑
                            </a>
                            <button type="button" onClick={onDelete} className="ven-btn ven-btn-danger">
                                删除
                            </button>
                        </div>
                    )}
                </article>
                {rendered.toc.length >= 2 && <Toc items={rendered.toc} />}
            </div>
        </Layout>
    );
}
