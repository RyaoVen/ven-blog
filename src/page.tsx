/** 首页：整屏板块（固定滚动：滚轮手势一屏一切换）——hero / 双列表 / 仪表盘 / 时间线 / 订阅 */

import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import gsap from "gsap";
import type { PageAppProps } from "./app/pageApp";
import { MarchLine, PulseLine, SignalWaves } from "./lib/ambient";
import { CountUp } from "./lib/countUp";
import { useFixedSections } from "./lib/fixedScroll";
import { formatDateTime } from "./lib/format";
import { Layout } from "./lib/layout";
import { LiveDuration } from "./lib/duration";
import { NixieClock } from "./lib/nixie";
import { Reveal, useInView } from "./lib/reveal";
import { Tilt } from "./lib/tilt";
import { Typewriter } from "./lib/typewriter";
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
    const { index, goTo } = useFixedSections(containerRef, 5);

    return (
        <Layout>
            <div ref={containerRef}>
                <Panel index={0} onNext={() => goTo(1)}>
                    <Hero state={state} />
                </Panel>
                <Panel index={1} onNext={() => goTo(2)}>
                    <DualLists state={state} />
                </Panel>
                <Panel index={2} onNext={() => goTo(3)}>
                    <Dashboard state={state} />
                </Panel>
                <Panel index={3} onNext={() => goTo(4)}>
                    <Timeline items={state.timeline} />
                </Panel>
                <Panel index={4} last>
                    <Subscribe />
                </Panel>
            </div>
            <PanelNav containerRef={containerRef} active={index} onSelect={goTo} />
        </Layout>
    );
}

