/** 首页：hero（介绍 + 3D 作者卡）/ 双短列表 / 仪表盘 / 文章时间线 / 订阅区 */

import { FormEvent, useMemo, useState } from "react";
import type { PageAppProps } from "./app/pageApp";
import { formatDateTime } from "./lib/format";
import { Layout } from "./lib/layout";
import { Tilt } from "./lib/tilt";
import { v } from "./lib/theme";
import type { HomeState, HomeTimelineItem } from "./home/types";

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

export default function HomePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? {
        recentPosts: [],
        recentMoments: [],
        stats: { posts: 0, words: 0 },
        projects: [],
        quotes: [],
        timeline: [],
        author: { username: "author", bio: "", avatarUrl: "", github: "" },
    }) as HomeState;

    return (
        <Layout>
            <Hero state={state} />
            <DualLists state={state} />
            <Dashboard state={state} />
            <Timeline items={state.timeline} />
            <Subscribe />
        </Layout>
    );
}

/* ===== 板块一：hero ===== */
function Hero({ state }: { state: HomeState }) {
    const { author } = state;
    return (
        <section
            className="ven-hero"
            style={{
                borderTop: `1px solid ${v.text}`,
                borderBottom: `1px solid ${v.border}`,
                padding: "48px 0 40px",
                marginBottom: 40,
            }}
        >
            <div>
                <p className="ven-meta" style={{ margin: 0 }}>
                    PERSONAL SITE / VEN-BLOG
                </p>
                <h1 style={{ fontSize: 40, letterSpacing: "-0.03em", margin: "14px 0 16px" }}>
                    RyaoVen 的博客
                </h1>
                <p style={{ fontSize: 15.5, color: v.textSecondary, maxWidth: 520, marginBottom: 24 }}>
                    记录技术与生活。聊聊框架设计、后端工程与渲染链路，本站由自研 VenHybird
                    框架驱动——SSR 直出、SPA 接管、ISR 物化、SSE 实时推送。
                </p>
                <div style={{ display: "flex", gap: 12 }}>
                    <a href="/posts" className="ven-btn ven-btn-primary">
                        开始阅读
                    </a>
                    <a href="/moments" className="ven-btn">
                        看看动态
                    </a>
                </div>
            </div>
            <Tilt max={9}>
                <div className="ven-card" style={{ width: 260, padding: "24px 22px", flexShrink: 0 }}>
                    <div style={{ display: "flex", alignItems: "center", gap: 14, marginBottom: 14 }}>
                        <span
                            style={{
                                width: 52,
                                height: 52,
                                borderRadius: 2,
                                background: v.text,
                                display: "inline-flex",
                                alignItems: "center",
                                justifyContent: "center",
                                fontSize: 22,
                                fontWeight: 700,
                                fontFamily: mono,
                                color: v.bg,
                            }}
                        >
                            {author.username.slice(0, 1).toUpperCase()}
                        </span>
                        <div>
                            <div style={{ fontWeight: 700, fontSize: 17 }}>{author.username}</div>
                            <div className="ven-meta">AUTHOR</div>
                        </div>
                    </div>
                    <p style={{ fontSize: 13.5, color: v.textSecondary, marginBottom: 18 }}>
                        {author.bio || "ven_hybird 框架作者，本站站长。喜欢折腾渲染链路与开发者工具。"}
                    </p>
                    <div style={{ display: "flex", gap: 10 }}>
                        <a href={`/author/${author.username}`} className="ven-btn" style={{ flex: 1 }}>
                            个人页
                        </a>
                        <a href={author.github} className="ven-btn" style={{ flex: 1 }} target="_blank" rel="noreferrer">
                            GitHub
                        </a>
                    </div>
                </div>
            </Tilt>
        </section>
    );
}

