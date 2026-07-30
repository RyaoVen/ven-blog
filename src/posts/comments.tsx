/** 评论区：列表（props 经 ISR/SSE 刷新，本地状态做即时反馈）+ 发表框 + 删除（本人或 author） */

import { FormEvent, useEffect, useState } from "react";
import { formatDateTime } from "../lib/format";
import { useRole } from "../lib/role";
import { v } from "../lib/theme";
import type { Comment } from "./types";

export function CommentsSection({
    postId,
    comments,
    viewerUserId,
}: {
    postId: string;
    comments: Comment[];
    viewerUserId: string | null;
}) {
    const role = useRole();
    const [list, setList] = useState(comments);
    const [draft, setDraft] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // SSE 推送刷新 initialState 后同步本地列表
    useEffect(() => setList(comments), [comments]);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch(`/api/posts/${postId}/comments`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ content: draft }),
            });
            if (!resp.ok) {
                setError("评论失败，请重试");
                return;
            }
            const created = (await resp.json()) as Comment;
            setDraft("");
            setList((l) => [created, ...l]); // 即时上屏，SSE 推送后整体校准
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    async function onDelete(id: string) {
        if (!confirm("删除这条评论？")) {
            return;
        }
        const resp = await fetch(`/api/comments/${id}`, { method: "DELETE" });
        if (resp.ok) {
            setList((l) => l.filter((c) => c.id !== id));
        }
    }

    return (
        <section style={{ marginTop: 40 }}>
            <h2 style={{ fontSize: 18 }}>评论{list.length > 0 && `（${list.length}）`}</h2>
            {role ? (
                <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, margin: "16px 0 28px" }}>
                    <textarea
                        className="ven-input"
                        rows={3}
                        placeholder="写下你的评论…"
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        maxLength={2000}
                        required
                    />
                    {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                    <div>
                        <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                            {submitting ? "发表中…" : "发表评论"}
                        </button>
                    </div>
                </form>
            ) : (
                <p style={{ fontSize: 14, color: v.textSecondary }}>
                    <a href={`/login?next=/posts/${postId}`}>登录</a> 后参与评论。
                </p>
            )}
            {list.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>还没有评论，来抢沙发。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {list.map((c) => (
                        <li key={c.id} style={{ padding: "14px 0", borderTop: `1px solid ${v.border}` }}>
                            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
                                <div style={{ display: "flex", gap: 10, alignItems: "baseline" }}>
                                    <span style={{ fontWeight: 650, fontSize: 14 }}>{c.username}</span>
                                    <span className="ven-meta">{formatDateTime(c.createdAt)}</span>
                                </div>
                                {(viewerUserId === c.userId || role === "author") && (
                                    <button
                                        type="button"
                                        className="ven-btn ven-btn-danger"
                                        style={{ padding: "2px 10px", fontSize: 12 }}
                                        onClick={() => onDelete(c.id)}
                                    >
                                        删除
                                    </button>
                                )}
                            </div>
                            <p style={{ margin: "6px 0 0", fontSize: 14, whiteSpace: "pre-wrap" }}>{c.content}</p>
                        </li>
                    ))}
                </ul>
            )}
        </section>
    );
}
