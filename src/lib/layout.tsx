/** 站点通用布局：全局样式注入 + 顶部导航（左品牌与作者头像 / 中搜索 / 右主题与账户）+ 页脚 */

import { ReactNode, useEffect, useState } from "react";
import { navigate } from "../app/router";
import { globalCss } from "./globalCss";
import { HeaderSearch } from "./headerSearch";
import { PageEnter } from "./motion";
import { useRole } from "./role";
import { ThemeToggle } from "./themeToggle";
import { layout as layoutToken, v } from "./theme";

/** 全局样式注入（SSR/客户端同构输出，React 会原样渲染） */
export function GlobalStyle() {
    return <style>{globalCss}</style>;
}

const styles = {
    page: {
        maxWidth: layoutToken.container,
        margin: "0 auto",
        padding: "0 24px 64px",
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
    },
    header: {
        padding: "14px 0",
        borderBottom: `1px solid ${v.border}`,
        marginBottom: 32,
    },
    side: { display: "flex", alignItems: "center", gap: 14 },
    brand: {
        display: "flex",
        alignItems: "center",
        gap: 10,
        fontWeight: 700,
        fontSize: 18,
        color: v.text,
        textDecoration: "none",
    },
    brandDot: {
        width: 24,
        height: 24,
        borderRadius: 3,
        background: v.text,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        color: v.bg,
        fontSize: 12,
        fontWeight: 700,
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
    },
    authorAvatar: {
        width: 24,
        height: 24,
        borderRadius: 3,
        border: `1px solid ${v.borderStrong}`,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: 12,
        fontWeight: 700,
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
        color: v.textSecondary,
        textDecoration: "none",
    },
    nav: { display: "flex", gap: 14, alignItems: "center", fontSize: 14 },
    navLink: { color: v.textSecondary, textDecoration: "none" },
    main: { flex: 1 },
    footer: {
        marginTop: 48,
        borderTop: `1px solid ${v.border}`,
        padding: "32px 0 20px",
    },
    footerGrid: {
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
        gap: 28,
        marginBottom: 28,
    },
    footerColTitle: { margin: "0 0 12px" },
    footerLinks: {
        listStyle: "none",
        padding: 0,
        margin: 0,
        display: "flex",
        flexDirection: "column",
        gap: 8,
        fontSize: 14,
    },
    footerBar: {
        paddingTop: 16,
        borderTop: `1px solid ${v.border}`,
        display: "flex",
        justifyContent: "space-between",
        flexWrap: "wrap",
        gap: 8,
        fontSize: 13,
        color: v.textMuted,
    },
} as const;

/** 站点信息（导航栏作者头像用；模块级缓存，全站只取一次） */
let siteInfoPromise: Promise<{ authorName: string } | null> | null = null;

function fetchSiteInfo(): Promise<{ authorName: string } | null> {
    if (!siteInfoPromise) {
        siteInfoPromise = fetch("/api/site")
            .then((r) => (r.ok ? r.json() : null))
            .catch(() => null);
    }
    return siteInfoPromise;
}

/** 作者字母头像（链到作者主页） */
function AuthorAvatar() {
    const [name, setName] = useState<string | null>(null);
    useEffect(() => {
        let cancelled = false;
        fetchSiteInfo().then((data) => {
            if (!cancelled && data) {
                setName(data.authorName);
            }
        });
        return () => {
            cancelled = true;
        };
    }, []);
    return (
        <a
            href={name ? `/author/${name}` : "/"}
            title="作者主页"
            style={styles.authorAvatar}
            aria-label="作者主页"
        >
            {(name ?? "A").slice(0, 1).toUpperCase()}
        </a>
    );
}

/** 注销并回到文章列表 */
async function logout() {
    await fetch("/auth/logout", { method: "POST" });
    navigate("/posts");
}

