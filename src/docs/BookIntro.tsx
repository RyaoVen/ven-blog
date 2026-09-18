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

/** WireCover 线框几何构成封面：嵌套矩形 + 对角线 + 圆 + 基准线，纯描边无填充（蓝图制图风） */
function WireCover({ title }: { title: string }) {
    const line = v.accent;
    return (
        <div
            aria-hidden="true"
            style={{
                width: 168,
                height: 224,
                flexShrink: 0,
                borderRadius: 6,
                border: `1px solid ${v.border}`,
                background: v.bg,
                position: "relative",
                overflow: "hidden",
                boxShadow: "var(--shadow-soft)",
            }}
        >
            <svg width="100%" height="100%" viewBox="0 0 168 224" preserveAspectRatio="xMidYMid slice">
                {/* 结构线：嵌套矩形 */}
                <rect x="10" y="10" width="148" height="204" fill="none" stroke={line} strokeWidth="1" opacity="0.55" />
                <rect x="18" y="18" width="132" height="188" fill="none" stroke={line} strokeWidth="1" opacity="0.25" />
                {/* 对角结构线 */}
                <line x1="10" y1="10" x2="158" y2="214" stroke={line} strokeWidth="1" opacity="0.2" />
                <line x1="158" y1="10" x2="10" y2="214" stroke={line} strokeWidth="1" opacity="0.2" />
                {/* 几何主体：圆 + 内接三角 */}
                <circle cx="84" cy="96" r="44" fill="none" stroke={line} strokeWidth="1.2" opacity="0.8" />
                <circle cx="84" cy="96" r="26" fill="none" stroke={line} strokeWidth="1" opacity="0.45" />
                <path d="M84 60 L114 114 L54 114 Z" fill="none" stroke={line} strokeWidth="1" opacity="0.5" />
                {/* 基准线组 */}
                <line x1="24" y1="168" x2="144" y2="168" stroke={line} strokeWidth="1" opacity="0.45" />
                <line x1="24" y1="178" x2="120" y2="178" stroke={line} strokeWidth="1" opacity="0.25" />
                <line x1="24" y1="188" x2="96" y2="188" stroke={line} strokeWidth="1" opacity="0.25" />
                {/* 节点标记 */}
                <circle cx="84" cy="96" r="2.5" fill={line} />
                <circle cx="10" cy="10" r="2" fill={line} opacity="0.6" />
                <circle cx="158" cy="214" r="2" fill={line} opacity="0.6" />
            </svg>
            <span
                className="ven-meta"
                style={{
                    position: "absolute",
                    left: 24,
                    top: 22,
                    fontSize: 10,
                    letterSpacing: 2,
                }}
            >
                VEN · DOCS
            </span>
            <span
                style={{
                    position: "absolute",
                    left: 24,
                    right: 24,
                    bottom: 40,
                    fontWeight: 650,
                    fontSize: 16,
                    lineHeight: 1.4,
                    color: v.text,
                    display: "-webkit-box",
                    WebkitLineClamp: 3,
                    WebkitBoxOrient: "vertical",
                    overflow: "hidden",
                }}
            >
                {title}
            </span>
        </div>
    );
}

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
            {/* ===== 歌单式头部：线框封面 + 标题 + 元信息行 ===== */}
            <section style={{ display: "flex", gap: 24, marginBottom: 28 }}>
                <WireCover title={book.title} />

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
