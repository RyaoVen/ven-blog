/** docs 2.0 书籍详情页（mode=book）：歌单式头部（封面 + 元信息）+ 书籍介绍 + 章节目录 */

import { useMemo } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { formatDateTime } from "../lib/format";
import { renderMarkdown } from "../lib/markdown";
import { markdownCss } from "../lib/markdownCss";
import { v } from "../lib/theme";
import type { DocPageState } from "./types";

export function BookIntroPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { mode: "book", book: null, chapters: [], doc: null, children: [] }) as DocPageState;
    const book = state.doc ?? state.book;
    const chapters = state.chapters ?? [];
    const stats = state.stats;
    const rendered = useMemo(() => (book?.content ? renderMarkdown(book.content) : null), [book?.content]);

    if (!book) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>书籍不存在或未发布。</p>
            </Layout>
        );
    }

    const lastUpdated = stats?.lastUpdated ?? book.updatedAt;

    return (
        <Layout>
            <style>{markdownCss}</style>
            {/* ===== 歌单式头部：封面 + 标题 + 元信息行 ===== */}
            <section style={{ display: "flex", gap: 24, marginBottom: 28 }}>
                {/* 封面（大书脊卡） */}
                <div
                    aria-hidden="true"
                    style={{
                        width: 148,
                        height: 198,
                        flexShrink: 0,
                        borderRadius: 6,
                        background: `linear-gradient(160deg, ${v.accent} 0%, color-mix(in srgb, ${v.accent} 50%, ${v.bg}) 100%)`,
                        boxShadow: "var(--shadow-soft)",
                        display: "flex",
                        flexDirection: "column",
                        justifyContent: "space-between",
                        padding: 14,
                    }}
                >
                    <span className="ven-meta" style={{ color: "#fff", opacity: 0.9 }}>
                        VEN · DOCS
                    </span>
                    <span
                        className="ven-serif"
                        style={{
                            color: "#fff",
                            fontSize: 22,
                            fontWeight: 650,
                            lineHeight: 1.35,
                            display: "-webkit-box",
                            WebkitLineClamp: 4,
                            WebkitBoxOrient: "vertical",
                            overflow: "hidden",
                        }}
                    >
                        {book.title}
                    </span>
                    <span style={{ width: 46, height: 3, background: "rgba(255,255,255,0.7)" }} />
                </div>

                {/* 标题与元信息 */}
                <div style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column" }}>
                    <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                        文档 · {book.kind === "section" ? "文集" : "单篇"}
                    </p>
                    <h1 style={{ fontSize: 30, margin: "0 0 10px", lineHeight: 1.3 }}>{book.title}</h1>
                    {book.summary && <p style={{ color: v.textSecondary, marginBottom: 12 }}>{book.summary}</p>}
                    <div className="ven-meta" style={{ display: "flex", flexWrap: "wrap", gap: "6px 18px", marginTop: "auto" }}>
                        <span>{(stats?.chapters ?? chapters.length)} 章</span>
                        {stats && stats.totalChars > 0 && <span>约 {stats.totalChars.toLocaleString()} 字</span>}
                        <span>创建于 {formatDateTime(book.createdAt)}</span>
                        <span style={{ color: v.accent }}>最后更新于 {formatDateTime(lastUpdated)}</span>
                    </div>
                    {(book.tags ?? []).length > 0 && (
                        <div style={{ display: "flex", gap: 8, marginTop: 12, flexWrap: "wrap" }}>
                            {(book.tags ?? []).map((t) => (
                                <span key={t} className="ven-chip">
                                    {t}
                                </span>
                            ))}
                        </div>
                    )}
                </div>
            </section>

            {/* 书籍介绍（section 正文） */}
            {rendered?.html ? (
                <section className="ven-md" dangerouslySetInnerHTML={{ __html: rendered.html }} style={{ marginBottom: 36 }} />
            ) : null}

            {/* 章节目录（曲目式列表） */}
            <section>
                <div
                    style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "baseline",
                        marginBottom: 6,
                        paddingBottom: 8,
                        borderBottom: `1px solid ${v.border}`,
                    }}
                >
                    <h2 style={{ fontSize: 20, margin: 0 }}>目录</h2>
                    <span className="ven-meta">{chapters.length} 章</span>
                </div>
                {chapters.length === 0 ? (
                    <p style={{ color: v.textSecondary }}>这本书还没有章节。</p>
                ) : (
                    <ol style={{ listStyle: "none", margin: 0, padding: 0 }}>
                        {chapters.map((ch, i) => {
                            const href = `/docs/${ch.path}`;
                            return (
                                <li key={ch.id}>
                                    <a
                                        href={href}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            navigate(href);
                                        }}
                                        style={{
                                            display: "flex",
                                            gap: 14,
                                            alignItems: "baseline",
                                            padding: "11px 12px",
                                            borderRadius: 8,
                                            textDecoration: "none",
                                            borderBottom: `1px solid ${v.border}`,
                                        }}
                                        onMouseEnter={(e) => (e.currentTarget.style.background = v.bgSubtle ?? "rgba(128,128,128,0.06)")}
                                        onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
                                    >
                                        <span
                                            className="ven-serif"
                                            style={{ fontSize: 16, color: v.textSecondary, minWidth: 32, fontVariantNumeric: "tabular-nums" }}
                                        >
                                            {String(i + 1).padStart(2, "0")}
                                        </span>
                                        <span style={{ flex: 1, minWidth: 0 }}>
                                            <span style={{ display: "block", fontWeight: 550, color: v.text }}>{ch.title}</span>
                                            {ch.summary && (
                                                <span style={{ display: "block", fontSize: 13, color: v.textSecondary, marginTop: 2 }}>
                                                    {ch.summary}
                                                </span>
                                            )}
                                        </span>
                                        <span className="ven-meta" style={{ fontSize: 11 }}>
                                            {formatDateTime(ch.updatedAt)}
                                        </span>
                                    </a>
                                </li>
                            );
                        })}
                    </ol>
                )}
            </section>
        </Layout>
    );
}
