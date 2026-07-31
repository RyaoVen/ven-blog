/** 后台-评论管理：全站评论列表 + 删除 */

import { useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { formatDateTime } from "../../lib/format";
import { ConfirmModal } from "../../lib/modal";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import type { AdminComment, AdminCommentsState } from "../types";

export default function AdminCommentsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { comments: [] }) as AdminCommentsState;
    const [list, setList] = useState(state.comments);
    const [deleting, setDeleting] = useState<AdminComment | null>(null);

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
        <AdminLayout route={bootstrap.route}>
            <p className="ven-meta" style={{ margin: "0 0 20px" }}>
                共 {list.length} 条
            </p>
            {list.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>还没有评论。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {list.map((c) => (
                        <li
                            key={c.id}
                            style={{
                                display: "flex",
                                justifyContent: "space-between",
                                gap: 16,
                                padding: "12px 0",
                                borderBottom: `1px solid ${v.border}`,
                            }}
                        >
                            <div style={{ flex: 1, minWidth: 0 }}>
                                <div style={{ display: "flex", gap: 10, alignItems: "baseline", flexWrap: "wrap" }}>
                                    <span style={{ fontWeight: 650, fontSize: 14 }}>{c.username}</span>
                                    <span style={{ color: v.textMuted, fontSize: 13 }}>评论了</span>
                                    <a href={`/posts/${c.postId}`} style={{ fontSize: 13 }}>
                                        {c.postTitle || `#${c.postId}`}
                                    </a>
                                    <span className="ven-meta">{formatDateTime(c.createdAt)}</span>
                                </div>
                                <p
                                    style={{
                                        margin: "4px 0 0",
                                        fontSize: 13.5,
                                        color: v.textSecondary,
                                        overflow: "hidden",
                                        textOverflow: "ellipsis",
                                        whiteSpace: "nowrap",
                                    }}
                                >
                                    {c.content}
                                </p>
                            </div>
                            <button
                                type="button"
                                className="ven-btn ven-btn-danger"
                                style={{ padding: "3px 12px", fontSize: 12, flexShrink: 0 }}
                                onClick={() => setDeleting(c)}
                            >
                                删除
                            </button>
                        </li>
                    ))}
                </ul>
            )}
            <ConfirmModal
                open={deleting !== null}
                title="删除评论"
                message={`确定删除 ${deleting?.username} 的这条评论吗？不可恢复。`}
                confirmText="删除"
                danger
                onCancel={() => setDeleting(null)}
                onConfirm={confirmDelete}
            />
        </AdminLayout>
    );
}
