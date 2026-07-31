/** 作者主页：四模块——个人介绍 / 展示柜 / 友链横滑 / 留言板（SVG 动画点缀） */

import { FormEvent, useEffect, useRef, useState } from "react";
import gsap from "gsap";
import type { PageAppProps } from "../../app/pageApp";
import { MarchLine, PulseLine } from "../../lib/ambient";
import { formatDateTime } from "../../lib/format";
import { CheckIcon, TrashIcon } from "../../lib/icons";
import { Layout } from "../../lib/layout";
import { ConfirmModal } from "../../lib/modal";
import { useRole } from "../../lib/role";
import { v } from "../../lib/theme";
import { useViewer } from "../../lib/viewer";
import type { AuthorHomeState, GuestbookEntry } from "../types";

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

/** 技术栈等级配色（deep 深入玉青 / solid 熟练竹绿 / know 了解琥珀） */
const LEVEL_COLORS: Record<string, string> = {
    deep: "#0d9488",
    solid: "#16a34a",
    know: "#d97706",
};
const LEVEL_LABELS: Record<string, string> = {
    deep: "深入",
    solid: "熟练",
    know: "了解",
};

const EMPTY: AuthorHomeState = {
    author: { username: "author", role: "author", bio: "", avatarUrl: "", createdAt: "" },
    intro: { paragraphs: [], skills: [] },
    showcase: { projects: [], articles: [] },
    friendLinks: [],
    guestbook: [],
};

export default function AuthorHomePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? EMPTY) as AuthorHomeState;
    return (
        <Layout>
            <IntroSection state={state} />
            <ShowcaseSection state={state} />
            <FriendLinksSection state={state} />
            <GuestbookSection state={state} />
        </Layout>
    );
}

/** 板块标题（衬线 + 行军线） */
function SectionHeader({ title, meta }: { title: string; meta?: string }) {
    return (
        <div style={{ marginBottom: 28 }}>
            <div style={{ display: "flex", alignItems: "baseline", gap: 16 }}>
                <h2 style={{ fontSize: 26, margin: 0 }}>{title}</h2>
                {meta && <span className="ven-meta">{meta}</span>}
            </div>
            <div style={{ marginTop: 10 }}>
                <MarchLine width={180} />
            </div>
        </div>
    );
}

/* ===== 模块一：个人介绍 ===== */
function IntroSection({ state }: { state: AuthorHomeState }) {
    const dotsRef = useRef<HTMLDivElement>(null);

    // 装饰小方块缓慢上下漂移（循环）
    useEffect(() => {
        const ctx = gsap.context(() => {
            gsap.to(".ven-intro-dot", {
                y: -8,
                duration: 2.2,
                repeat: -1,
                yoyo: true,
                ease: "sine.inOut",
                stagger: 0.5,
            });
        }, dotsRef);
        return () => ctx.revert();
    }, []);

    return (
        <section style={{ marginBottom: 72, position: "relative" }}>
            <div ref={dotsRef} aria-hidden="true" style={{ position: "absolute", right: 0, top: 0, display: "flex", gap: 12 }}>
                {[0, 1, 2].map((i) => (
                    <span
                        key={i}
                        className="ven-intro-dot"
                        style={{ width: 8, height: 8, background: i === 1 ? v.accent : v.borderStrong }}
                    />
                ))}
            </div>
            <SectionHeader title="关于我" meta="ABOUT" />
            <div style={{ maxWidth: 720 }}>
                {state.intro.paragraphs.map((p, i) => (
                    <p key={i} className="ven-serif" style={{ fontSize: 16, lineHeight: 1.9, color: i === 0 ? v.text : v.textSecondary }}>
                        {p}
                    </p>
                ))}
            </div>
            {state.intro.skills.length > 0 && (
                <div style={{ marginTop: 24 }}>
                    <p className="ven-meta" style={{ margin: "0 0 12px" }}>
                        技术栈（颜色代表熟练度）
                    </p>
                    <div style={{ display: "flex", flexWrap: "wrap", gap: 10 }}>
                        {state.intro.skills.map((s) => (
                            <span
                                key={s.name}
                                className="ven-chip"
                                style={{ display: "inline-flex", alignItems: "center", gap: 7, padding: "3px 12px" }}
                                title={LEVEL_LABELS[s.level] ?? s.level}
                            >
                                <span style={{ width: 7, height: 7, borderRadius: "50%", background: LEVEL_COLORS[s.level] ?? v.borderStrong }} />
                                {s.name}
                            </span>
                        ))}
                    </div>
                    <div className="ven-meta" style={{ display: "flex", gap: 14, marginTop: 12 }}>
                        {Object.entries(LEVEL_LABELS).map(([level, label]) => (
                            <span key={level} style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                                <span style={{ width: 7, height: 7, borderRadius: "50%", background: LEVEL_COLORS[level] }} />
                                {label}
                            </span>
                        ))}
                    </div>
                </div>
            )}
        </section>
    );
}

