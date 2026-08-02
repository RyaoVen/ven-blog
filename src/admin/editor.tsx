/** 文章编辑器（后台 /admin/posts/new 与 /admin/posts/[id]/edit 共用）。
 * 主区仅正文（+预览/插图）；标题/分类/摘要/标签/封面集中在「发布设置」弹窗填写。
 * 封面留空且正文有图时，服务端自动取第一张图为封面（弹窗内也可点选正文图片）。 */

import { ChangeEvent, FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { navigate } from "../app/router";
import { CheckIcon, XIcon } from "../lib/icons";
import { renderMarkdown } from "../lib/markdown";
import { markdownCss } from "../lib/markdownCss";
import { Modal } from "../lib/modal";
import { v } from "../lib/theme";
import type { Post } from "../posts/types";

interface PostMeta {
    title: string;
    category: string;
    summary: string;
    coverUrl: string;
    tags: string[];
}

/** 提取正文中的图片 URL（markdown 图片语法） */
function contentImagesOf(content: string): string[] {
    const pattern = /!\[[^\]]*\]\(([^)\s]+)\)/g;
    const urls: string[] = [];
    let m: RegExpExecArray | null;
    while ((m = pattern.exec(content)) !== null) {
        if (!urls.includes(m[1])) {
            urls.push(m[1]);
        }
    }
    return urls;
}

