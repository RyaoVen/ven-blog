/** 站点通用布局：顶部导航 + 内容容器；按登录角色显隐入口 */

import { ReactNode } from "react";
import { navigate } from "../app/router";
import { useRole } from "./role";

const styles = {
    page: {
        maxWidth: 760,
        margin: "0 auto",
        padding: "0 20px 48px",
        fontFamily:
            '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
        color: "#24292f",
        lineHeight: 1.6,
    },
    header: {
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: "16px 0",
        borderBottom: "1px solid #d0d7de",
        marginBottom: 24,
    },
    brand: { fontWeight: 700, fontSize: 18, textDecoration: "none", color: "inherit" },
    nav: { display: "flex", gap: 16, alignItems: "center", fontSize: 14 },
    link: { textDecoration: "none", color: "#0969da" },
    button: {
        border: "1px solid #d0d7de",
        borderRadius: 6,
        background: "#f6f8fa",
        padding: "4px 10px",
        fontSize: 14,
        cursor: "pointer",
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
            <header style={styles.header}>
                <a href="/" style={styles.brand}>
                    ven-blog
                </a>
                <nav style={styles.nav}>
                    <a href="/posts" style={styles.link}>
                        文章
                    </a>
                    {role === "author" && (
                        <a href="/write" style={styles.link}>
                            写文章
                        </a>
                    )}
                    {role ? (
                        <button type="button" style={styles.button} onClick={logout}>
                            注销（{role}）
                        </button>
                    ) : (
                        <a href="/login" style={styles.link}>
                            登录
                        </a>
                    )}
                </nav>
            </header>
            <main>{children}</main>
        </div>
    );
}
