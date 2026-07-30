/** 发文/编辑页（仅 author 可访问，由框架守卫）；带 ?id= 时为编辑模式，表单回填被编辑文章 */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";
import type { PostState } from "../posts/types";

export default function WritePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { post: null }) as PostState;
    const editing = state.post;
    const [title, setTitle] = useState(editing?.title ?? "");
    const [content, setContent] = useState(editing?.content ?? "");
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch(editing ? `/api/posts/${editing.id}` : "/api/posts", {
                method: editing ? "PUT" : "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ title, content }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error ?? "保存失败");
                return;
            }
            navigate(`/posts/${editing ? editing.id : data.id}`);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <Layout>
            <h1 style={{ fontSize: 24, marginBottom: 20 }}>{editing ? "编辑文章" : "写文章"}</h1>
            <form style={{ display: "flex", flexDirection: "column", gap: 14 }} onSubmit={onSubmit}>
                <input
                    className="ven-input"
                    style={{ fontSize: 16, fontWeight: 550 }}
                    placeholder="标题"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    required
                />
                <textarea
                    className="ven-input"
                    placeholder="正文（Markdown 支持即将上线）"
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    rows={16}
                    required
                />
                {error && <p style={{ color: v.danger, fontSize: 14, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : editing ? "保存" : "发布"}
                    </button>
                    <a style={{ fontSize: 14, color: v.textSecondary }} href={editing ? `/posts/${editing.id}` : "/posts"}>
                        取消
                    </a>
                </div>
            </form>
        </Layout>
    );
}
