/** 站点通用布局：全局样式注入 + 顶部导航 + 内容容器 + 页脚；按登录角色显隐入口 */

import { ReactNode } from "react";
import { navigate } from "../app/router";
import { globalCss } from "./globalCss";
import { useRole } from "./role";
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
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: "14px 0",
        borderBottom: `1px solid ${v.border}`,
        marginBottom: 32,
    },
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
    nav: { display: "flex", gap: 18, alignItems: "center", fontSize: 14 },
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
            <header style={styles.header}>
                <a href="/" style={styles.brand}>
                    <span style={styles.brandDot}>V</span>
                    ven-blog
                </a>
                <nav style={styles.nav}>
                    <a href="/posts" style={styles.navLink}>
                        文章
                    </a>
                    {role === "author" && (
                        <a href="/write" className="ven-btn ven-btn-primary">
                            写文章
                        </a>
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
                </nav>
            </header>
            <main style={styles.main}>{children}</main>
            <footer style={styles.footer} className="ven-meta">
                <span>© 2026 RYAOVEN</span>
                <span>POWERED BY VENHYBIRD</span>
            </footer>
        </div>
    );
}
