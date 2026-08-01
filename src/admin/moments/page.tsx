/** 后台-动态管理：发布框（支持插图）+ 历史列表 + 删除 */

import { ChangeEvent, FormEvent, useMemo, useRef, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { formatDateTime } from "../../lib/format";
import { TrashIcon } from "../../lib/icons";
import { ConfirmModal } from "../../lib/modal";
import { v } from "../../lib/theme";
import type { Moment } from "../../moments/types";
import { AdminLayout } from "../adminLayout";
import { SearchBar } from "../searchBar";
import type { AdminMomentsState } from "../types";

export default function AdminMomentsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { moments: [] }) as AdminMomentsState;
    const [list, setList] = useState(state.moments);
    const [draft, setDraft] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<Moment | null>(null);
    const [keyword, setKeyword] = useState("");
    const fileRef = useRef<HTMLInputElement>(null);

    const filtered = useMemo(
        () => list.filter((m) => !keyword || m.content.toLowerCase().includes(keyword.toLowerCase())),
        [list, keyword],
    );
    const draftRef = useRef<HTMLTextAreaElement>(null);

    // insertImage 把图片 Markdown 插到 textarea 光标处（无光标时追加末尾）
    function insertImage(url: string) {
        const snippet = `![](${url})`;
        const ta = draftRef.current;
        if (!ta) {
            setDraft((prev) => prev + snippet);
            return;
        }
        const start = ta.selectionStart;
        const end = ta.selectionEnd;
        setDraft((prev) => prev.slice(0, start) + snippet + prev.slice(end));
    }

    async function onPickImage(event: ChangeEvent<HTMLInputElement>) {
        const file = event.target.files?.[0];
        event.target.value = "";
        if (!file) {
            return;
        }
        setUploading(true);
        setError(null);
        try {
            const form = new FormData();
            form.append("file", file);
            const resp = await fetch("/api/upload", { method: "POST", body: form });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error ?? "图片上传失败");
                return;
            }
            insertImage(data.url);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setUploading(false);
        }
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/moments", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ content: draft }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error ?? "发布失败");
                return;
            }
            setDraft("");
            // 本地即时上屏（详情字段以服务端下次拉取为准）
            setList((l) => [
                { id: data.id, content: draft, authorName: "author", createdAt: new Date().toISOString() },
                ...l,
            ]);
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
        const resp = await fetch(`/api/moments/${deleting.id}`, { method: "DELETE" });
        if (resp.ok) {
            setList((l) => l.filter((m) => m.id !== deleting.id));
        }
        setDeleting(null);
    }

    return (
        <AdminLayout route={bootstrap.route}>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, marginBottom: 32 }}>
                <textarea
                    ref={draftRef}
                    className="ven-input"
                    rows={3}
                    placeholder="写点什么…（1000 字以内，支持 Markdown 与图片）"
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                    maxLength={1000}
                    required
                />
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "发布中…" : "发布动态"}
                    </button>
                    <button className="ven-btn" type="button" disabled={uploading} onClick={() => fileRef.current?.click()}>
                        {uploading ? "上传中…" : "插入图片"}
                    </button>
                    <input
                        ref={fileRef}
                        type="file"
                        accept="image/jpeg,image/png,image/webp,image/gif"
                        style={{ display: "none" }}
                        onChange={onPickImage}
                    />
                </div>
            </form>
            <SearchBar keyword={keyword} onKeyword={setKeyword} placeholder="搜索动态内容…" />
            {filtered.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>没有匹配的动态。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {filtered.map((m) => (
                        <li
                            key={m.id}
                            style={{
                                display: "flex",
                                justifyContent: "space-between",
                                gap: 16,
                                padding: "12px 0",
                                borderBottom: `1px solid ${v.border}`,
                            }}
                        >
                            <div style={{ flex: 1, minWidth: 0 }}>
                                <p
                                    style={{
                                        margin: "0 0 4px",
                                        fontSize: 14,
                                        overflow: "hidden",
                                        textOverflow: "ellipsis",
                                        whiteSpace: "nowrap",
                                    }}
                                >
                                    {m.content}
                                </p>
                                <span className="ven-meta">{formatDateTime(m.createdAt)}</span>
                            </div>
                            <button
                                type="button"
                                className="ven-btn ven-btn-danger"
                                style={{ padding: "3px 12px", fontSize: 12, flexShrink: 0 }}
                                onClick={() => setDeleting(m)}
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
                title="删除动态"
                message="确定删除这条动态吗？不可恢复。"
                confirmText="删除"
                danger
                onCancel={() => setDeleting(null)}
                onConfirm={confirmDelete}
            />
        </AdminLayout>
    );
}