export function PostEditor({ post, categories }: { post: Post | null; categories: string[] }) {
    const editing = post;
    const [meta, setMeta] = useState<PostMeta>({
        title: editing?.title ?? "",
        category: editing?.category || categories[0] || "",
        summary: editing?.summary ?? "",
        coverUrl: editing?.coverUrl ?? "",
        tags: editing?.tags ?? [],
    });
    const [content, setContent] = useState(editing?.content ?? "");
    const [modalOpen, setModalOpen] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [preview, setPreview] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [linkModalOpen, setLinkModalOpen] = useState(false);
    const fileRef = useRef<HTMLInputElement>(null);
    const contentRef = useRef<HTMLTextAreaElement>(null);

    const contentImages = useMemo(() => contentImagesOf(content), [content]);

    // insertSnippet 把片段插到 textarea 光标处（预览态/无光标时追加末尾）
    function insertSnippet(snippet: string) {
        const ta = contentRef.current;
        if (!ta) {
            setContent((prev) => prev + snippet);
            return;
        }
        const start = ta.selectionStart;
        const end = ta.selectionEnd;
        setContent((prev) => prev.slice(0, start) + snippet + prev.slice(end));
    }

    // insertImage 把图片 Markdown 插到光标处
    function insertImage(url: string) {
        insertSnippet(`![](${url})`);
    }

    // insertLinkCard 把链接块插到光标处（:::link URL + 标题/简介/图标 三行）
    function insertLinkCard(link: { url: string; title: string; desc: string; icon: string }) {
        insertSnippet(`\n:::link ${link.url}\n${link.title}\n${link.desc}\n${link.icon}\n:::\n`);
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
        if (!meta.title.trim()) {
            setError("请先填写标题（发布设置）");
            setModalOpen(true);
            return;
        }
        if (!meta.category) {
            setError("请选择分类（发布设置）");
            setModalOpen(true);
            return;
        }
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch(editing ? `/api/posts/${editing.id}` : "/api/posts", {
                method: editing ? "PUT" : "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    title: meta.title,
                    category: meta.category,
                    content,
                    summary: meta.summary,
                    coverUrl: meta.coverUrl,
                    tags: meta.tags,
                }),
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
        <form style={{ display: "flex", flexDirection: "column", gap: 14 }} onSubmit={onSubmit}>
            {/* 参数摘要行 + 弹窗入口 */}
            <div
                className="ven-card"
                style={{ padding: "14px 18px", display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}
            >
                <span style={{ fontWeight: 650, fontSize: 15, color: meta.title ? v.text : v.textMuted }}>
                    {meta.title || "未命名文章"}
                </span>
                <span className="ven-chip">{meta.category || "未分类"}</span>
                {meta.tags.map((t) => (
                    <span key={t} className="ven-chip">
                        {t}
                    </span>
                ))}
                <span className="ven-meta">{meta.coverUrl ? "封面已设置" : "封面自动（首图）"}</span>
                <button type="button" className="ven-btn" style={{ marginLeft: "auto" }} onClick={() => setModalOpen(true)}>
                    发布设置
                </button>
            </div>

            {preview ? (
                <div className="ven-card ven-prose" style={{ padding: "16px 20px", minHeight: 300 }}>
                    <style>{markdownCss}</style>
                    <div dangerouslySetInnerHTML={{ __html: renderMarkdown(content).html }} />
                </div>
            ) : (
                <textarea
                    ref={contentRef}
                    className="ven-input"
                    placeholder="正文（Markdown：代码块/表格/列表/引用/:::warning 警告、:::tip 提示、:::note 注意、:::link 链接块）"
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    rows={18}
                    required
                />
            )}
            {error && <p style={{ color: v.danger, fontSize: 14, margin: 0 }}>{error}</p>}
            <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
                <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                    {submitting ? "保存中…" : editing ? "保存" : "发布"}
                </button>
                <button className="ven-btn" type="button" onClick={() => setPreview((p) => !p)}>
                    {preview ? "继续编辑" : "预览"}
                </button>
                <button className="ven-btn" type="button" disabled={uploading} onClick={() => fileRef.current?.click()}>
                    {uploading ? "上传中…" : "插入图片"}
                </button>
                <button className="ven-btn" type="button" onClick={() => setLinkModalOpen(true)}>
                    插入链接块
                </button>
                <input
                    ref={fileRef}
                    type="file"
                    accept="image/jpeg,image/png,image/webp,image/gif"
                    style={{ display: "none" }}
                    onChange={onPickImage}
                />
                <a style={{ fontSize: 14, color: v.textSecondary }} href={editing ? `/posts/${editing.id}` : "/admin/posts"}>
                    取消
                </a>
            </div>

            <MetaModal
                open={modalOpen}
                onClose={() => setModalOpen(false)}
                meta={meta}
                onChange={setMeta}
                categories={categories}
                contentImages={contentImages}
            />
            <LinkBlockModal
                open={linkModalOpen}
                onClose={() => setLinkModalOpen(false)}
                onInsert={(link) => {
                    insertLinkCard(link);
                    setLinkModalOpen(false);
                }}
            />
        </form>
    );
}

/* ===== 链接块弹窗：填 URL → 服务端解析站名/简介/图标 → 可修改后插入 ===== */
interface LinkDraft {
    url: string;
    title: string;
    desc: string;
    icon: string;
}

function LinkBlockModal({
    open,
    onClose,
    onInsert,
}: {
    open: boolean;
    onClose: () => void;
    onInsert: (link: LinkDraft) => void;
}) {
    const [url, setUrl] = useState("");
    const [draft, setDraft] = useState<LinkDraft | null>(null);
    const [parsing, setParsing] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // 每次打开重置
    useEffect(() => {
        if (open) {
            setUrl("");
            setDraft(null);
            setError(null);
        }
    }, [open]);

    async function resolve() {
        const target = url.trim();
        if (!/^https?:\/\/\S+$/i.test(target)) {
            setError("请输入 http(s) 链接");
            return;
        }
        setParsing(true);
        setError(null);
        try {
            const resp = await fetch(`/api/admin/linkpreview?url=${encodeURIComponent(target)}`);
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error === "fetch failed" ? "抓取失败（目标站不可达或拒绝）" : (data?.error ?? "解析失败"));
                return;
            }
            setDraft({ url: data.url || target, title: data.title ?? "", desc: data.desc ?? "", icon: data.icon ?? "" });
        } catch {
            setError("网络错误，请重试");
        } finally {
            setParsing(false);
        }
    }

    return (
        <Modal open={open} onClose={onClose} width={480}>
            <h3 style={{ margin: "0 0 16px", fontSize: 17 }}>插入链接块</h3>
            <div style={{ display: "flex", gap: 8, marginBottom: 14 }}>
                <input
                    className="ven-input"
                    style={{ flex: 1 }}
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    onKeyDown={(e) => {
                        if (e.key === "Enter") {
                            e.preventDefault();
                            resolve();
                        }
                    }}
                    placeholder="https://…"
                    autoFocus
                />
                <button className="ven-btn ven-btn-primary" type="button" disabled={parsing || !url.trim()} onClick={resolve}>
                    {parsing ? "解析中…" : "解析"}
                </button>
            </div>
            {error && <p style={{ color: v.danger, fontSize: 13, margin: "0 0 12px" }}>{error}</p>}
            {draft && (
                <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                    {/* 卡片预览 */}
                    <div
                        className="ven-card"
                        style={{ display: "flex", alignItems: "center", gap: 12, padding: "12px 14px" }}
                    >
                        {draft.icon ? (
                            <img
                                src={draft.icon}
                                alt=""
                                style={{ width: 36, height: 36, borderRadius: 3, objectFit: "cover", border: `1px solid ${v.border}` }}
                                onError={(e) => ((e.target as HTMLImageElement).style.visibility = "hidden")}
                            />
                        ) : (
                            <span
                                style={{
                                    width: 36,
                                    height: 36,
                                    borderRadius: 3,
                                    border: `1px solid ${v.border}`,
                                    display: "inline-flex",
                                    alignItems: "center",
                                    justifyContent: "center",
                                    color: v.textMuted,
                                    flexShrink: 0,
                                }}
                            >
                                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
                                    <circle cx="12" cy="12" r="9" />
                                    <path d="M3 12h18M12 3c2.5 2.6 4 5.7 4 9s-1.5 6.4-4 9c-2.5-2.6-4-5.7-4-9s1.5-6.4 4-9z" />
                                </svg>
                            </span>
                        )}
                        <div style={{ minWidth: 0 }}>
                            <div style={{ fontWeight: 650, fontSize: 14, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                                {draft.title || "（无标题）"}
                            </div>
                            <div className="ven-meta" style={{ fontSize: 11 }}>
                                {(() => {
                                    try {
                                        return new URL(draft.url).host;
                                    } catch {
                                        return draft.url;
                                    }
                                })()}
                            </div>
                        </div>
                    </div>
                    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                        网站名
                        <input className="ven-input" value={draft.title} onChange={(e) => setDraft({ ...draft, title: e.target.value })} />
                    </label>
                    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                        网站简介
                        <textarea className="ven-input" rows={2} value={draft.desc} onChange={(e) => setDraft({ ...draft, desc: e.target.value })} />
                    </label>
                    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                        图标 URL
                        <input className="ven-input" value={draft.icon} onChange={(e) => setDraft({ ...draft, icon: e.target.value })} />
                    </label>
                    <div style={{ display: "flex", justifyContent: "flex-end", gap: 10 }}>
                        <button className="ven-btn" type="button" onClick={onClose}>
                            取消
                        </button>
                        <button className="ven-btn ven-btn-primary" type="button" onClick={() => onInsert(draft)}>
                            插入
                        </button>
                    </div>
                </div>
            )}
        </Modal>
    );
}

/* ===== 发布设置弹窗 ===== */
function MetaModal({
    open,
    onClose,
    meta,
    onChange,
    categories,
    contentImages,
}: {
    open: boolean;
    onClose: () => void;
    meta: PostMeta;
    onChange: (meta: PostMeta) => void;
    categories: string[];
    contentImages: string[];
}) {
    const [draft, setDraft] = useState<PostMeta>(meta);
    const [tagInput, setTagInput] = useState("");
    const [error, setError] = useState<string | null>(null);

    // 每次打开时用外部 meta 重置草稿
    useEffect(() => {
        if (open) {
            setDraft(meta);
            setError(null);
            setTagInput("");
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open]);

    function patch(partial: Partial<PostMeta>) {
        setDraft((d) => ({ ...d, ...partial }));
    }

    function addTag() {
        const name = tagInput.trim();
        if (!name) {
            return;
        }
        if (draft.tags.includes(name)) {
            setTagInput("");
            return;
        }
        if (draft.tags.length >= 8) {
            setError("标签最多 8 个");
            return;
        }
        patch({ tags: [...draft.tags, name] });
        setTagInput("");
        setError(null);
    }

    function save() {
        if (!draft.title.trim()) {
            setError("标题必填");
            return;
        }
        if (!draft.category) {
            setError("分类必选");
            return;
        }
        onChange({ ...draft, title: draft.title.trim() });
        onClose();
    }

    return (
        <Modal open={open} onClose={onClose} width={520}>
            <h3 style={{ margin: "0 0 16px", fontSize: 17 }}>发布设置</h3>
            <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
                <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                    标题 <span className="ven-meta">必填</span>
                    <input
                        className="ven-input"
                        value={draft.title}
                        onChange={(e) => patch({ title: e.target.value })}
                        placeholder="文章标题"
                    />
                </label>
                <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                    分类 <span className="ven-meta">必选</span>
                    <select className="ven-input" value={draft.category} onChange={(e) => patch({ category: e.target.value })}>
                        {categories.length === 0 && <option value="">（先到 设置 → 文章分类 添加）</option>}
                        {categories.map((c) => (
                            <option key={c} value={c}>
                                {c}
                            </option>
                        ))}
                    </select>
                </label>
                <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                    副标题 <span className="ven-meta">可选，列表摘要（留空自动截取正文）</span>
                    <textarea
                        className="ven-input"
                        rows={2}
                        maxLength={200}
                        value={draft.summary}
                        onChange={(e) => patch({ summary: e.target.value })}
                    />
                </label>
                <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 14 }}>
                    标签 <span className="ven-meta">可选，任意个数（最多 8）</span>
                    {draft.tags.length > 0 && (
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                            {draft.tags.map((t) => (
                                <span key={t} className="ven-chip" style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                                    {t}
                                    <button
                                        type="button"
                                        onClick={() => patch({ tags: draft.tags.filter((x) => x !== t) })}
                                        style={{ border: "none", background: "none", padding: 0, cursor: "pointer", color: "inherit", display: "inline-flex" }}
                                        aria-label={`移除标签 ${t}`}
                                    >
                                        <XIcon size={10} />
                                    </button>
                                </span>
                            ))}
                        </div>
                    )}
                    <div style={{ display: "flex", gap: 8 }}>
                        <input
                            className="ven-input"
                            value={tagInput}
                            onChange={(e) => setTagInput(e.target.value)}
                            onKeyDown={(e) => {
                                if (e.key === "Enter") {
                                    e.preventDefault();
                                    addTag();
                                }
                            }}
                            placeholder="输入标签名，回车添加"
                        />
                        <button className="ven-btn" type="button" onClick={addTag}>
                            添加
                        </button>
                    </div>
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 8, fontSize: 14 }}>
                    封面图 <span className="ven-meta">可选；不选且正文有图时自动用第一张</span>
                    <input
                        className="ven-input"
                        value={draft.coverUrl}
                        onChange={(e) => patch({ coverUrl: e.target.value })}
                        placeholder="封面图 URL（或从下方正文图片选择）"
                    />
                    {contentImages.length > 0 && (
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 10 }}>
                            {contentImages.map((url) => {
                                const selected = draft.coverUrl === url;
                                return (
                                    <button
                                        key={url}
                                        type="button"
                                        onClick={() => patch({ coverUrl: selected ? "" : url })}
                                        style={{
                                            border: selected ? `2px solid ${v.accent}` : `1px solid ${v.border}`,
                                            borderRadius: 3,
                                            padding: 0,
                                            cursor: "pointer",
                                            background: "none",
                                            position: "relative",
                                        }}
                                        title={selected ? "取消选择" : "设为封面"}
                                    >
                                        <img src={url} alt="正文图片" style={{ display: "block", width: 88, height: 60, objectFit: "cover", borderRadius: 2 }} />
                                        {selected && (
                                            <span
                                                style={{
                                                    position: "absolute",
                                                    top: 3,
                                                    right: 3,
                                                    width: 16,
                                                    height: 16,
                                                    borderRadius: "50%",
                                                    background: v.accent,
                                                    color: "#fff",
                                                    display: "inline-flex",
                                                    alignItems: "center",
                                                    justifyContent: "center",
                                                }}
                                            >
                                                <CheckIcon size={10} />
                                            </span>
                                        )}
                                    </button>
                                );
                            })}
                        </div>
                    )}
                </div>
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 4 }}>
                    <button className="ven-btn" type="button" onClick={onClose}>
                        取消
                    </button>
                    <button className="ven-btn ven-btn-primary" type="button" onClick={save}>
                        完成
                    </button>
                </div>
            </div>
        </Modal>
    );
}
