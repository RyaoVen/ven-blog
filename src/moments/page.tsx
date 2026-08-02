/** 动态页（ISR 静态页）：说说时间线，点击卡片弹窗看详情；数据变更由框架失效再生 + SSE 推送刷新 */

import { useMemo, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { formatDateTime } from "../lib/format";
import { HeartIcon, MessageIcon } from "../lib/icons";
import { Layout } from "../lib/layout";
import { renderMarkdown } from "../lib/markdown";
import { markdownCss } from "../lib/markdownCss";
import { Modal } from "../lib/modal";
import { v } from "../lib/theme";
import { CommentsSection } from "../posts/comments";
import { useMomentLikes } from "./likes";
import type { Moment, MomentsState } from "./types";

export default function MomentsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { moments: [], commentCounts: {}, likeCounts: {} }) as MomentsState;
    const [selected, setSelected] = useState<Moment | null>(null);
    const { counts, likedIds, toggleLike } = useMomentLikes(state.likeCounts);

    return (
        <Layout>
            <style>{markdownCss}</style>
            <header style={{ marginBottom: 24 }}>
                <h1 style={{ fontSize: 28 }}>动态</h1>
                <p className="ven-meta" style={{ margin: 0 }}>
                    共 {state.moments.length} 条
                </p>
            </header>
            {state.moments.length === 0 ? (
                <p style={{ color: v.textSecondary }}>还没有动态。</p>
            ) : (
                <ul
                    style={{
                        listStyle: "none",
                        padding: 0,
                        margin: 0,
                        display: "flex",
                        flexDirection: "column",
                        gap: 16,
                    }}
                >
                    {state.moments.map((m) => (
                        <li key={m.id}>
                            <div
                                role="button"
                                tabIndex={0}
                                onClick={() => setSelected(m)}
                                onKeyDown={(e) => e.key === "Enter" && setSelected(m)}
                                className="ven-card ven-card-hover ven-clickable"
                                style={{
                                    display: "block",
                                    width: "100%",
                                    textAlign: "left",
                                    padding: "18px 22px",
                                }}
                            >
                                <div style={{ display: "flex", alignItems: "center", gap: "8px 12px", flexWrap: "wrap", marginBottom: 12 }}>
                                    <MomentAvatar name={m.authorName} />
                                    <span style={{ fontWeight: 550 }}>{m.authorName}</span>
                                    {m.pinned && (
                                        <span
                                            className="ven-chip"
                                            style={{ color: v.accent, borderColor: v.accent, flexShrink: 0 }}
                                        >
                                            置顶
                                        </span>
                                    )}
                                    <span className="ven-meta">{formatDateTime(m.createdAt)}</span>
                                    <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 14 }}>
                                        <button
                                            type="button"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                toggleLike(m.id);
                                            }}
                                            className="ven-meta ven-inline-action"
                                            style={{
                                                display: "inline-flex",
                                                alignItems: "center",
                                                gap: 4,
                                                border: "none",
                                                background: "none",
                                                cursor: "pointer",
                                                padding: 0,
                                                color: likedIds.has(m.id) ? v.accent : v.textMuted,
                                            }}
                                        >
                                            <HeartIcon size={12} filled={likedIds.has(m.id)} />
                                            {counts[m.id] ?? 0}
                                        </button>
                                        <span className="ven-meta" style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
                                            <MessageIcon size={12} />
                                            {state.commentCounts[m.id] ?? 0}
                                        </span>
                                    </span>
                                </div>
                                <div
                                    className="ven-prose ven-comment-prose"
                                    style={{
                                        overflow: "hidden",
                                        display: "-webkit-box",
                                        WebkitLineClamp: 3,
                                        WebkitBoxOrient: "vertical",
                                    }}
                                >
                                    <MomentBody content={m.content} />
                                </div>
                            </div>
                        </li>
                    ))}
                </ul>
            )}
            <Modal open={selected !== null} onClose={() => setSelected(null)} width={880}>
                {selected && (
                    <div>
                        <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
                            <MomentAvatar name={selected.authorName} size={36} />
                            <div>
                                <div style={{ fontWeight: 650 }}>{selected.authorName}</div>
                                <span className="ven-meta">{formatDateTime(selected.createdAt)}</span>
                            </div>
                        </div>
                        <div className="ven-prose">
                            <MomentBody content={selected.content} />
                        </div>
                        <CommentsSection targetPath={`/moments/${selected.id}`} />
                    </div>
                )}
            </Modal>
        </Layout>
    );
}

/** 动态正文 Markdown 渲染（同文章管线，html:false 防 XSS） */
function MomentBody({ content }: { content: string }) {
    const rendered = useMemo(() => renderMarkdown(content), [content]);
    return <div dangerouslySetInnerHTML={{ __html: rendered.html }} />;
}

/** 动态作者字母头像（方角黑底） */
function MomentAvatar({ name, size = 28 }: { name: string; size?: number }) {
    return (
        <span
            style={{
                width: size,
                height: size,
                borderRadius: 3,
                background: v.text,
                display: "inline-flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: size * 0.46,
                fontWeight: 700,
                fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
                color: v.bg,
                flexShrink: 0,
            }}
        >
            {name.slice(0, 1).toUpperCase()}
        </span>
    );
}
