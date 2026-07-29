/** 发文/编辑页（仅 author 可访问，由框架守卫）；带 ?id= 时为编辑模式，表单回填被编辑文章 */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import type { PostState } from "../posts/types";

const styles = {
    form: { display: "flex", flexDirection: "column", gap: 12 },
    input: {
        padding: "8px 12px",
        fontSize: 16,
        border: "1px solid #d0d7de",
        borderRadius: 6,
    },
    textarea: {
        padding: "8px 12px",
        fontSize: 14,
        border: "1px solid #d0d7de",
        borderRadius: 6,
        fontFamily: "inherit",
        resize: "vertical",
    },
    submit: {
        padding: "8px 20px",
        fontSize: 14,
        color: "#fff",
        background: "#1f883d",
        border: "none",
        borderRadius: 6,
        cursor: "pointer",
    },
    cancel: { color: "#57606a", textDecoration: "none", fontSize: 14, alignSelf: "center" },
    error: { color: "#cf222e", fontSize: 14, margin: 0 },
} as const;

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
            <h1>{editing ? "编辑文章" : "写文章"}</h1>
            <form style={styles.form} onSubmit={onSubmit}>
                <input
                    style={styles.input}
                    placeholder="标题"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    required
                />
                <textarea
                    style={styles.textarea}
                    placeholder="正文（纯文本）"
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    rows={16}
                    required
                />
                {error && <p style={styles.error}>{error}</p>}
                <div style={{ display: "flex", gap: 12 }}>
                    <button style={styles.submit} type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : editing ? "保存" : "发布"}
                    </button>
                    <a style={styles.cancel} href={editing ? `/posts/${editing.id}` : "/posts"}>
                        取消
                    </a>
                </div>
            </form>
        </Layout>
    );
}
