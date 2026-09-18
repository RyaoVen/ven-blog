/** docs 2.0 章节阅读页（mode=chapter）：可收起左侧章节目录 + 正文（复用文章详情渲染样式）+ 上一章/下一章 */

import { useMemo, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { formatDateTime } from "../lib/format";
import { renderMarkdown } from "../lib/markdown";
import { markdownCss } from "../lib/markdownCss";
import { v } from "../lib/theme";
import type { DocPageState, DocView } from "./types";

export function ChapterReaderPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { mode: "chapter", book: null, chapters: [], doc: null, children: [], prev: null, next: null }) as DocPageState;
    const doc = state.doc;
    const chapters = state.chapters ?? [];
    const book = state.book;
    const [sidebarOpen, setSidebarOpen] = useState(true);
    const rendered = useMemo(() => (doc?.content ? renderMarkdown(doc.content) : null), [doc?.content]);

    if (!doc) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>章节不存在或未发布。</p>
            </Layout>
        );
    }

    const renderedHtml = rendered?.html ?? "";

    return (
        <Layout>
            <style>{markdownCss}</style>
            <div style={{ display: "flex", gap: 28, alignItems: "flex-start" }}>
                {/* 章节目录侧栏（可收起） */}
                <aside
                    style={{
                        width: sidebarOpen ? 250 : 40,
                        flexShrink: 0,
                        position: "sticky",
                        top: 80,
                        transition: "width 0.24s var(--ease-out)",
                        overflow: "hidden",
                    }}
                    className="ven-docs-sidebar"
                >
                    <button
                        type="button"
                        aria-label={sidebarOpen ? "收起目录" : "展开目录"}
                        onClick={() => setSidebarOpen((s) => !s)}
                        style={{
                            border: `1px solid ${v.border}`,
                            background: v.bg,
                            color: v.textSecondary,
                            borderRadius: 8,
                            padding: "5px 0",
                            width: "100%",
                            cursor: "pointer",
                            fontSize: 13,
                            marginBottom: 10,
                        }}
                    >
                        {sidebarOpen ? "‹ 收起目录" : "›"}
                    </button>
                    {sidebarOpen && (
                        <div>
                            {book && (
                                <div style={{ padding: "0 8px 8px", borderBottom: `1px solid ${v.border}`, marginBottom: 8 }}>
                                    <a
                                        href={`/docs/${book.path}`}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            navigate(`/docs/${book.path}`);
                                        }}
                                        className="ven-meta"
                                        style={{ color: v.textSecondary, textDecoration: "none" }}
                                        title="返回书籍介绍"
                                    >
                                        《{book.title}》
                                    </a>
                                </div>
                            )}
                            {chapters.map((ch: DocView, i: number) => {
                                const active = ch.path === doc.path;
                                return (
                                    <a
                                        key={ch.id}
                                        href={`/docs/${ch.path}`}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            navigate(`/docs/${ch.path}`);
                                        }}
                                        style={{
                                            display: "flex",
                                            gap: 8,
                                            padding: "6px 8px",
                                            borderRadius: 8,
                                            fontSize: 13,
                                            color: active ? v.text : v.textSecondary,
                                            background: active ? "rgba(13,148,136,0.10)" : "transparent",
                                            textDecoration: "none",
                                            alignItems: "baseline",
                                        }}
                                    >
                                        <span style={{ fontVariantNumeric: "tabular-nums", color: active ? v.accent : v.textSecondary, fontSize: 12 }}>
                                            {String(i + 1).padStart(2, "0")}
                                        </span>
                                        <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                                            {ch.title}
                                        </span>
                                    </a>
                                );
                            })}
                        </div>
                    )}
                </aside>

                {/* 正文（复用文章详情渲染样式） */}
                <article style={{ flex: 1, minWidth: 0 }}>
                    <p style={{ color: v.textSecondary, fontSize: 13, margin: "0 0 8px" }}>
                        {book ? `《${book.title}》 · ` : ""}
                        {doc.path}
                    </p>
                    <h1 style={{ fontSize: 30, marginBottom: 12 }}>{doc.title}</h1>
                    {doc.summary && <p style={{ color: v.textSecondary, marginBottom: 20 }}>{doc.summary}</p>}
                    {renderedHtml ? (
                        <div className="ven-md" dangerouslySetInnerHTML={{ __html: renderedHtml }} />
                    ) : (
                        <p style={{ color: v.textSecondary }}>本章暂无正文。</p>
                    )}

                    {/* 上一章 / 下一章 */}
                    <div
                        style={{
                            display: "flex",
                            justifyContent: "space-between",
                            gap: 16,
                            marginTop: 40,
                            paddingTop: 16,
                            borderTop: `1px solid ${v.border}`,
                        }}
                    >
                        {state.prev ? (
                            <a
                                href={`/docs/${state.prev.path}`}
                                onClick={(e) => {
                                    e.preventDefault();
                                    navigate(`/docs/${state.prev!.path}`);
                                }}
                                style={{ color: v.textPrimary, textDecoration: "none" }}
                            >
                                ← {state.prev.title}
                            </a>
                        ) : (
                            <span />
                        )}
                        {state.next ? (
                            <a
                                href={`/docs/${state.next.path}`}
                                onClick={(e) => {
                                    e.preventDefault();
                                    navigate(`/docs/${state.next!.path}`);
                                }}
                                style={{ color: v.textPrimary, textDecoration: "none", textAlign: "right" }}
                            >
                                {state.next.title} →
                            </a>
                        ) : (
                            <span />
                        )}
                    </div>
                    <p style={{ color: v.textSecondary, fontSize: 12, marginTop: 24 }}>
                        更新于 {formatDateTime(doc.updatedAt)}
                    </p>
                </article>
            </div>
        </Layout>
    );
}