/* ===== 整屏板块外壳（底部 chevron 非线性滚向下一屏） ===== */
function Panel({
    index,
    last = false,
    onNext,
    children,
}: {
    index: number;
    last?: boolean;
    onNext?: () => void;
    children: React.ReactNode;
}) {
    return (
        <section className="ven-panel" data-panel={index}>
            {children}
            {!last && (
                <button
                    type="button"
                    className="ven-panel-chevron ven-btn"
                    style={{ padding: "6px 10px" }}
                    aria-label="滚动到下一屏"
                    onClick={onNext}
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
function PanelNav({
    containerRef,
    active,
    onSelect,
}: {
    containerRef: React.RefObject<HTMLDivElement | null>;
    active: number;
    onSelect: (index: number) => void;
}) {
    const [panels, setPanels] = useState<Element[]>([]);

    useEffect(() => {
        setPanels(Array.from(containerRef.current?.querySelectorAll(".ven-panel") ?? []));
    }, [containerRef]);

    if (panels.length === 0) {
        return null;
    }
    return (
        <nav className="ven-panel-nav" aria-label="章节导航">
            {panels.map((_, i) => (
                <a
                    key={i}
                    href={`#panel-${i}`}
                    className={active === i ? "active" : ""}
                    title={PANEL_LABELS[i]}
                    onClick={(e) => {
                        e.preventDefault();
                        onSelect(i);
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
        <div className="ven-hero ven-crosshair">
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
            <Tilt max={10}>
                <HeroAuthorCard state={state} />
            </Tilt>
        </div>
    );
}

/* ===== hero 作者卡：裱框 + 双环旋转头像 + 状态脉冲 + 元信息栏 ===== */
function HeroAuthorCard({ state }: { state: HomeState }) {
    const { author } = state;
    return (
        <div className="ven-frame" style={{ width: 330, padding: 14, flexShrink: 0 }}>
            <svg className="ven-frame-corners" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                <path d="M12 1 H1 V12" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
                <path d="M88 1 H99 V12" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
                <path d="M88 99 H99 V88" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
                <path d="M12 99 H1 V88" fill="none" strokeWidth="1" vectorEffect="non-scaling-stroke" />
            </svg>
            <div className="ven-frame-inner ven-card" style={{ padding: "26px 24px 22px" }}>
                {/* 头像：双层旋转环（外虚线顺时针 / 内玉青弧逆时针） */}
                <div style={{ display: "flex", alignItems: "center", gap: 18, marginBottom: 18 }}>
                    <div style={{ position: "relative", width: 88, height: 88, flexShrink: 0 }}>
                        <svg className="ven-spin-slow" viewBox="0 0 88 88" style={{ position: "absolute", inset: 0 }} aria-hidden="true">
                            <circle cx="44" cy="44" r="42" fill="none" stroke="var(--border-strong)" strokeWidth="1" strokeDasharray="3 7" />
                        </svg>
                        <svg className="ven-spin-rev" viewBox="0 0 88 88" style={{ position: "absolute", inset: 0 }} aria-hidden="true">
                            <circle cx="44" cy="44" r="35" fill="none" stroke="var(--accent)" strokeWidth="1.5" strokeDasharray="52 168" strokeLinecap="round" />
                        </svg>
                        <div style={{ position: "absolute", inset: 10 }}>
                            {author.avatarUrl ? (
                                <img
                                    src={author.avatarUrl}
                                    alt={`${author.username} 的头像`}
                                    style={{ width: "100%", height: "100%", borderRadius: 3, objectFit: "cover", border: `1px solid ${v.border}` }}
                                />
                            ) : (
                                <span
                                    style={{
                                        width: "100%",
                                        height: "100%",
                                        borderRadius: 3,
                                        background: v.text,
                                        display: "inline-flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                        fontSize: 28,
                                        fontWeight: 700,
                                        fontFamily: mono,
                                        color: v.bg,
                                    }}
                                >
                                    {author.username.slice(0, 1).toUpperCase()}
                                </span>
                            )}
                        </div>
                    </div>
                    <div style={{ minWidth: 0 }}>
                        <div className="ven-serif" style={{ fontWeight: 700, fontSize: 24, letterSpacing: "-0.01em", lineHeight: 1.2 }}>
                            {author.username}
                        </div>
                        <div className="ven-meta" style={{ marginTop: 4 }}>
                            AUTHOR · 站长
                        </div>
                        <div style={{ display: "inline-flex", alignItems: "center", gap: 7, marginTop: 10 }}>
                            <span
                                style={{
                                    width: 7,
                                    height: 7,
                                    borderRadius: "50%",
                                    background: "#22c55e",
                                    boxShadow: "0 0 6px rgba(34,197,94,0.6)",
                                    animation: "ven-blink 1.8s steps(1) infinite",
                                }}
                            />
                            <span className="ven-meta" style={{ fontSize: 10 }}>
                                OPEN FOR IDEAS
                            </span>
                        </div>
                    </div>
                </div>
                <p className="ven-serif" style={{ fontSize: 14.5, lineHeight: 1.8, color: v.textSecondary, margin: "0 0 18px" }}>
                    {author.bio || "ven_hybird 框架作者，本站站长。喜欢折腾渲染链路与开发者工具。"}
                </p>
                {/* 元信息栏 */}
                <div
                    style={{
                        display: "grid",
                        gridTemplateColumns: "repeat(3, 1fr)",
                        gap: 8,
                        padding: "12px 0",
                        borderTop: `1px solid ${v.border}`,
                        borderBottom: `1px solid ${v.border}`,
                        marginBottom: 18,
                    }}
                >
                    {[
                        ["ROLE", "AUTHOR"],
                        ["FIELD", "RENDERING"],
                        ["STACK", "GO+NODE"],
                    ].map(([k, val]) => (
                        <div key={k}>
                            <div className="ven-meta" style={{ fontSize: 9, marginBottom: 3 }}>
                                {k}
                            </div>
                            <div style={{ fontSize: 12, fontWeight: 650, fontFamily: mono, color: v.text }}>
                                {val}
                            </div>
                        </div>
                    ))}
                </div>
                <div style={{ display: "flex", gap: 10 }}>
                    <a href={`/author/${author.username}`} className="ven-btn ven-btn-primary" style={{ flex: 1, justifyContent: "center" }}>
                        个人页
                    </a>
                    <a href={author.github} className="ven-btn" style={{ flex: 1, justifyContent: "center" }} target="_blank" rel="noreferrer">
                        GitHub ↗
                    </a>
                </div>
            </div>
        </div>
    );
}

/* ===== 板块二：双短列表 ===== */
function DualLists({ state }: { state: HomeState }) {
    return (
        <Reveal>
            <div style={{ marginBottom: 28 }}>
                <MarchLine width={220} />
            </div>
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))",
                    gap: 48,
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
                                    className="ven-accent-item"
                                    style={{
                                        display: "flex",
                                        justifyContent: "space-between",
                                        alignItems: "baseline",
                                        gap: 16,
                                        padding: "16px 0 16px 12px",
                                        borderBottom: `1px solid ${v.border}`,
                                    }}
                                >
                                    <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0 }}>
                                        {p.pinned && (
                                            <span
                                                className="ven-chip"
                                                style={{ color: v.accent, borderColor: v.accent, flexShrink: 0 }}
                                            >
                                                置顶
                                            </span>
                                        )}
                                        <a
                                            href={`/posts/${p.id}`}
                                            style={{
                                                fontSize: 16,
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
                                    </div>
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
                                    className="ven-accent-item"
                                    style={{ padding: "16px 0 16px 12px", borderBottom: `1px solid ${v.border}` }}
                                >
                                    <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
                                        {m.pinned && (
                                            <span
                                                className="ven-chip"
                                                style={{ color: v.accent, borderColor: v.accent, flexShrink: 0 }}
                                            >
                                                置顶
                                            </span>
                                        )}
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
                                    </div>
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
                        gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))",
                        gap: 16,
                        marginBottom: 36,
                    }}
                >
                    <StatCard label="文章总数" value={state.stats.posts} unit="篇" />
                    <StatCard label="累计字数" value={state.stats.words} unit="字" />
                    <StatCard label="运营时长" durationSince={state.stats.launchAt} fallbackDays={state.stats.days} />
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 28 }}>
                    <div>
                        <p className="ven-meta" style={{ margin: "0 0 12px" }}>
                            收藏的句子
                        </p>
                        <Typewriter items={state.quotes} />
                        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginTop: 16 }}>
                            <a
                                href={state.stats.latestID ? `/posts/${state.stats.latestID}` : "/posts"}
                                className="ven-card ven-card-hover ven-clickable"
                                style={{ padding: "16px 18px", textDecoration: "none", display: "block" }}
                            >
                                <div className="ven-meta" style={{ marginBottom: 8 }}>
                                    最新文章
                                </div>
                                <div style={{ fontFamily: mono, fontSize: 17, fontWeight: 700, color: v.text }}>
                                    更新于 {state.stats.latestAgo || "—"}
                                </div>
                            </a>
                            <NixieClock />
                        </div>
                    </div>
                    <div>
                        <p className="ven-meta" style={{ margin: "0 0 12px" }}>
                            维护的项目
                        </p>
                        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                            {state.projects.map((p) => (
                                <a
                                    key={p.name}
                                    href={p.url}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="ven-card ven-card-hover ven-clickable"
                                    style={{ padding: "14px 18px", textDecoration: "none", display: "block" }}
                                >
                                    <span style={{ fontWeight: 650, fontSize: 14.5, fontFamily: mono, color: v.text }}>
                                        {p.name} ↗
                                    </span>
                                    <p style={{ margin: "6px 0 0", fontSize: 13, color: v.textSecondary }}>{p.desc}</p>
                                </a>
                            ))}
                        </div>
                    </div>
                </div>
            </Reveal>
        </div>
    );
}

