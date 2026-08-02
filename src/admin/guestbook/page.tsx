/** 后台-留言管理：待审核区（通过/驳回/删除）+ 被驳回区（原因 + 恢复/删除）+ 全量列表（搜索 + 状态筛选） */

import { useMemo, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { formatDateTime } from "../../lib/format";
import { CheckIcon, TrashIcon } from "../../lib/icons";
import { ConfirmModal, Modal } from "../../lib/modal";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import { FilterSelect, SearchBar } from "../searchBar";
import type { AdminGuestbookEntry, AdminGuestbookState } from "../types";

export default function AdminGuestbookPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { entries: [], pending: [], rejected: [] }) as AdminGuestbookState;
    const [list, setList] = useState(state.entries);
    const [pending, setPending] = useState(state.pending);
    const [rejected, setRejected] = useState(state.rejected);
    const [deleting, setDeleting] = useState<AdminGuestbookEntry | null>(null);
    const [rejecting, setRejecting] = useState<AdminGuestbookEntry | null>(null);
    const [reason, setReason] = useState("");
    const [rejectingErr, setRejectingErr] = useState<string | null>(null);
    const [keyword, setKeyword] = useState("");
    const [status, setStatus] = useState("");

    const filtered = useMemo(
        () =>
            list.filter(
                (e) =>
                    (!keyword ||
                        e.content.toLowerCase().includes(keyword.toLowerCase()) ||
                        e.username.toLowerCase().includes(keyword.toLowerCase())) &&
                    (!status || e.status === status),
            ),
        [list, keyword, status],
    );

    // 全量列表内同步某条留言（approve/reject/recover 后状态迁移）
    function syncInList(e: AdminGuestbookEntry) {
        setList((l) => l.map((x) => (x.id === e.id ? { ...x, status: e.status, rejectedReason: e.rejectedReason } : x)));
    }

    async function approve(e: AdminGuestbookEntry) {
        const resp = await fetch(`/api/guestbook/${e.id}/approve`, { method: "POST" });
        if (resp.ok) {
            const updated = { ...e, status: "approved", rejectedReason: "" };
            syncInList(updated);
            setPending((l) => l.filter((x) => x.id !== e.id));
            setRejected((l) => l.filter((x) => x.id !== e.id));
        }
    }

    async function doReject() {
        if (!rejecting) {
            return;
        }
        const resp = await fetch(`/api/guestbook/${rejecting.id}/reject`, {
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

    function openReject(e: AdminGuestbookEntry) {
        setRejecting(e);
        setReason(e.rejectedReason ?? "");
        setRejectingErr(null);
    }

    async function recover(e: AdminGuestbookEntry) {
        const resp = await fetch(`/api/guestbook/${e.id}/recover`, { method: "POST" });
        if (resp.ok) {
            const updated = { ...e, status: "approved", rejectedReason: "" };
            syncInList(updated);
            setRejected((l) => l.filter((x) => x.id !== e.id));
            setPending((l) => l.filter((x) => x.id !== e.id));
        }
    }

    async function confirmDelete() {
        if (!deleting) {
            return;
        }
        const resp = await fetch(`/api/guestbook/${deleting.id}`, { method: "DELETE" });
        if (resp.ok) {
            setList((l) => l.filter((e) => e.id !== deleting.id));
            setPending((l) => l.filter((e) => e.id !== deleting.id));
            setRejected((l) => l.filter((e) => e.id !== deleting.id));
        }
        setDeleting(null);
    }

    function entryMeta(e: AdminGuestbookEntry) {
        return (
            <div style={{ display: "flex", gap: 10, alignItems: "baseline", flexWrap: "wrap" }}>
                <span style={{ fontWeight: 650, fontSize: 14 }}>{e.username}</span>
                <span className="ven-meta">{formatDateTime(e.createdAt)}</span>
                {e.status === "pending" && <span className="ven-chip">待审核</span>}
                {e.status === "rejected" && (
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
                        {pending.map((e) => (
                            <li key={e.id} style={{ display: "flex", justifyContent: "space-between", gap: 16, padding: "10px 0", borderBottom: `1px solid ${v.border}` }}>
                                <div style={{ flex: 1, minWidth: 0 }}>
                                    {entryMeta(e)}
                                    <p style={{ margin: "4px 0 0", fontSize: 13.5, color: v.textSecondary }}>{e.content}</p>
                                </div>
                                <div style={{ display: "flex", gap: 8, flexShrink: 0, alignItems: "flex-start" }}>
                                    <button type="button" className="ven-btn ven-btn-primary" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => approve(e)}>
                                        <CheckIcon size={12} />
                                        通过
                                    </button>
                                    <button type="button" className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => openReject(e)}>
                                        驳回
                                    </button>
                                    <button type="button" className="ven-btn ven-btn-danger" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => setDeleting(e)}>
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
                        {rejected.map((e) => (
                            <li key={e.id} style={{ display: "flex", justifyContent: "space-between", gap: 16, padding: "10px 0", borderBottom: `1px solid ${v.border}` }}>
                                <div style={{ flex: 1, minWidth: 0 }}>
                                    {entryMeta(e)}
                                    <p style={{ margin: "4px 0 0", fontSize: 13.5, color: v.textSecondary }}>{e.content}</p>
                                    <p style={{ margin: "4px 0 0", fontSize: 12.5, color: v.danger }}>
                                        驳回原因：{e.rejectedReason || "（未填写）"}
                                    </p>
                                </div>
                                <div style={{ display: "flex", gap: 8, flexShrink: 0, alignItems: "flex-start" }}>
                                    <button type="button" className="ven-btn ven-btn-primary" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => recover(e)}>
                                        恢复
                                    </button>
                                    <button type="button" className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => openReject(e)}>
                                        改判
                                    </button>
                                    <button type="button" className="ven-btn ven-btn-danger" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => setDeleting(e)}>
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
                <p style={{ color: v.textMuted, fontSize: 14 }}>没有匹配的留言。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {filtered.map((e) => (
                        <li
                            key={e.id}
                            style={{
                                display: "flex",
                                justifyContent: "space-between",
                                gap: 16,
                                padding: "12px 0",
                                borderBottom: `1px solid ${v.border}`,
                            }}
                        >
                            <div style={{ flex: 1, minWidth: 0 }}>
                                {entryMeta(e)}
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
                                    {e.content}
                                </p>
                                {e.status === "rejected" && e.rejectedReason && (
                                    <p style={{ margin: "4px 0 0", fontSize: 12.5, color: v.danger }}>驳回原因：{e.rejectedReason}</p>
                                )}
                            </div>
                            <button
                                type="button"
                                className="ven-btn ven-btn-danger"
                                style={{ padding: "3px 12px", fontSize: 12, flexShrink: 0 }}
                                onClick={() => setDeleting(e)}
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
                title="删除留言"
                message={`确定删除 ${deleting?.username} 的这条留言吗？不可恢复。`}
                confirmText="删除"
                danger
                onCancel={() => setDeleting(null)}
                onConfirm={confirmDelete}
            />
            <Modal open={rejecting !== null} onClose={() => setRejecting(null)} width={440}>
                <h3 style={{ margin: "0 0 8px", fontSize: 16 }}>驳回留言</h3>
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
