/** 后台-评论管理：待审核区（通过/驳回/删除）+ 被驳回区（原因 + 恢复）+ 全站评论列表 */

import { useMemo, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { formatDateTime } from "../../lib/format";
import { CheckIcon, TrashIcon } from "../../lib/icons";
import { ConfirmModal, Modal } from "../../lib/modal";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import { FilterSelect, SearchBar } from "../searchBar";
import type { AdminComment, AdminCommentsState } from "../types";

export default function AdminCommentsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { comments: [], pending: [], rejected: [] }) as AdminCommentsState;
    const [list, setList] = useState(state.comments);
    const [pending, setPending] = useState(state.pending);
    const [rejected, setRejected] = useState(state.rejected);
    const [deleting, setDeleting] = useState<AdminComment | null>(null);
    const [rejecting, setRejecting] = useState<AdminComment | null>(null);
    const [reason, setReason] = useState("");
    const [rejectingErr, setRejectingErr] = useState<string | null>(null);
    const [keyword, setKeyword] = useState("");
    const [status, setStatus] = useState("");

    const filtered = useMemo(
        () =>
            list.filter(
                (c) =>
                    (!keyword ||
                        c.content.toLowerCase().includes(keyword.toLowerCase()) ||
                        c.username.toLowerCase().includes(keyword.toLowerCase())) &&
                    (!status || c.status === status),
            ),
        [list, keyword, status],
    );

    // 全量列表内同步某条评论（approve/reject/recover 后状态迁移）
    function syncInList(c: AdminComment) {
        setList((l) => l.map((x) => (x.id === c.id ? { ...x, status: c.status, rejectedReason: c.rejectedReason } : x)));
    }

    async function approve(c: AdminComment) {
        const resp = await fetch(`/api/comments/${c.id}/approve`, { method: "POST" });
        if (resp.ok) {
            const updated = { ...c, status: "approved", rejectedReason: "" };
            syncInList(updated);
            setPending((l) => l.filter((x) => x.id !== c.id));
            setRejected((l) => l.filter((x) => x.id !== c.id));
        }
    }

    async function doReject() {
        if (!rejecting) {
            return;
        }
        const resp = await fetch(`/api/comments/${rejecting.id}/reject`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ reason }),
        });
        const data = await resp.json().catch(() => null);
        if (!resp.ok) {
            setRejectingErr(data?.error ?? "驳回失败，请重试");
            return;
        }
        const updated = { ...rejecting, status: "rejected", rejectedReason: reason };
        syncInList(updated);
        setPending((l) => l.filter((x) => x.id !== rejecting.id));
        setRejected((l) => (l.some((x) => x.id === rejecting.id) ? l.map((x) => (x.id === rejecting.id ? updated : x)) : [updated, ...l]));
        setRejecting(null);
        setReason("");
        setRejectingErr(null);
    }

    function openReject(c: AdminComment) {
        setRejecting(c);
        setReason(c.rejectedReason ?? "");
        setRejectingErr(null);
    }

    async function recover(c: AdminComment) {
        const resp = await fetch(`/api/comments/${c.id}/recover`, { method: "POST" });
        if (resp.ok) {
            const updated = { ...c, status: "approved", rejectedReason: "" };
            syncInList(updated);
            setRejected((l) => l.filter((x) => x.id !== c.id));
            setPending((l) => l.filter((x) => x.id !== c.id));
        }
    }

    async function confirmDelete() {
        if (!deleting) {
            return;
        }
        const resp = await fetch(`/api/comments/${deleting.id}`, { method: "DELETE" });
        if (resp.ok) {
            setList((l) => l.filter((c) => c.id !== deleting.id));
            setPending((l) => l.filter((c) => c.id !== deleting.id));
            setRejected((l) => l.filter((c) => c.id !== deleting.id));
        }
        setDeleting(null);
    }

    function commentMeta(c: AdminComment) {
        return (
            <div style={{ display: "flex", gap: 10, alignItems: "baseline", flexWrap: "wrap" }}>
                <span style={{ fontWeight: 650, fontSize: 14 }}>{c.username}</span>
                <span style={{ color: v.textMuted, fontSize: 13 }}>评论了</span>
                <a href={`/posts/${c.postId}`} style={{ fontSize: 13 }}>
                    {c.postTitle || `#${c.postId}`}
                </a>
                <span className="ven-meta">{formatDateTime(c.createdAt)}</span>
                {c.status === "pending" && <span className="ven-chip">待审核</span>}
                {c.status === "rejected" && (
                    <span className="ven-chip" style={{ color: v.danger, borderColor: v.danger }}>
                        已驳回
                    </span>
                )}
            </div>
        );
    }

    return (
        <AdminLayout route={bootstrap.route}>
            {pending.length > 0 && (
                <section className="ven-card" style={{ padding: "18px 20px", marginBottom: 28, borderLeft: `3px solid ${v.accent}` }}>
                    <h2 style={{ fontSize: 16, margin: "0 0 12px" }}>待审核（{pending.length}）</h2>
                    <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                        {pending.map((c) => (
                            <li key={c.id} className="ven-admin-row" style={{ display: "flex", justifyContent: "space-between", gap: "10px 16px", flexWrap: "wrap", padding: "10px 0", borderBottom: `1px solid ${v.border}` }}>
                                <div style={{ flex: 1, minWidth: 0 }}>
                                    {commentMeta(c)}
                                    <p style={{ margin: "4px 0 0", fontSize: 13.5, color: v.textSecondary }}>{c.content}</p>
                                </div>
                                <div style={{ display: "flex", gap: 8, flexShrink: 0, alignItems: "flex-start" }}>
                                    <button type="button" className="ven-btn ven-btn-primary" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => approve(c)}>
                                        <CheckIcon size={12} />
                                        通过
                                    </button>
                                    <button type="button" className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => openReject(c)}>
                                        驳回
                                    </button>
                                    <button type="button" className="ven-btn ven-btn-danger" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => setDeleting(c)}>
                                        删除
                                    </button>
                                </div>
                            </li>
                        ))}
                    </ul>
                </section>
            )}
            {rejected.length > 0 && (
                <section className="ven-card" style={{ padding: "18px 20px", marginBottom: 28, borderLeft: `3px solid ${v.danger}` }}>
                    <h2 style={{ fontSize: 16, margin: "0 0 12px" }}>已驳回（{rejected.length}）</h2>
                    <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                        {rejected.map((c) => (
                            <li key={c.id} className="ven-admin-row" style={{ display: "flex", justifyContent: "space-between", gap: "10px 16px", flexWrap: "wrap", padding: "10px 0", borderBottom: `1px solid ${v.border}` }}>
                                <div style={{ flex: 1, minWidth: 0 }}>
                                    {commentMeta(c)}
                                    <p style={{ margin: "4px 0 0", fontSize: 13.5, color: v.textSecondary }}>{c.content}</p>
                                    <p style={{ margin: "4px 0 0", fontSize: 12.5, color: v.danger }}>
                                        驳回原因：{c.rejectedReason || "（未填写）"}
                                    </p>
                                </div>
                                <div style={{ display: "flex", gap: 8, flexShrink: 0, alignItems: "flex-start" }}>
                                    <button type="button" className="ven-btn ven-btn-primary" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => recover(c)}>
                                        恢复
                                    </button>
                                    <button type="button" className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => openReject(c)}>
                                        改判
                                    </button>
                                    <button type="button" className="ven-btn ven-btn-danger" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => setDeleting(c)}>
                                        删除
                                    </button>
                                </div>
                            </li>
                        ))}
                    </ul>
                </section>
            )}
            <SearchBar keyword={keyword} onKeyword={setKeyword} placeholder="搜索内容 / 用户名…">
                <FilterSelect
                    value={status}
                    onChange={setStatus}
                    options={[
                        { value: "", label: "全部状态" },
                        { value: "approved", label: "已通过" },
                        { value: "pending", label: "待审核" },
                        { value: "rejected", label: "已驳回" },
                    ]}
                />
            </SearchBar>
            <p className="ven-meta" style={{ margin: "0 0 20px" }}>
                共 {filtered.length} / {list.length} 条
            </p>
            {filtered.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>没有匹配的评论。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {filtered.map((c) => (
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
                                {commentMeta(c)}
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
                                {c.status === "rejected" && c.rejectedReason && (
                                    <p style={{ margin: "4px 0 0", fontSize: 12.5, color: v.danger }}>驳回原因：{c.rejectedReason}</p>
                                )}
                            </div>
                            <button
                                type="button"
                                className="ven-btn ven-btn-danger"
                                style={{ padding: "3px 12px", fontSize: 12, flexShrink: 0 }}
                                onClick={() => setDeleting(c)}
                            >
                                <TrashIcon size={12} />
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
            <Modal open={rejecting !== null} onClose={() => setRejecting(null)} width={440}>
                <h3 style={{ margin: "0 0 8px", fontSize: 16 }}>驳回评论</h3>
                <p style={{ margin: "0 0 14px", fontSize: 13.5, color: v.textSecondary }}>
                    {rejecting?.username}：{rejecting?.content}
                </p>
                <textarea
                    className="ven-input"
                    rows={3}
                    maxLength={200}
                    placeholder="驳回原因（必填，≤200 字）"
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    autoFocus
                />
                {rejectingErr && <p style={{ color: v.danger, fontSize: 13, margin: "8px 0 0" }}>{rejectingErr}</p>}
                <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 16 }}>
                    <button type="button" className="ven-btn" onClick={() => setRejecting(null)}>
                        取消
                    </button>
                    <button type="button" className="ven-btn ven-btn-danger" onClick={doReject} disabled={!reason.trim()}>
                        驳回
                    </button>
                </div>
            </Modal>
        </AdminLayout>
    );
}
