/** 站点通用布局：全局样式注入 + 顶部导航（左品牌与作者头像 / 中搜索 / 右主题与账户）+ 页脚 */

import { ReactNode, useEffect, useState } from "react";
import gsap from "gsap";
import { navigate } from "../app/router";
import { useCardGlow } from "./cardGlow";
import { globalCss } from "./globalCss";
import { HeaderSearch } from "./headerSearch";
import { GridIcon, LoginIcon, LogoutIcon, PenIcon, RssIcon, UserPlusIcon } from "./icons";
import { LoginForm, RegisterForm } from "./authForms";
import { Modal } from "./modal";
import { PageEnter } from "./motion";
import { useRole } from "./role";
import { ThemeToggle } from "./themeToggle";
import { layout as layoutToken, v } from "./theme";
import { useViewer } from "./viewer";

/** 全局样式注入（SSR/客户端同构输出，React 会原样渲染） */
export function GlobalStyle() {
    return <style>{globalCss}</style>;
}

const styles = {
    page: {
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
    },
    container: {
        width: "100%",
        maxWidth: layoutToken.container,
        margin: "0 auto",
        padding: "0 24px 64px",
        flex: 1,
        display: "flex",
        flexDirection: "column",
    },
    header: {
        position: "sticky",
        top: 0,
        zIndex: 100,
        background: "var(--bg)",
        borderBottom: `1px solid ${v.border}`,
    },
    headerInner: {
        width: "100%",
        padding: "14px 24px",
        position: "relative",
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

/** 站点信息（导航品牌标/作者头像/favicon 用；模块级缓存，全站只取一次） */
interface SiteInfo {
    name: string;
    authorName: string;
    icon: string;
}

let siteInfoPromise: Promise<SiteInfo | null> | null = null;

function fetchSiteInfo(): Promise<SiteInfo | null> {
    if (!siteInfoPromise) {
        siteInfoPromise = fetch("/api/site")
            .then((r) => (r.ok ? r.json() : null))
            .catch(() => null);
    }
    return siteInfoPromise;
}

/** 品牌标：站点图标（设置页可配）缺省回退字母标 */
function BrandMark({ size = 24, fontSize = 12 }: { size?: number; fontSize?: number }) {
    const [icon, setIcon] = useState<string | null>(null);
    useEffect(() => {
        let cancelled = false;
        fetchSiteInfo().then((data) => {
            if (!cancelled && data?.icon) {
                setIcon(data.icon);
                // favicon 同步（SSR 头由框架控制，这里客户端替换）
                let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
                if (!link) {
                    link = document.createElement("link");
                    link.rel = "icon";
                    document.head.appendChild(link);
                }
                link.href = data.icon;
            }
        });
        return () => {
            cancelled = true;
        };
    }, []);
    if (icon) {
        return (
            <img
                src={icon}
                alt=""
                style={{ width: size, height: size, borderRadius: 3, objectFit: "cover", display: "block" }}
            />
        );
    }
    return <span style={{ ...styles.brandDot, width: size, height: size, fontSize }}>V</span>;
}

/** 作者字母头像（链到作者主页）；labeled 时以抽屉导航行展示（头像 + 文字） */
function AuthorAvatar({ labeled = false, onNavigate }: { labeled?: boolean; onNavigate?: () => void }) {
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
    const href = name ? `/author/${name}` : "/";
    if (labeled) {
        return (
            <a href={href} className="ven-drawer-nav-link" onClick={onNavigate} title="作者主页">
                <span style={styles.authorAvatar}>{(name ?? "A").slice(0, 1).toUpperCase()}</span>
                <span>作者主页</span>
            </a>
        );
    }
    return (
        <a href={href} style={styles.authorAvatar} title="作者主页" aria-label="作者主页">
            {(name ?? "A").slice(0, 1).toUpperCase()}
        </a>
    );
}

/** 注销并回到文章列表 */
async function logout() {
    await fetch("/auth/logout", { method: "POST" });
    navigate("/posts");
}

/** 个人主页入口（登录态）：头像/字母标 + 用户名，链到 /users/<username> */
function ProfileEntry() {
    const viewer = useViewer();
    if (!viewer || !viewer.username) {
        return null;
    }
    return (
        <a
            href={`/users/${viewer.username}`}
            title="个人主页"
            aria-label="个人主页"
            style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 8,
                textDecoration: "none",
                color: v.text,
                fontSize: 14,
                fontWeight: 550,
            }}
        >
            {viewer.avatarUrl ? (
                <img
                    src={viewer.avatarUrl}
                    alt=""
                    style={{ width: 24, height: 24, borderRadius: 3, objectFit: "cover", border: `1px solid ${v.borderStrong}` }}
                />
            ) : (
                <span style={styles.authorAvatar}>{viewer.username.slice(0, 1).toUpperCase()}</span>
            )}
            {viewer.username}
        </a>
    );
}