function StatCard({
    label,
    value,
    unit,
    durationSince,
    fallbackDays,
}: {
    label: string;
    value?: number;
    unit?: string;
    durationSince?: string;
    fallbackDays?: number;
}) {
    return (
        <div className="ven-card" style={{ padding: "20px 22px" }}>
            <div className="ven-meta" style={{ marginBottom: 10 }}>
                {label}
            </div>
            {durationSince !== undefined ? (
                <div style={{ fontFamily: mono, fontSize: 17, fontWeight: 700, letterSpacing: "-0.01em" }}>
                    {durationSince ? <LiveDuration since={durationSince} /> : `${fallbackDays ?? 0} 天`}
                </div>
            ) : (
                <div style={{ fontFamily: mono, fontSize: 34, fontWeight: 700, letterSpacing: "-0.02em" }}>
                    <CountUp value={value ?? 0} format={(n) => (n >= 10000 ? (n / 10000).toFixed(1) + "w" : String(n))} />
                    {unit && (
                        <span style={{ fontSize: 13, fontWeight: 400, color: v.textMuted, marginLeft: 6 }}>{unit}</span>
                    )}
                </div>
            )}
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
                { scaleX: 0, transformOrigin: "left" },
                { scaleX: 1, duration: 0.9, ease: "power2.out" },
            );
            gsap.fromTo(
                ".ven-tl-dot",
                { scale: 0 },
                { scale: 1, duration: 0.35, stagger: 0.12, ease: "back.out(2.5)", delay: 0.3 },
            );
            gsap.to(".ven-tl-dot", {
                opacity: 0.25,
                duration: 1.2,
                repeat: -1,
                yoyo: true,
                ease: "sine.inOut",
                stagger: 0.5,
                delay: 1.2,
            });
        }, ref);
        return () => ctx.revert();
    }, [inView, ref]);

    if (items.length === 0) {
        return null;
    }
    return (
        <div ref={ref}>
            <ListHeader title="文章时间线" moreHref="/posts" />
            <div style={{ overflowX: "auto", paddingBottom: 10 }}>
                <div style={{ display: "flex", gap: 36, position: "relative", minWidth: "max-content", paddingTop: 22 }}>
                    <div
                        className="ven-tl-line"
                        style={{
                            position: "absolute",
                            top: 4,
                            left: 0,
                            right: 0,
                            height: 1,
                            background: v.borderStrong,
                            transformOrigin: "left",
                        }}
                    />
                    {groups.map(([month, posts]) => (
                        <div key={month} style={{ position: "relative", minWidth: 180 }}>
                            <span
                                className="ven-tl-dot"
                                style={{ position: "absolute", top: -22, left: 0, width: 9, height: 9, background: v.text }}
                            />
                            <p className="ven-meta" style={{ margin: "0 0 10px", fontWeight: 700 }}>
                                {month}（{posts.length} 篇）
                            </p>
                            <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                                {posts.map((p) => (
                                    <li key={p.id} style={{ padding: "5px 0" }}>
                                        <div className="ven-meta" style={{ marginBottom: 2 }}>
                                            {p.createdAt.slice(8, 10)} 日
                                        </div>
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