/* ===== 模块二：展示柜（4 固定卡位，空位敬请期待） ===== */
function ShowcaseSection({ state }: { state: AuthorHomeState }) {
    const slots: ("project" | "article" | "empty")[] = [];
    const projects = state.showcase.projects.slice(0, 2);
    const articles = state.showcase.articles.slice(0, 2);
    for (let i = 0; i < 2; i++) {
        slots.push(i < projects.length ? "project" : "empty");
    }
    for (let i = 0; i < 2; i++) {
        slots.push(i < articles.length ? "article" : "empty");
    }

    return (
        <section style={{ marginBottom: 72 }}>
            <SectionHeader title="展示柜" meta="SHOWCASE" />
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))", gap: 18 }}>
                {slots.map((kind, i) => {
                    if (kind === "project") {
                        const p = projects[i];
                        return (
                            <a
                                key={`p-${i}`}
                                href={p.url}
                                target="_blank"
                                rel="noreferrer"
                                className="ven-card ven-card-hover ven-clickable"
                                style={{ padding: "18px 20px", textDecoration: "none", display: "block", minHeight: 120 }}
                            >
                                <p className="ven-meta" style={{ margin: "0 0 8px" }}>
                                    项目
                                </p>
                                <div style={{ fontFamily: mono, fontWeight: 650, fontSize: 15, color: v.text }}>{p.name} ↗</div>
                                <p style={{ margin: "8px 0 0", fontSize: 13, color: v.textSecondary }}>{p.desc}</p>
                            </a>
                        );
                    }
                    if (kind === "article") {
                        const post = articles[i - 2];
                        return (
                            <a
                                key={`a-${i}`}
                                href={`/posts/${post.id}`}
                                className="ven-card ven-card-hover ven-clickable"
                                style={{ padding: "18px 20px", textDecoration: "none", display: "block", minHeight: 120 }}
                            >
                                <p className="ven-meta" style={{ margin: "0 0 8px" }}>
                                    文章
                                </p>
                                <div style={{ fontWeight: 650, fontSize: 15, color: v.text }}>{post.title}</div>
                                <p className="ven-meta" style={{ margin: "8px 0 0" }}>
                                    {formatDateTime(post.createdAt)}
                                </p>
                            </a>
                        );
                    }
                    return (
                        <div
                            key={`e-${i}`}
                            style={{
                                padding: "18px 20px",
                                minHeight: 120,
                                border: `1px dashed ${v.borderStrong}`,
                                borderRadius: 3,
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                            }}
                        >
                            <span className="ven-meta">敬请期待</span>
                        </div>
                    );
                })}
            </div>
        </section>
    );
}

/* ===== 模块三：友链（横向条带随页面滚动横移） ===== */
function FriendLinksSection({ state }: { state: AuthorHomeState }) {
    const trackRef = useRef<HTMLDivElement>(null);
    const wrapRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const onScroll = () => {
            const wrap = wrapRef.current;
            const track = trackRef.current;
            if (!wrap || !track) {
                return;
            }
            const max = track.scrollWidth - wrap.clientWidth;
            if (max <= 0) {
                return;
            }
            const doc = document.documentElement;
            const range = doc.scrollHeight - window.innerHeight;
            const progress = range > 0 ? Math.min(1, window.scrollY / range) : 0;
            track.style.transform = `translateX(${-progress * max}px)`;
        };
        onScroll();
        window.addEventListener("scroll", onScroll, { passive: true });
        return () => window.removeEventListener("scroll", onScroll);
    }, [state.friendLinks.length]);

    if (state.friendLinks.length === 0) {
        return null;
    }
    return (
        <section style={{ marginBottom: 72 }}>
            <SectionHeader title="友链" meta="LINKS" />
            <div ref={wrapRef} style={{ overflow: "hidden" }}>
                <div
                    ref={trackRef}
                    style={{
                        display: "flex",
                        gap: 16,
                        width: "max-content",
                        transition: "transform 0.18s cubic-bezier(0.16, 1, 0.3, 1)",
                    }}
                >
                    {state.friendLinks.map((f, i) => (
                        <a
                            key={i}
                            href={f.url}
                            target={f.url.startsWith("http") ? "_blank" : undefined}
                            rel="noreferrer"
                            className="ven-card ven-card-hover ven-clickable"
                            style={{
                                display: "flex",
                                alignItems: "center",
                                gap: 12,
                                padding: "14px 18px",
                                textDecoration: "none",
                                minWidth: 220,
                            }}
                        >
                            <span
                                style={{
                                    width: 34,
                                    height: 34,
                                    borderRadius: 3,
                                    background: v.text,
                                    display: "inline-flex",
                                    alignItems: "center",
                                    justifyContent: "center",
                                    fontSize: 15,
                                    fontWeight: 700,
                                    fontFamily: mono,
                                    color: v.bg,
                                    flexShrink: 0,
                                }}
                            >
                                {f.name.slice(0, 1).toUpperCase()}
                            </span>
                            <span>
                                <span style={{ display: "block", fontWeight: 650, fontSize: 14, color: v.text }}>{f.name}</span>
                                <span className="ven-meta" style={{ fontSize: 11 }}>
                                    {f.desc}
                                </span>
                            </span>
                        </a>
                    ))}
                </div>
            </div>
        </section>
    );
}