export function Layout({ children }: { children: ReactNode }) {
    const role = useRole();

    // 卡片鼠标跟随光斑（委托监听，仅设置 CSS 变量）+ 大图 blur-up（捕获 load 事件）
    useEffect(() => {
        const onMove = (e: MouseEvent) => {
            const card = (e.target as Element | null)?.closest?.(".ven-card");
            if (!card) {
                return;
            }
            const el = card as HTMLElement;
            const rect = el.getBoundingClientRect();
            el.style.setProperty("--mouse-x", `${e.clientX - rect.left}px`);
            el.style.setProperty("--mouse-y", `${e.clientY - rect.top}px`);
        };
        const onLoad = (e: Event) => {
            if (e.target instanceof HTMLImageElement) {
                e.target.classList.add("ven-img-loaded");
            }
        };
        document.addEventListener("mousemove", onMove, { passive: true });
        document.addEventListener("load", onLoad, true);
        return () => {
            document.removeEventListener("mousemove", onMove);
            document.removeEventListener("load", onLoad, true);
        };
    }, []);

    return (
        <div style={styles.page}>
            <GlobalStyle />
            <header className="ven-header" style={styles.header}>
                <div style={styles.side}>
                    <a href="/" style={styles.brand}>
                        <span style={styles.brandDot}>V</span>
                        ven-blog
                    </a>
                    <AuthorAvatar />
                    <nav style={styles.nav}>
                        <a href="/posts" style={styles.navLink}>
                            文章
                        </a>
                        <a href="/moments" style={styles.navLink}>
                            动态
                        </a>
                    </nav>
                </div>
                <HeaderSearch />
                <div style={styles.side}>
                    <ThemeToggle />
                    {role === "author" && (
                        <>
                            <a href="/admin" style={styles.navLink}>
                                后台
                            </a>
                            <a href="/admin/posts/new" className="ven-btn ven-btn-primary">
                                写文章
                            </a>
                        </>
                    )}
                    {role ? (
                        <button type="button" className="ven-btn" onClick={logout}>
                            注销（{role}）
                        </button>
                    ) : (
                        <>
                            <a href="/login" className="ven-btn">
                                登录
                            </a>
                            <a href="/register" className="ven-btn ven-btn-primary">
                                注册
                            </a>
                        </>
                    )}
                </div>
            </header>
            <main style={styles.main}>
                <PageEnter>{children}</PageEnter>
            </main>
            <footer style={styles.footer}>
                <div style={styles.footerGrid}>
                    <div>
                        <a href="/" style={styles.brand}>
                            <span style={styles.brandDot}>V</span>
                            ven-blog
                        </a>
                        <p style={{ fontSize: 13.5, color: v.textSecondary, margin: "12px 0 0", maxWidth: 260 }}>
                            记录技术与生活——框架设计、后端工程与渲染链路。
                        </p>
                    </div>
                    <div>
                        <p className="ven-meta" style={styles.footerColTitle}>
                            导航
                        </p>
                        <ul style={styles.footerLinks}>
                            <li>
                                <a href="/posts">全部文章</a>
                            </li>
                            <li>
                                <a href="/moments">动态</a>
                            </li>
                            <li>
                                <a href="/search">搜索</a>
                            </li>
                            <li>
                                <a href="/author/author">作者主页</a>
                            </li>
                        </ul>
                    </div>
                    <div>
                        <p className="ven-meta" style={styles.footerColTitle}>
                            订阅与联系
                        </p>
                        <ul style={styles.footerLinks}>
                            <li>
                                <a href="/rss.xml" target="_blank" rel="noreferrer">
                                    RSS 订阅
                                </a>
                            </li>
                            <li>
                                <a href="https://github.com/RyaoVen" target="_blank" rel="noreferrer">
                                    GitHub
                                </a>
                            </li>
                            <li>
                                <a href="https://github.com/RyaoVen/ven_hybird" target="_blank" rel="noreferrer">
                                    VenHybird 框架
                                </a>
                            </li>
                            <li>
                                <a href="https://github.com/RyaoVen/ven-blog" target="_blank" rel="noreferrer">
                                    本站源码
                                </a>
                            </li>
                        </ul>
                    </div>
                </div>
                <div style={styles.footerBar} className="ven-meta">
                    <span>© 2026 RYAOVEN</span>
                    <span>POWERED BY VENHYBIRD · SSR + SPA + ISR + SSE</span>
                </div>
            </footer>
        </div>
    );
}
