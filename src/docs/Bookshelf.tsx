/** docs 2.0 书架页：顶层文档 = 一本书（复用个人主页裱框卡视觉：ven-frame + 四角括线 + 展品签）。
 * 左栏分类筛选（书 tags 聚合，文章列表页同款 TagRow 交互）+ 关键词过滤 + 右侧书卡网格。 */

import { useMemo, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { formatDateTime } from "../lib/format";
import { v } from "../lib/theme";
import type { BookShelfItem, DocsHomeState } from "./types";

/** FrameCorners 四角括线（复刻 author 页裱框 SVG） */
function FrameCorners() {
    return (
        <svg className="ven-frame-corners" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
            <path d="M12 1 H1 V12" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
            <path d="M88 1 H99 V12" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
            <path d="M88 99 H99 V88" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
            <path d="M12 99 H1 V88" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
        </svg>
    );
}

/** BookCard 书本裱框卡：左侧书脊 + 书名/摘要 + 展品签 */
function BookCard({ book, exhibit }: { book: BookShelfItem; exhibit: number }) {
    const href = `/docs/${book.path}`;
    return (
        <a
            href={href}
            className="ven-frame ven-clickable"
            onClick={(e) => {
                e.preventDefault();
                navigate(href);
            }}
            style={{ display: "block", minHeight: 200, textDecoration: "none" }}
        >
            <FrameCorners />
            <div className="ven-frame-inner" style={{ flexDirection: "row", gap: 16, padding: 0, overflow: "hidden" }}>
                {/* 书脊 */}
                <div
                    aria-hidden="true"
                    style={{
                        width: 26,
                        flexShrink: 0,
                        background: `linear-gradient(180deg, ${v.accent} 0%, color-mix(in srgb, ${v.accent} 55%, ${v.bg}) 100%)`,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                    }}
                >
                    <span
                        style={{
                            writingMode: "vertical-rl",
                            fontSize: 10,
                            letterSpacing: 3,
                            color: "#fff",
                            opacity: 0.9,
                            whiteSpace: "nowrap",
                        }}
                    >
                        VEN · DOCS
                    </span>
                </div>
                {/* 封面内容 */}
                <div style={{ flex: 1, display: "flex", flexDirection: "column", padding: "16px 18px 14px 0", minWidth: 0 }}>
                    <p className="ven-meta" style={{ margin: "0 0 8px" }}>
                        {book.kind === "section" ? "文集" : "单篇"} · {book.chapters} 章
                    </p>
                    <div style={{ fontWeight: 650, fontSize: 17, color: v.text, lineHeight: 1.4 }}>{book.title}</div>
                    <p
                        style={{
                            margin: "8px 0 0",
                            fontSize: 13,
                            color: v.textSecondary,
                            overflow: "hidden",
                            display: "-webkit-box",
                            WebkitLineClamp: 3,
                            WebkitBoxOrient: "vertical",
                        }}
                    >
                        {book.summary || "（无简介）"}
                    </p>
                    <div style={{ marginTop: "auto", paddingTop: 14, display: "flex", justifyContent: "space-between", alignItems: "flex-end" }}>
                        <span className="ven-meta" style={{ fontSize: 11 }}>
                            更新于 {formatDateTime(book.updatedAt)}
                        </span>
                        <span className="ven-meta" style={{ fontSize: 10 }}>
                            VOL. {String(exhibit).padStart(2, "0")}
                        </span>
                    </div>
                </div>
            </div>
        </a>
    );
}

export default function DocsBookshelfPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { mode: "bookshelf", books: [], tree: [] }) as DocsHomeState;
    const books = state.books ?? [];
    const [category, setCategory] = useState("");
    const [keyword, setKeyword] = useState("");

    // 分类 = 全部书的 tags 聚合（去重排序）；关键词过滤标题/摘要。
    const categories = useMemo(
        () => [...new Set(books.flatMap((b) => b.tags ?? []))].sort((a, b) => a.localeCompare(b, "zh")),
        [books],
    );
    const filtered = useMemo(() => {
        const kw = keyword.trim().toLowerCase();
        return books.filter((b) => {
            if (category && !(b.tags ?? []).includes(category)) {
                return false;
            }
            if (kw && !`${b.title}${b.summary}`.toLowerCase().includes(kw)) {
                return false;
            }
            return true;
        });
    }, [books, category, keyword]);

    return (
        <Layout>
            <header style={{ marginBottom: 20 }}>
                <h1 style={{ fontSize: 30, margin: "0 0 8px" }}>书架</h1>
                <p style={{ color: v.textSecondary }}>笔记与项目文档，按册陈列。</p>
            </header>
            <div style={{ display: "flex", gap: 28, alignItems: "flex-start" }}>
                {/* 左栏：分类筛选（文章列表页同款 TagRow） */}
                <aside style={{ width: 168, flexShrink: 0, position: "sticky", top: 80 }}>
                    <div style={{ display: "flex", flexDirection: "column" }}>
                        <FilterTag label="全部" active={category === ""} onClick={() => setCategory("")} />
                        {categories.map((c) => (
                            <FilterTag key={c} label={c} active={category === c} onClick={() => setCategory(c)} />
                        ))}
                    </div>
                </aside>
                {/* 右侧：搜索 + 书卡网格 */}
                <section style={{ flex: 1, minWidth: 0 }}>
                    <input
                        className="ven-input"
                        style={{ width: "100%", maxWidth: 420, padding: "8px 12px", borderRadius: 8, border: `1px solid ${v.border}`, marginBottom: 18 }}
                        value={keyword}
                        onChange={(e) => setKeyword(e.target.value)}
                        placeholder="搜索书名或简介…"
                    />
                    <p className="ven-meta" style={{ margin: "0 0 14px" }}>
                        {category ? `分类「${category}」 · ` : ""}
                        共 {filtered.length} 册
                    </p>
                    {filtered.length === 0 ? (
                        <p style={{ color: v.textSecondary }}>
                            {books.length === 0
                                ? "书架空空如也——通过后台或 agent 创建第一本书吧。"
                                : "没有匹配的书。"}
                        </p>
                    ) : (
                        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))", gap: 20 }}>
                            {filtered.map((b, i) => (
                                <BookCard key={b.path} book={b} exhibit={i + 1} />
                            ))}
                        </div>
                    )}
                </section>
            </div>
        </Layout>
    );
}

/** FilterTag 分类筛选行（复刻文章列表页 TagRow 交互，客户端过滤） */
function FilterTag({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
    return (
        <button
            type="button"
            className="ven-accent-item"
            onClick={onClick}
            style={{
                display: "block",
                padding: "6px 0 6px 12px",
                fontSize: 14,
                textAlign: "left",
                textDecoration: "none",
                background: "none",
                border: "none",
                cursor: "pointer",
                color: active ? v.accent : v.textSecondary,
                fontWeight: active ? 650 : 400,
            }}
        >
            {label}
        </button>
    );
}
