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
        borderRadius: 2,
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
        borderRadius: 2,
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
        paddingTop: 20,
        borderTop: `1px solid ${v.border}`,
        display: "flex",
        justifyContent: "space-between",
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
            <footer style={styles.footer} className="ven-meta">
                <span>© 2026 RYAOVEN</span>
                <span>POWERED BY VENHYBIRD</span>
            </footer>
        </div>
    );
}