/* ===== 模块四：留言板 ===== */
function GuestbookSection({ state }: { state: AuthorHomeState }) {
    const role = useRole();
    const viewer = useViewer();
    const [entries, setEntries] = useState<GuestbookEntry[]>(state.guestbook);
    const [draft, setDraft] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<GuestbookEntry | null>(null);
    const [justPosted, setJustPosted] = useState(false);

    // SSE 推送刷新 initialState 后同步
    useEffect(() => setEntries(state.guestbook), [state.guestbook]);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/guestbook", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ content: draft }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error ?? "发表失败，请重试");
                return;
            }
            setDraft("");
            setEntries((l) => [data as GuestbookEntry, ...l]);
            setJustPosted(true);
            window.setTimeout(() => setJustPosted(false), 1800);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    async function confirmDelete() {
        if (!deleting) {
            return;
        }
        const resp = await fetch(`/api/guestbook/${deleting.id}`, { method: "DELETE" });
        if (resp.ok) {
            setEntries((l) => l.filter((e) => e.id !== deleting.id));
        }
        setDeleting(null);
    }

    return (
        <section style={{ marginBottom: 24 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end" }}>
                <SectionHeader title="留言板" meta="GUESTBOOK" />
                <PulseLine />
            </div>
            {role ? (
                <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 10, marginBottom: 28 }}>
                    <textarea
                        className="ven-input"
                        rows={3}
                        placeholder="写下你的留言…（500 字以内）"
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        maxLength={500}
                        required
                    />
                    {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                    <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                        <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                            {submitting ? "发表中…" : "发表留言"}
                        </button>
                        {justPosted && (
                            <span style={{ display: "inline-flex", alignItems: "center", gap: 5, fontSize: 13, color: v.accent }}>
                                <CheckIcon size={13} />
                                已发表
                            </span>
                        )}
                    </div>
                </form>
            ) : (
                <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>
                    登录后即可留言（点击右上角"登录"）。
                </p>
            )}
            {entries.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>还没有留言，来抢沙发。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {entries.map((e) => (
                        <li
                            key={e.id}
                            className="ven-accent-item"
                            style={{ padding: "12px 0 12px 12px", borderBottom: `1px solid ${v.border}` }}
                        >
                            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
                                <div style={{ display: "flex", gap: 10, alignItems: "baseline" }}>
                                    <span style={{ fontWeight: 650, fontSize: 14 }}>{e.username}</span>
                                    <span className="ven-meta">{formatDateTime(e.createdAt)}</span>
                                </div>
                                {(viewer?.userId === e.userId || role === "author") && (
                                    <button
                                        type="button"
                                        className="ven-meta"
                                        style={{
                                            border: "none",
                                            background: "none",
                                            cursor: "pointer",
                                            padding: 0,
                                            color: v.danger,
                                            display: "inline-flex",
                                            alignItems: "center",
                                            gap: 4,
                                        }}
                                        onClick={() => setDeleting(e)}
                                    >
                                        <TrashIcon size={12} />
                                        删除
                                    </button>
                                )}
                            </div>
                            <p style={{ margin: "6px 0 0", fontSize: 14.5, whiteSpace: "pre-wrap" }}>{e.content}</p>
                        </li>
                    ))}
                </ul>
            )}
            <ConfirmModal
                open={deleting !== null}
                title="删除留言"
                message={`确定删除 ${deleting?.username} 的这条留言吗？不可恢复。`}
                confirmText="删除"
                danger
                onCancel={() => setDeleting(null)}
                onConfirm={confirmDelete}
            />
        </section>
    );
}
