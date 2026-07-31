/** 评论区：@ 平铺回复 + Markdown 渲染 + 自研确认弹窗删除
 * 列表 props 经 ISR/SSE 刷新，本地状态做即时反馈。
 */

import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { formatDateTime } from "../lib/format";
import { renderMarkdown } from "../lib/markdown";
import { ConfirmModal } from "../lib/modal";
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
    const [replyTo, setReplyTo] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<Comment | null>(null);
    const inputRef = useRef<HTMLTextAreaElement>(null);

    // SSE 推送刷新 initialState 后同步本地列表
    useEffect(() => setList(comments), [comments]);

    function startReply(username: string) {
        setReplyTo(username);
        inputRef.current?.focus();
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch(`/api/posts/${postId}/comments`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ content: draft, replyTo: replyTo ?? "" }),
            });
            if (!resp.ok) {
                setError("评论失败，请重试");
                return;
            }
            const created = (await resp.json()) as Comment;
            setDraft("");
            setReplyTo(null);
            setList((l) => [created, ...l]); // 即时上屏，SSE 推送后整体校准
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    async function confirmDelete() {
        if (!deleting) {
            return;
        }
        const resp = await fetch(`/api/comments/${deleting.id}`, { method: "DELETE" });
        if (resp.ok) {
            setList((l) => l.filter((c) => c.id !== deleting.id));
        }
        setDeleting(null);
    }

    return (
        <section style={{ marginTop: 40 }}>
            <h2 style={{ fontSize: 18 }}>评论{list.length > 0 && `（${list.length}）`}</h2>
            {role ? (
                <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, margin: "16px 0 28px" }}>
                    {replyTo && (
                        <div style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13 }}>
                            <span className="ven-chip">回复 @{replyTo}</span>
                            <button
                                type="button"
                                className="ven-meta"
                                style={{ border: "none", background: "none", cursor: "pointer", padding: 0 }}
                                onClick={() => setReplyTo(null)}
                            >
                                取消
                            </button>
                        </div>
                    )}
                    <textarea
                        ref={inputRef}
                        className="ven-input"
                        rows={3}
                        placeholder="写下你的评论…（支持 Markdown）"
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
                        <CommentItem
                            key={c.id}
                            comment={c}
                            canDelete={viewerUserId === c.userId || role === "author"}
                            canReply={role !== null}
                            onReply={() => startReply(c.username)}
                            onDelete={() => setDeleting(c)}
                        />
                    ))}
                </ul>
            )}
            <ConfirmModal
                open={deleting !== null}
                title="删除评论"
                message={`确定删除${deleting?.username ?? "该"} 的这条评论吗？此操作不可撤销。`}
                confirmText="删除"
                danger
                onCancel={() => setDeleting(null)}
                onConfirm={confirmDelete}
            />
        </section>
    );
}

/** 单条评论（Markdown 渲染 + @ 前缀 + 回复/删除操作） */
function CommentItem({
    comment: c,
    canDelete,
    canReply,
    onReply,
    onDelete,
}: {
    comment: Comment;
    canDelete: boolean;
    canReply: boolean;
    onReply: () => void;
    onDelete: () => void;
}) {
    const rendered = useMemo(() => renderMarkdown(c.content), [c.content]);
    return (
        <li style={{ padding: "14px 0", borderTop: `1px solid ${v.border}` }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
                <div style={{ display: "flex", gap: 10, alignItems: "baseline", flexWrap: "wrap" }}>
                    <span style={{ fontWeight: 650, fontSize: 14 }}>{c.username}</span>
                    {c.replyTo && <span className="ven-chip">@{c.replyTo}</span>}
                    <span className="ven-meta">{formatDateTime(c.createdAt)}</span>
                </div>
                <div style={{ display: "flex", gap: 8 }}>
                    {canReply && (
                        <button
                            type="button"
                            className="ven-meta"
                            style={{ border: "none", background: "none", cursor: "pointer", padding: 0 }}
                            onClick={onReply}
                        >
                            回复
                        </button>
                    )}
                    {canDelete && (
                        <button
                            type="button"
                            className="ven-meta"
                            style={{ border: "none", background: "none", cursor: "pointer", padding: 0, color: v.danger }}
                            onClick={onDelete}
                        >
                            删除
                        </button>
                    )}
                </div>
            </div>
            <div
                className="ven-prose ven-comment-prose"
                style={{ marginTop: 6 }}
                dangerouslySetInnerHTML={{ __html: rendered.html }}
            />
        </li>
    );
}