export function Layout({ children }: { children: ReactNode }) {
    const role = useRole();
    const [authView, setAuthView] = useState<"login" | "register" | null>(null);
    const [menuOpen, setMenuOpen] = useState(false);
    useCardGlow();

    // 移动端抽屉：打开时锁定 body 滚动 + Esc 关闭
    useEffect(() => {
        if (!menuOpen) {
            return;
        }
        const prevOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                setMenuOpen(false);
            }
        };
        document.addEventListener("keydown", onKey);
        return () => {
            document.body.style.overflow = prevOverflow;
            document.removeEventListener("keydown", onKey);
        };
    }, [menuOpen]);

    // 大图 blur-up（捕获 load 事件）+ 代码块交互（委托监听，SSR 结构不变）
    useEffect(() => {
        const onLoad = (e: Event) => {
            if (e.target instanceof HTMLImageElement) {
                e.target.classList.add("ven-img-loaded");
            }
        };
        // 代码块交互（委托监听，SSR 结构不变）：复制 + 展开/收起
        const onClick = (e: MouseEvent) => {
            const target = e.target as Element | null;
            const copyBtn = target?.closest?.(".ven-codeblock-copy");
            if (copyBtn) {
                const code = copyBtn.closest(".ven-codeblock")?.querySelector("code");
                if (code) {
                    const text = code.textContent ?? "";
                    const done = () => {
                        copyBtn.textContent = "已复制 ✓";
                        window.setTimeout(() => {
                            copyBtn.textContent = "复制";
                        }, 1600);
                    };
                    if (navigator.clipboard?.writeText) {
                        void navigator.clipboard.writeText(text).then(done, done);
                    } else {
                        const ta = document.createElement("textarea");
                        ta.value = text;
                        document.body.appendChild(ta);
                        ta.select();
                        document.execCommand("copy");
                        document.body.removeChild(ta);
                        done();
                    }
                }
                return;
            }
            const toggleBtn = target?.closest?.(".ven-codeblock-toggle");
            if (toggleBtn) {
                const block = toggleBtn.closest(".ven-codeblock");
                const body = block?.querySelector(".ven-codeblock-body") as HTMLElement | null;
                const mask = block?.querySelector(".ven-codeblock-mask") as HTMLElement | null;
                if (block && body) {
                    const collapsed = block.getAttribute("data-collapsed") === "true";
                    // 展开/收起高度动画（遮罩淡入淡出），完成后清理内联样式
                    if (collapsed) {
                        block.setAttribute("data-collapsed", "false");
                        toggleBtn.textContent = "收起";
                        const full = body.scrollHeight;
                        gsap.fromTo(body, { maxHeight: 116, overflow: "hidden" }, {
                            maxHeight: full, duration: 0.45, ease: "power2.inOut",
                            onComplete: () => gsap.set(body, { clearProps: "maxHeight,overflow" }),
                        });
                        if (mask) gsap.to(mask, { opacity: 0, duration: 0.3 });
                    } else {
                        block.setAttribute("data-collapsed", "true");
                        toggleBtn.textContent = "展开";
                        gsap.fromTo(body, { maxHeight: body.scrollHeight, overflow: "hidden" }, {
                            maxHeight: 116, duration: 0.4, ease: "power2.inOut",
                            onComplete: () => gsap.set(body, { clearProps: "maxHeight,overflow" }),
                        });
                        if (mask) gsap.to(mask, { opacity: 1, duration: 0.3, delay: 0.15 });
                    }
                }
            }
        };
        document.addEventListener("load", onLoad, true);
        document.addEventListener("click", onClick);
        return () => {
            document.removeEventListener("load", onLoad, true);
            document.removeEventListener("click", onClick);
        };
    }, []);

    return (
        <div style={styles.page}>
            <GlobalStyle />
            <header style={styles.header}>
                <div className="ven-header" style={styles.headerInner}>
                <div style={styles.side}>
                    <a href="/" style={styles.brand}>
                        <BrandMark />
                        ven-blog
                    </a>
                    <nav style={styles.nav} className="ven-header-nav">
                        <a href="/posts" style={styles.navLink} className="ven-nav-link">
                            文章
                        </a>
                        <a href="/moments" style={styles.navLink} className="ven-nav-link">
                            动态
                        </a>
                    </nav>
                </div>
                <HeaderSearch />
                <div style={styles.side} className="ven-header-actions">
                    <div className="ven-theme-wrap"><ThemeToggle /></div>
                    {role === "author" && (
                        <>
                            <a href="/admin" style={styles.navLink} title="后台">
                                <GridIcon style={{ verticalAlign: "-2px" }} />
                            </a>
                            <a href="/admin/posts/new" className="ven-btn ven-btn-primary">
                                <PenIcon />
                                写文章
                            </a>
                        </>
                    )}
                    {role ? (
                        <>
                            <ProfileEntry />
                            <button type="button" className="ven-btn" onClick={logout}>
                                <LogoutIcon />
                                注销（{role}）
                            </button>
                        </>
                    ) : (
                        <>
                            <button type="button" className="ven-btn" onClick={() => setAuthView("login")}>
                                <LoginIcon />
                                登录
                            </button>
                            <button type="button" className="ven-btn ven-btn-primary" onClick={() => setAuthView("register")}>
                                <UserPlusIcon />
                                注册
                            </button>
                        </>
                    )}
                    <button type="button" className="ven-menu-btn" onClick={() => setMenuOpen(true)} aria-label="打开菜单">
                        <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
                            <path d="M2 4.5 H16 M2 9 H16 M2 13.5 H16" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
                        </svg>
                    </button>
                </div>
                </div>
            </header>
            {/* 移动端抽屉：≤720px 汉堡按钮展开，收纳导航/主题/账号操作 */}
            {menuOpen && (
                <>
                    <div className="ven-drawer-overlay" onClick={() => setMenuOpen(false)} aria-hidden="true" />
                    <div className="ven-drawer" role="dialog" aria-modal="true" aria-label="导航菜单">
                        <div className="ven-drawer-head">
                            <a href="/" style={styles.brand} onClick={() => setMenuOpen(false)}>
                                <BrandMark />
                                ven-blog
                            </a>
                            <button type="button" className="ven-drawer-close" onClick={() => setMenuOpen(false)} aria-label="关闭菜单">
                                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                                    <path d="M3 3 L13 13 M13 3 L3 13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
                                </svg>
                            </button>
                        </div>
                        <div className="ven-drawer-body">
                            <nav className="ven-drawer-nav" aria-label="移动端导航">
                                <a className="ven-drawer-nav-link" href="/posts" onClick={() => setMenuOpen(false)}>
                                    文章
                                </a>
                                <a className="ven-drawer-nav-link" href="/moments" onClick={() => setMenuOpen(false)}>
                                    动态
                                </a>
                                <AuthorAvatar labeled onNavigate={() => setMenuOpen(false)} />
                            </nav>
                            <div className="ven-drawer-actions">
                                <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                                    <span className="ven-meta">主题</span>
                                    <div className="ven-theme-wrap"><ThemeToggle /></div>
                                </div>
                                {role === "author" && (
                                    <>
                                        <a href="/admin" className="ven-btn" onClick={() => setMenuOpen(false)}>
                                            <GridIcon />
                                            后台
                                        </a>
                                        <a href="/admin/posts/new" className="ven-btn ven-btn-primary" onClick={() => setMenuOpen(false)}>
                                            <PenIcon />
                                            写文章
                                        </a>
                                    </>
                                )}
                                {role ? (
                                    <>
                                        <div className="ven-drawer-profile" onClick={() => setMenuOpen(false)}>
                                            <ProfileEntry />
                                        </div>
                                        <button type="button" className="ven-btn" onClick={() => { logout(); setMenuOpen(false); }}>
                                            <LogoutIcon />
                                            注销（{role}）
                                        </button>
                                    </>
                                ) : (
                                    <>
                                        <button type="button" className="ven-btn" onClick={() => { setAuthView("login"); setMenuOpen(false); }}>
                                            <LoginIcon />
                                            登录
                                        </button>
                                        <button type="button" className="ven-btn ven-btn-primary" onClick={() => { setAuthView("register"); setMenuOpen(false); }}>
                                            <UserPlusIcon />
                                            注册
                                        </button>
                                    </>
                                )}
                            </div>
                        </div>
                    </div>
                </>
            )}
            <div style={styles.container}>
            <main style={styles.main}>
                <PageEnter>{children}</PageEnter>
            </main>
            <footer style={styles.footer}>
                <div style={styles.footerGrid}>
                    <div>
                        <a href="/" style={styles.brand}>
                            <BrandMark />
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
                                <a href="/admin">自留地</a>
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
                                    <RssIcon size={13} style={{ verticalAlign: "-2px", marginRight: 4 }} />
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
            <Modal open={authView !== null} onClose={() => setAuthView(null)} width={400}>
                {authView === "login" ? (
                    <div>
                        <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                            SIGN IN
                        </p>
                        <h2 style={{ fontSize: 20, marginBottom: 16 }}>登录</h2>
                        <LoginForm onSuccess={() => window.location.reload()} />
                        <p style={{ fontSize: 13, color: v.textSecondary, margin: "14px 0 0" }}>
                            没有账号？
                            <button
                                type="button"
                                onClick={() => setAuthView("register")}
                                style={{ border: "none", background: "none", padding: 0, cursor: "pointer", color: v.accent, font: "inherit", fontSize: 13 }}
                            >
                                去注册
                            </button>
                        </p>
                    </div>
                ) : (
                    <div>
                        <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                            SIGN UP
                        </p>
                        <h2 style={{ fontSize: 20, marginBottom: 16 }}>注册</h2>
                        <RegisterForm onSuccess={() => window.location.reload()} />
                        <p style={{ fontSize: 13, color: v.textSecondary, margin: "14px 0 0" }}>
                            已有账号？
                            <button
                                type="button"
                                onClick={() => setAuthView("login")}
                                style={{ border: "none", background: "none", padding: 0, cursor: "pointer", color: v.accent, font: "inherit", fontSize: 13 }}
                            >
                                去登录
                            </button>
                        </p>
                    </div>
                )}
            </Modal>
        </div>
    );
}
