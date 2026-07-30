/** 文章详情页（ISR 静态页）；编辑/删除入口仅 author 在客户端挂载后可见 */

import type { PageAppProps } from "../../app/pageApp";
import { navigate } from "../../app/router";
import { Layout } from "../../lib/layout";
import { formatDateTime } from "../../lib/format";
import { useRole } from "../../lib/role";
import { v } from "../../lib/theme";
import type { PostState } from "../types";

export default function PostDetailPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { post: null }) as PostState;
    const post = state.post;
    const role = useRole();

    if (!post) {
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
            <article>
                <h1 style={{ fontSize: 30, marginBottom: 12 }}>{post.title}</h1>
                <div
                    style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 10,
                        fontSize: 13,
                        color: v.textMuted,
                        paddingBottom: 20,
                        marginBottom: 24,
                        borderBottom: `1px solid ${v.border}`,
                    }}
                >
                    <span
                        style={{
                            width: 28,
                            height: 28,
                            borderRadius: "50%",
                            background: v.bgInset,
                            display: "inline-flex",
                            alignItems: "center",
                            justifyContent: "center",
                            fontSize: 13,
                            fontWeight: 650,
                            color: v.textSecondary,
                        }}
                    >
                        {post.authorName.slice(0, 1).toUpperCase()}
                    </span>
                    <span style={{ color: v.textSecondary, fontWeight: 550 }}>{post.authorName}</span>
                    <span>发布于 {formatDateTime(post.createdAt)}</span>
                    {post.updatedAt !== post.createdAt && <span>（更新于 {formatDateTime(post.updatedAt)}）</span>}
                    {post.tags.map((t) => (
                        <span key={t} className="ven-chip">
                            {t}
                        </span>
                    ))}
                </div>
                <div style={{ whiteSpace: "pre-wrap", fontSize: 15, lineHeight: 1.8 }}>{post.content}</div>
            </article>
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
        </Layout>
    );
}
