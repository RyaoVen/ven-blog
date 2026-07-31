/** 首页：整屏板块（滚动磁吸 + 非线性跳转）——hero / 双列表 / 仪表盘 / 时间线 / 订阅 */

import { FormEvent, RefObject, useEffect, useMemo, useRef, useState } from "react";
import gsap from "gsap";
import type { PageAppProps } from "./app/pageApp";
import { PulseLine, SignalWaves } from "./lib/ambient";
import { CountUp } from "./lib/countUp";
import { formatDateTime } from "./lib/format";
import { Layout } from "./lib/layout";
import { Reveal, useInView } from "./lib/reveal";
import { scrollToElement, scrollToNextSection } from "./lib/scrollAnim";
import { Tilt } from "./lib/tilt";
import { v } from "./lib/theme";
import type { HomeState, HomeTimelineItem } from "./home/types";

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";
const PANEL_LABELS = ["首页", "列表", "仪表盘", "时间线", "订阅"];

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
    const containerRef = useRef<HTMLDivElement>(null);

    // 滚动磁吸仅首页生效（离开页面解除）
    useEffect(() => {
        document.documentElement.classList.add("ven-home-snap");
        return () => document.documentElement.classList.remove("ven-home-snap");
    }, []);

    return (
        <Layout>
            <div ref={containerRef}>
                <Panel index={0}>
                    <Hero state={state} />
                </Panel>
                <Panel index={1}>
                    <DualLists state={state} />
                </Panel>
                <Panel index={2}>
                    <Dashboard state={state} />
                </Panel>
                <Panel index={3}>
                    <Timeline items={state.timeline} />
                </Panel>
                <Panel index={4} last>
                    <Subscribe />
                </Panel>
            </div>
            <PanelNav containerRef={containerRef} />
        </Layout>
    );
}

/* ===== 整屏板块外壳（底部 chevron 非线性滚向下一屏） ===== */
function Panel({ index, last = false, children }: { index: number; last?: boolean; children: React.ReactNode }) {
    const ref = useRef<HTMLElement>(null);
    return (
        <section ref={ref} className="ven-panel" data-panel={index}>
            {children}
            {!last && (
                <button
                    type="button"
                    className="ven-panel-chevron ven-btn"
                    style={{ padding: "6px 10px" }}
                    aria-label="滚动到下一屏"
                    onClick={() => scrollToNextSection(ref.current)}
                >
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                        <path d="M2 5 L8 11 L14 5" stroke="currentColor" strokeWidth="1.5" />
                    </svg>
                </button>
            )}
        </section>
    );
}

/* ===== 右侧圆点章节导航 ===== */
function PanelNav({ containerRef }: { containerRef: RefObject<HTMLDivElement | null> }) {
    const [panels, setPanels] = useState<Element[]>([]);
    const [active, setActive] = useState(0);

    useEffect(() => {
        const els = Array.from(containerRef.current?.querySelectorAll(".ven-panel") ?? []);
        setPanels(els);
        const observer = new IntersectionObserver(
            (entries) => {
                for (const entry of entries) {
                    if (entry.isIntersecting) {
                        setActive(els.indexOf(entry.target));
                    }
                }
            },
            { threshold: 0.5 },
        );
        els.forEach((el) => observer.observe(el));
        return () => observer.disconnect();
    }, [containerRef]);

    if (panels.length === 0) {
        return null;
    }
    return (
        <nav className="ven-panel-nav" aria-label="章节导航">
            {panels.map((p, i) => (
                <a
                    key={i}
                    href={`#panel-${i}`}
                    className={active === i ? "active" : ""}
                    title={PANEL_LABELS[i]}
                    onClick={(e) => {
                        e.preventDefault();
                        scrollToElement(p);
                    }}
                />
            ))}
        </nav>
    );
}

/* ===== 板块一：hero ===== */
function Hero({ state }: { state: HomeState }) {
    const { author } = state;
    return (
        <div className="ven-hero">
            <div>
                <p className="ven-meta" style={{ margin: 0 }}>
                    PERSONAL SITE / VEN-BLOG
                </p>
                <h1 style={{ fontSize: 52, letterSpacing: "-0.03em", margin: "16px 0 18px" }}>
                    RyaoVen 的博客
                </h1>
                <p style={{ fontSize: 16, color: v.textSecondary, maxWidth: 520, marginBottom: 28 }}>
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
        </div>
    );
}

