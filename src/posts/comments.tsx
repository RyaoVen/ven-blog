/** 评论区（文章/动态通用）：@ 平铺回复 + Markdown 渲染 + 自研确认弹窗删除。
 * 传 initialComments 时以 ISR/SSE 数据为准做即时反馈（文章）；
 * 不传则挂载后向 `${targetPath}/comments` 拉取（动态弹窗等场景）。 */

import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { formatDateTime } from "../lib/format";
import { MessageIcon, TrashIcon } from "../lib/icons";
import { renderMarkdown } from "../lib/markdown";
import { ConfirmModal } from "../lib/modal";
import { useRole } from "../lib/role";
import { v } from "../lib/theme";
import { useViewer } from "../lib/viewer";
import type { Comment } from "./types";

export function CommentsSection({
    targetPath,
    initialComments,
}: {
    targetPath: string;
    initialComments?: Comment[];
}) {
    const role = useRole();
    const viewer = useViewer();
    const [list, setList] = useState<Comment[]>(initialComments ?? []);
    const [draft, setDraft] = useState("");
    const [replyTo, setReplyTo] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<Comment | null>(null);
    const [recentId, setRecentId] = useState<string | null>(null);
    const inputRef = useRef<HTMLTextAreaElement>(null);

    // 文章：SSE 推送刷新 initialState 后同步本地列表
    useEffect(() => {
        if (initialComments !== undefined) {
            setList(initialComments);
        }
    }, [initialComments]);

    // 动态等场景：挂载后拉取评论
    useEffect(() => {
        if (initialComments !== undefined) {
            return;
        }
        let cancelled = false;
        fetch(`/api${targetPath}/comments`)
            .then((r) => (r.ok ? r.json() : null))
            .then((data) => {
                if (!cancelled && data) {
                    setList(data.comments ?? []);
                }
            })
            .catch(() => {});
        return () => {
            cancelled = true;
        };
    }, [targetPath, initialComments]);

    function startReply(username: string) {
        setReplyTo(username);
        inputRef.current?.focus();
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch(`/api${targetPath}/comments`, {
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
            setRecentId(created.id);
            setList((l) => [created, ...l]); // 即时上屏（pending 本地可见，公开列表待审核后出现）
            if (created.status === "pending") {
                setError("评论已提交，待作者审核通过后公开显示");
            }
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
        <section style={{ marginTop: 24 }}>
            <h2 style={{ fontSize: 18, display: "flex", alignItems: "center", gap: 8 }}>
                <MessageIcon size={17} />
                评论{list.length > 0 && `（${list.length}）`}
            </h2>
            {role ? (
                <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, margin: "16px 0 28px" }}>
                    {replyTo && (
                        <div style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13 }}>
                            <span className="ven-chip">回复 @{replyTo}</span>
                            <button
                                type="button"
                                className="ven-meta ven-inline-action"
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
                    <a href={`/login?next=${targetPath}`}>登录</a> 后参与评论。
                </p>
            )}
            {list.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>还没有评论，来抢沙发。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {list.map((c) => (
                        <CommentItem
                            key={c.id}
                            isNew={c.id === recentId}
                            comment={c}
                            canDelete={viewer?.userId === c.userId || role === "author"}
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
    isNew,
    canDelete,
    canReply,
    onReply,
    onDelete,
}: {
    comment: Comment;
    isNew?: boolean;
    canDelete: boolean;
    canReply: boolean;
    onReply: () => void;
    onDelete: () => void;
}) {
    const rendered = useMemo(() => renderMarkdown(c.content), [c.content]);
    return (
        <li className={isNew ? "ven-comment-enter" : undefined} style={{ padding: "14px 0", borderTop: `1px solid ${v.border}` }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
                <div style={{ display: "flex", gap: 10, alignItems: "baseline", flexWrap: "wrap" }}>
                    <span style={{ fontWeight: 650, fontSize: 14 }}>{c.username}</span>
                    {c.replyTo && <span className="ven-chip">@{c.replyTo}</span>}
                    {c.status === "pending" && (
                        <span className="ven-chip" style={{ color: v.accent, borderColor: v.accent }}>
                            待审核
                        </span>
                    )}
                    <span className="ven-meta">{formatDateTime(c.createdAt)}</span>
                </div>
                <div style={{ display: "flex", gap: 8 }}>
                    {canReply && (
                        <button
                            type="button"
                            className="ven-meta ven-inline-action"
                            style={{ border: "none", background: "none", cursor: "pointer", padding: 0 }}
                            onClick={onReply}
                        >
                            回复
                        </button>
                    )}
                    {canDelete && (
                        <button
                            type="button"
                            className="ven-meta ven-inline-action"
                            style={{ border: "none", background: "none", cursor: "pointer", padding: 0, color: v.danger, display: "inline-flex", alignItems: "center", gap: 4 }}
                            onClick={onDelete}
                        >
                            <TrashIcon size={12} />
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