/* ===== 板块二：双短列表 ===== */
function DualLists({ state }: { state: HomeState }) {
    return (
        <section
            style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))",
                gap: 32,
                marginBottom: 48,
            }}
        >
            <div>
                <ListHeader title="最近文章" moreHref="/posts" />
                {state.recentPosts.length === 0 ? (
                    <p style={{ color: v.textMuted, fontSize: 14 }}>还没有文章。</p>
                ) : (
                    <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                        {state.recentPosts.map((p) => (
                            <li
                                key={p.id}
                                style={{
                                    display: "flex",
                                    justifyContent: "space-between",
                                    alignItems: "baseline",
                                    gap: 16,
                                    padding: "10px 0",
                                    borderBottom: `1px solid ${v.border}`,
                                }}
                            >
                                <a
                                    href={`/posts/${p.id}`}
                                    style={{
                                        fontSize: 14.5,
                                        fontWeight: 550,
                                        color: v.text,
                                        textDecoration: "none",
                                        overflow: "hidden",
                                        textOverflow: "ellipsis",
                                        whiteSpace: "nowrap",
                                    }}
                                >
                                    {p.title}
                                </a>
                                <span className="ven-meta" style={{ flexShrink: 0 }}>
                                    {p.createdAt.slice(0, 10)}
                                </span>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
            <div>
                <ListHeader title="最近动态" moreHref="/moments" />
                {state.recentMoments.length === 0 ? (
                    <p style={{ color: v.textMuted, fontSize: 14 }}>还没有动态。</p>
                ) : (
                    <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                        {state.recentMoments.map((m) => (
                            <li
                                key={m.id}
                                style={{
                                    padding: "10px 0",
                                    borderBottom: `1px solid ${v.border}`,
                                }}
                            >
                                <p
                                    style={{
                                        margin: "0 0 4px",
                                        fontSize: 14,
                                        color: v.text,
                                        overflow: "hidden",
                                        textOverflow: "ellipsis",
                                        whiteSpace: "nowrap",
                                    }}
                                >
                                    {m.content}
                                </p>
                                <span className="ven-meta">{formatDateTime(m.createdAt)}</span>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </section>
    );
}

function ListHeader({ title, moreHref }: { title: string; moreHref: string }) {
    return (
        <div
            style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "baseline",
                paddingBottom: 10,
                marginBottom: 4,
                borderBottom: `1px solid ${v.text}`,
            }}
        >
            <h2 style={{ fontSize: 18, margin: 0 }}>{title}</h2>
            <a href={moreHref} className="ven-meta" style={{ textDecoration: "none" }}>
                更多 →
            </a>
        </div>
    );
}

/* ===== 板块三：仪表盘 ===== */
function Dashboard({ state }: { state: HomeState }) {
    return (
        <section style={{ marginBottom: 48 }}>
            <ListHeader title="仪表盘" moreHref="/rss.xml" />
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))",
                    gap: 16,
                    margin: "20px 0 28px",
                }}
            >
                <StatCard label="文章总数" value={String(state.stats.posts)} unit="篇" />
                <StatCard label="累计字数" value={formatWords(state.stats.words)} unit="字" />
            </div>
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
                    gap: 24,
                }}
            >
                <div>
                    <p className="ven-meta" style={{ margin: "0 0 12px" }}>
                        收藏的句子
                    </p>
                    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                        {state.quotes.map((q, i) => (
                            <blockquote
                                key={i}
                                className="ven-card"
                                style={{ margin: 0, padding: "14px 18px", borderLeft: `2px solid ${v.text}` }}
                            >
                                <p style={{ margin: "0 0 6px", fontSize: 14 }}>{q.text}</p>
                                <cite className="ven-meta" style={{ fontStyle: "normal" }}>
                                    —— {q.source}
                                </cite>
                            </blockquote>
                        ))}
                    </div>
                </div>
                <div>
                    <p className="ven-meta" style={{ margin: "0 0 12px" }}>
                        维护的项目
                    </p>
                    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                        {state.projects.map((p) => (
                            <div key={p.name} className="ven-card ven-card-hover" style={{ padding: "14px 18px" }}>
                                <a
                                    href={p.url}
                                    target="_blank"
                                    rel="noreferrer"
                                    style={{ fontWeight: 650, fontSize: 14.5, fontFamily: mono, color: v.text }}
                                >
                                    {p.name} ↗
                                </a>
                                <p style={{ margin: "6px 0 0", fontSize: 13, color: v.textSecondary }}>{p.desc}</p>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </section>
    );
}

function StatCard({ label, value, unit }: { label: string; value: string; unit: string }) {
    return (
        <div className="ven-card" style={{ padding: "18px 20px" }}>
            <div className="ven-meta" style={{ marginBottom: 8 }}>
                {label}
            </div>
            <div style={{ fontFamily: mono, fontSize: 30, fontWeight: 700, letterSpacing: "-0.02em" }}>
                {value}
                <span style={{ fontSize: 13, fontWeight: 400, color: v.textMuted, marginLeft: 6 }}>{unit}</span>
            </div>
        </div>
    );
}

function formatWords(n: number): string {
    if (n >= 10000) {
        return (n / 10000).toFixed(1) + "w";
    }
    return String(n);
}

/* ===== 板块四：文章时间线 ===== */
function Timeline({ items }: { items: HomeTimelineItem[] }) {
    const groups = useMemo(() => {
        const map = new Map<string, HomeTimelineItem[]>();
        for (const item of items) {
            const key = item.createdAt.slice(0, 7);
            const list = map.get(key) ?? [];
            list.push(item);
            map.set(key, list);
        }
        return [...map.entries()];
    }, [items]);

    if (items.length === 0) {
        return null;
    }
    return (
        <section style={{ marginBottom: 48 }}>
            <ListHeader title="文章时间线" moreHref="/posts" />
            <div style={{ borderLeft: `1px solid ${v.borderStrong}`, marginLeft: 6, paddingLeft: 24 }}>
                {groups.map(([month, posts]) => (
                    <div key={month} style={{ position: "relative", paddingBottom: 24 }}>
                        <span
                            style={{
                                position: "absolute",
                                left: -29,
                                top: 6,
                                width: 9,
                                height: 9,
                                background: v.text,
                                borderRadius: 0,
                            }}
                        />
                        <p className="ven-meta" style={{ margin: "0 0 10px", fontWeight: 700 }}>
                            {month}（{posts.length} 篇）
                        </p>
                        <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                            {posts.map((p) => (
                                <li
                                    key={p.id}
                                    style={{
                                        display: "flex",
                                        gap: 14,
                                        alignItems: "baseline",
                                        padding: "4px 0",
                                    }}
                                >
                                    <span className="ven-meta" style={{ flexShrink: 0, width: 44 }}>
                                        {p.createdAt.slice(8, 10)} 日
                                    </span>
                                    <a
                                        href={`/posts/${p.id}`}
                                        style={{ fontSize: 14.5, color: v.text, textDecoration: "none" }}
                                    >
                                        {p.title}
                                    </a>
                                </li>
                            ))}
                        </ul>
                    </div>
                ))}
            </div>
        </section>
    );
}

/* ===== 板块五：订阅 ===== */
function Subscribe() {
    const [email, setEmail] = useState("");
    const [message, setMessage] = useState<string | null>(null);
    const [ok, setOk] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setMessage(null);
        try {
            const resp = await fetch("/api/subscribe", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email }),
            });
            if (!resp.ok) {
                setOk(false);
                setMessage("邮箱格式不正确，请检查后重试");
                return;
            }
            const data = await resp.json();
            setOk(true);
            setMessage(data.already ? "这个邮箱已经订阅过啦" : "已记录你的邮箱，感谢订阅！");
            setEmail("");
        } catch {
            setOk(false);
            setMessage("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section style={{ marginBottom: 16 }}>
            <ListHeader title="订阅本站" moreHref="/rss.xml" />
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
                    gap: 24,
                    marginTop: 20,
                }}
            >
                <div className="ven-card" style={{ padding: "20px 22px" }}>
                    <p className="ven-meta" style={{ margin: "0 0 8px" }}>
                        邮箱订阅
                    </p>
                    <p style={{ fontSize: 13.5, color: v.textSecondary, margin: "0 0 14px" }}>
                        留下邮箱，新文章发布后通知你（投递能力接入中，先记录地址）。
                    </p>
                    <form onSubmit={onSubmit} style={{ display: "flex", gap: 10 }}>
                        <input
                            className="ven-input"
                            type="email"
                            placeholder="you@example.com"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            required
                        />
                        <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                            {submitting ? "提交中…" : "订阅"}
                        </button>
                    </form>
                    {message && (
                        <p style={{ margin: "10px 0 0", fontSize: 13, color: ok ? v.text : v.danger }}>{message}</p>
                    )}
                </div>
                <div className="ven-card" style={{ padding: "20px 22px" }}>
                    <p className="ven-meta" style={{ margin: "0 0 8px" }}>
                        RSS 订阅
                    </p>
                    <p style={{ fontSize: 13.5, color: v.textSecondary, margin: "0 0 14px" }}>
                        用你习惯的阅读器订阅本站 RSS，新文章自动送达。
                    </p>
                    <a href="/rss.xml" className="ven-btn" target="_blank" rel="noreferrer">
                        /rss.xml ↗
                    </a>
                </div>
            </div>
        </section>
    );
}