/* ===== 板块二：双短列表 ===== */
function DualLists({ state }: { state: HomeState }) {
    return (
        <Reveal>
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))",
                    gap: 40,
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
                                        padding: "12px 0",
                                        borderBottom: `1px solid ${v.border}`,
                                    }}
                                >
                                    <a
                                        href={`/posts/${p.id}`}
                                        style={{
                                            fontSize: 15,
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
                                <li key={m.id} style={{ padding: "12px 0", borderBottom: `1px solid ${v.border}` }}>
                                    <p
                                        style={{
                                            margin: "0 0 4px",
                                            fontSize: 14.5,
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
            </div>
        </Reveal>
    );
}

function ListHeader({ title, moreHref }: { title: string; moreHref: string }) {
    return (
        <div
            style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "baseline",
                paddingBottom: 12,
                marginBottom: 6,
                borderBottom: `1px solid ${v.text}`,
            }}
        >
            <h2 style={{ fontSize: 20, margin: 0 }}>{title}</h2>
            <a href={moreHref} className="ven-meta" style={{ textDecoration: "none" }}>
                更多 →
            </a>
        </div>
    );
}

/* ===== 板块三：仪表盘 ===== */
function Dashboard({ state }: { state: HomeState }) {
    return (
        <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
                <ListHeader title="仪表盘" moreHref="/rss.xml" />
                <PulseLine />
            </div>
            <Reveal>
                <div
                    style={{
                        display: "grid",
                        gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
                        gap: 16,
                        marginBottom: 36,
                    }}
                >
                    <StatCard label="文章总数" value={state.stats.posts} unit="篇" />
                    <StatCard label="累计字数" value={state.stats.words} unit="字" />
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 28 }}>
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
            </Reveal>
        </div>
    );
}

function StatCard({ label, value, unit }: { label: string; value: number; unit: string }) {
    return (
        <div className="ven-card" style={{ padding: "20px 22px" }}>
            <div className="ven-meta" style={{ marginBottom: 10 }}>
                {label}
            </div>
            <div style={{ fontFamily: mono, fontSize: 34, fontWeight: 700, letterSpacing: "-0.02em" }}>
                <CountUp value={value} format={(n) => (n >= 10000 ? (n / 10000).toFixed(1) + "w" : String(n))} />
                <span style={{ fontSize: 13, fontWeight: 400, color: v.textMuted, marginLeft: 6 }}>{unit}</span>
            </div>
        </div>
    );
}

/* ===== 板块四：文章时间线（竖线生长 + 节点弹出） ===== */
function Timeline({ items }: { items: HomeTimelineItem[] }) {
    const { ref, inView } = useInView<HTMLDivElement>(0.25);
    const played = useRef(false);

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

    useEffect(() => {
        if (!inView || played.current || !ref.current) {
            return;
        }
        played.current = true;
        const ctx = gsap.context(() => {
            gsap.fromTo(
                ".ven-tl-line",
                { scaleY: 0, transformOrigin: "top" },
                { scaleY: 1, duration: 0.9, ease: "power2.out" },
            );
            gsap.fromTo(
                ".ven-tl-dot",
                { scale: 0 },
                { scale: 1, duration: 0.35, stagger: 0.12, ease: "back.out(2.5)", delay: 0.3 },
            );
        }, ref);
        return () => ctx.revert();
    }, [inView, ref]);

    if (items.length === 0) {
        return null;
    }
    return (
        <div ref={ref}>
            <ListHeader title="文章时间线" moreHref="/posts" />
            <div className="ven-tl-line" style={{ borderLeft: `1px solid ${v.borderStrong}`, marginLeft: 6, paddingLeft: 24 }}>
                {groups.map(([month, posts]) => (
                    <div key={month} style={{ position: "relative", paddingBottom: 26 }}>
                        <span
                            className="ven-tl-dot"
                            style={{
                                position: "absolute",
                                left: -29,
                                top: 6,
                                width: 9,
                                height: 9,
                                background: v.text,
                            }}
                        />
                        <p className="ven-meta" style={{ margin: "0 0 10px", fontWeight: 700 }}>
                            {month}（{posts.length} 篇）
                        </p>
                        <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                            {posts.map((p) => (
                                <li key={p.id} style={{ display: "flex", gap: 14, alignItems: "baseline", padding: "5px 0" }}>
                                    <span className="ven-meta" style={{ flexShrink: 0, width: 44 }}>
                                        {p.createdAt.slice(8, 10)} 日
                                    </span>
                                    <a href={`/posts/${p.id}`} style={{ fontSize: 15, color: v.text, textDecoration: "none" }}>
                                        {p.title}
                                    </a>
                                </li>
                            ))}
                        </ul>
                    </div>
                ))}
            </div>
        </div>
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
        <Reveal>
            <ListHeader title="订阅本站" moreHref="/rss.xml" />
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 28, marginTop: 24 }}>
                <div className="ven-card" style={{ padding: "22px 24px" }}>
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
                <div className="ven-card" style={{ padding: "22px 24px", display: "flex", justifyContent: "space-between", gap: 16 }}>
                    <div>
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
                    <SignalWaves />
                </div>
            </div>
        </Reveal>
    );
}
