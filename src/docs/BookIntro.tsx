/** docs 2.0 书籍详情页（mode=book）：书籍介绍正文 + 章节目录索引 */

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
    const rendered = useMemo(() => (book?.content ? renderMarkdown(book.content) : null), [book?.content]);

    if (!book) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>书籍不存在或未发布。</p>
            </Layout>
        );
    }

    return (
        <Layout>
            <style>{markdownCss}</style>
            <div style={{ display: "flex", gap: 20, alignItems: "baseline", marginBottom: 8 }}>
                <span className="ven-meta">VEN · DOCS</span>
                <span className="ven-meta">{chapters.length} 章</span>
            </div>
            <h1 style={{ fontSize: 32, margin: "0 0 10px", lineHeight: 1.3 }}>{book.title}</h1>
            {book.summary && <p style={{ color: v.textSecondary, marginBottom: 20 }}>{book.summary}</p>}

            {/* 书籍介绍（section 正文） */}
            {rendered?.html ? (
                <section className="ven-md" dangerouslySetInnerHTML={{ __html: rendered.html }} style={{ marginBottom: 36 }} />
            ) : null}

            {/* 章节目录索引 */}
            <section>
                <h2 style={{ fontSize: 20, margin: "0 0 14px", paddingBottom: 8, borderBottom: `1px solid ${v.border}` }}>
                    目录
                </h2>
                {chapters.length === 0 ? (
                    <p style={{ color: v.textSecondary }}>这本书还没有章节。</p>
                ) : (
                    <ol style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: 10 }}>
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
                                        className="ven-card ven-card-hover"
                                        style={{
                                            display: "flex",
                                            gap: 14,
                                            alignItems: "baseline",
                                            padding: "12px 16px",
                                            borderRadius: 10,
                                            textDecoration: "none",
                                        }}
                                    >
                                        <span
                                            className="ven-serif"
                                            style={{ fontSize: 18, color: v.accent, minWidth: 34, fontVariantNumeric: "tabular-nums" }}
                                        >
                                            {String(i + 1).padStart(2, "0")}
                                        </span>
                                        <span style={{ flex: 1, minWidth: 0 }}>
                                            <span style={{ display: "block", fontWeight: 550, color: v.text }}>{ch.title}</span>
                                            {ch.summary && (
                                                <span style={{ display: "block", fontSize: 13, color: v.textSecondary, marginTop: 3 }}>
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
