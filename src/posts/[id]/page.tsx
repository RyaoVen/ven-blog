/** 文章详情页（ISR 静态页）；编辑/删除入口仅 author 在客户端挂载后可见 */

import type { PageAppProps } from "../../app/pageApp";
import { navigate } from "../../app/router";
import { Layout } from "../../lib/layout";
import { formatDateTime } from "../../lib/format";
import { useRole } from "../../lib/role";
import type { PostState } from "../types";

export default function PostDetailPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { post: null }) as PostState;
    const post = state.post;
    const role = useRole();

    if (!post) {
        return (
            <Layout>
                <p>文章不存在或已删除。</p>
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
                <h1 style={{ marginBottom: 4 }}>{post.title}</h1>
                <div style={{ fontSize: 13, color: "#57606a", marginBottom: 24 }}>
                    发布于 {formatDateTime(post.createdAt)}
                    {post.updatedAt !== post.createdAt && `（更新于 ${formatDateTime(post.updatedAt)}）`}
                </div>
                <div style={{ whiteSpace: "pre-wrap" }}>{post.content}</div>
            </article>
            {role === "author" && (
                <div style={{ marginTop: 32, display: "flex", gap: 12 }}>
                    <a href={`/write?id=${post.id}`} style={{ color: "#0969da", textDecoration: "none" }}>
                        编辑
                    </a>
                    <button
                        type="button"
                        onClick={onDelete}
                        style={{
                            border: "1px solid #d0d7de",
                            borderRadius: 6,
                            background: "#f6f8fa",
                            color: "#cf222e",
                            padding: "4px 10px",
                            fontSize: 14,
                            cursor: "pointer",
                        }}
                    >
                        删除
                    </button>
                </div>
            )}
        </Layout>
    );
}
