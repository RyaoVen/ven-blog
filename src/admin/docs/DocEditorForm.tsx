/** docs 后台：轻量文档编辑器（新建/编辑共用；正文 textarea——复制适配不强行抽象 posts editor） */

import { FormEvent, useState } from "react";
import { navigate } from "../../app/router";
import { v } from "../../lib/theme";

interface DocDraft {
    title: string;
    path: string;
    kind: "doc" | "section";
    summary: string;
    content: string;
    tags: string;
    order: number;
    status: "draft" | "published";
}

export function DocEditorForm({
    initial,
    mode,
}: {
    initial?: Partial<DocDraft> & { id?: string };
    mode: "create" | "edit";
}) {
    const [draft, setDraft] = useState<DocDraft>({
        title: initial?.title ?? "",
        path: initial?.path ?? "",
        kind: initial?.kind ?? "doc",
        summary: initial?.summary ?? "",
        content: initial?.content ?? "",
        tags: (initial?.tags ?? []).join(", "),
        order: initial?.order ?? 0,
        status: initial?.status ?? "published",
    });
    const [error, setError] = useState("");
    const [saving, setSaving] = useState(false);

    function patch(partial: Partial<DocDraft>) {
        setDraft((d) => ({ ...d, ...partial }));
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSaving(true);
        setError("");
        const payload = (extra: Record<string, unknown> = {}) => ({
            title: draft.title,
            summary: draft.summary,
            content: draft.content,
            tags: draft.tags.split(",").map((t) => t.trim()).filter(Boolean),
            order: draft.order,
            status: draft.status,
            ...extra,
        });
        try {
            let resp: Response;
            if (mode === "create") {
                resp = await fetch("/api/admin/docs", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(payload({ path: draft.path, kind: draft.kind })),
                });
            } else {
                resp = await fetch(`/api/admin/docs/${initial?.id}`, {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(payload()),
                });
            }
            if (resp.ok) {
                navigate("/admin/docs");
                return;
            }
            const body = (await resp.json()) as { error?: string };
            setError(body.error ?? "保存失败");
        } finally {
            setSaving(false);
        }
    }

    const inputStyle = { width: "100%", padding: "8px 10px", borderRadius: 8, border: `1px solid ${v.border}` };

    return (
        <form onSubmit={onSubmit} style={{ display: "grid", gap: 14, maxWidth: 760 }}>
            <label style={{ display: "grid", gap: 6, fontSize: 13 }}>
                标题 *
                <input className="ven-input" style={inputStyle} value={draft.title} onChange={(e) => patch({ title: e.target.value })} required />
            </label>
            {mode === "create" && (
                <>
                    <label style={{ display: "grid", gap: 6, fontSize: 13 }}>
                        路径（全路径，如 ai/attention；中间目录自动创建）
                        <input className="ven-input" style={inputStyle} value={draft.path} onChange={(e) => patch({ path: e.target.value })} placeholder="notes/ai/attention" />
                    </label>
                    <label style={{ display: "grid", gap: 6, fontSize: 13 }}>
                        类型
                        <select className="ven-input" style={inputStyle} value={draft.kind} onChange={(e) => patch({ kind: e.target.value as "doc" | "section" })}>
                            <option value="doc">文档</option>
                            <option value="section">目录</option>
                        </select>
                    </label>
                </>
            )}
            <label style={{ display: "grid", gap: 6, fontSize: 13 }}>
                摘要（≤200 字）
                <textarea className="ven-input" style={{ ...inputStyle, minHeight: 60 }} value={draft.summary} onChange={(e) => patch({ summary: e.target.value })} />
            </label>
            <label style={{ display: "grid", gap: 6, fontSize: 13 }}>
                正文（Markdown）
                <textarea className="ven-input" style={{ ...inputStyle, minHeight: 320, fontFamily: "ui-monospace, monospace" }} value={draft.content} onChange={(e) => patch({ content: e.target.value })} />
            </label>
            <div style={{ display: "flex", gap: 14 }}>
                <label style={{ display: "grid", gap: 6, fontSize: 13, flex: 1 }}>
                    标签（逗号分隔，≤8 个）
                    <input className="ven-input" style={inputStyle} value={draft.tags} onChange={(e) => patch({ tags: e.target.value })} />
                </label>
                <label style={{ display: "grid", gap: 6, fontSize: 13, width: 120 }}>
                    排序
                    <input className="ven-input" style={inputStyle} type="number" value={draft.order} onChange={(e) => patch({ order: Number(e.target.value) })} />
                </label>
                <label style={{ display: "grid", gap: 6, fontSize: 13, width: 140 }}>
                    状态
                    <select className="ven-input" style={inputStyle} value={draft.status} onChange={(e) => patch({ status: e.target.value as "draft" | "published" })}>
                        <option value="published">发布</option>
                        <option value="draft">草稿</option>
                    </select>
                </label>
            </div>
            {error && <p style={{ color: "#c0392b" }}>{error}</p>}
            <div style={{ display: "flex", gap: 12 }}>
                <button className="ven-btn ven-btn-primary" type="submit" disabled={saving}>
                    {saving ? "保存中…" : "保存"}
                </button>
                <button className="ven-btn" type="button" onClick={() => navigate("/admin/docs")}>
                    取消
                </button>
            </div>
        </form>
    );
}
