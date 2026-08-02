/** 后台子布局：独立门户（不套博客导航/页脚）——固定侧边栏 + 主内容区 */

import { ReactNode } from "react";
import { useCardGlow } from "../lib/cardGlow";
import { GlobalStyle } from "../lib/layout";
import { LogoutIcon } from "../lib/icons";
import { PageEnter } from "../lib/motion";
import { v } from "../lib/theme";

const TABS = [
    { href: "/admin", label: "数据面板", exact: true },
    { href: "/admin/posts", label: "文章", exact: false },
    { href: "/admin/comments", label: "评论", exact: false },
    { href: "/admin/guestbook", label: "留言", exact: false },
    { href: "/admin/moments", label: "动态", exact: false },
    { href: "/admin/author", label: "个人主页", exact: false },
    { href: "/admin/settings", label: "设置", exact: false },
] as const;

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

const styles = {
    shell: { minHeight: "100vh" },
    sidebar: {
        position: "fixed",
        top: 0,
        left: 0,
        bottom: 0,
        width: 208,
        background: "var(--bg-subtle)",
        borderRight: `1px solid ${v.border}`,
        display: "flex",
        flexDirection: "column",
        padding: "22px 14px",
        zIndex: 50,
    },
    brand: {
        display: "flex",
        alignItems: "center",
        gap: 10,
        fontWeight: 700,
        fontSize: 15,
        color: v.text,
        textDecoration: "none",
        padding: "0 8px",
        marginBottom: 28,
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
        fontFamily: mono,
    },
    nav: { display: "flex", flexDirection: "column", gap: 2, flex: 1 },
    bottom: { display: "flex", flexDirection: "column", gap: 2, borderTop: `1px solid ${v.border}`, paddingTop: 12 },
    main: { marginLeft: 208, padding: "36px 44px 64px", minHeight: "100vh" },
} as const;

/** 注销并回登入页 */
async function logout() {
    await fetch("/auth/logout", { method: "POST" });
    window.location.href = "/admin/login";
}

export function AdminLayout({ route, children }: { route: string; children: ReactNode }) {
    useCardGlow();
    return (
        <div style={styles.shell}>
            <GlobalStyle />
            <aside className="ven-admin-side" style={styles.sidebar}>
                <a href="/admin" style={styles.brand}>
                    <span style={styles.brandDot}>V</span>
                    后台管理
                </a>
                <nav style={styles.nav}>
                    {TABS.map((t) => (
                        <SideLink key={t.href} href={t.href} label={t.label} active={t.exact ? route === t.href : route.startsWith(t.href)} />
                    ))}
                    <div style={{ marginTop: 18 }}>
                        <SideLink href="/admin/posts/new" label="+ 新建文章" active={route === "/admin/posts/new"} />
                    </div>
                </nav>
                <div style={styles.bottom}>
                    <SideLink href="/" label="← 返回博客" active={false} />
                    <button
                        type="button"
                        onClick={logout}
                        className="ven-meta"
                        style={{
                            display: "flex",
                            alignItems: "center",
                            gap: 8,
                            padding: "8px 10px",
                            border: "none",
                            background: "none",
                            cursor: "pointer",
                            color: v.textSecondary,
                            fontSize: 13,
                            textAlign: "left",
                        }}
                    >
                        <LogoutIcon size={13} />
                        注销
                    </button>
                </div>
            </aside>
            <main className="ven-admin-main" style={styles.main}>
                <PageEnter>{children}</PageEnter>
            </main>
        </div>
    );
}

/** 侧边栏链接（激活态玉青竖线 + 加粗） */
function SideLink({ href, label, active }: { href: string; label: string; active: boolean }) {
    return (
        <a
            href={href}
            style={{
                display: "block",
                padding: "8px 10px",
                fontSize: 14,
                textDecoration: "none",
                borderRadius: 3,
                color: active ? v.accent : v.textSecondary,
                fontWeight: active ? 650 : 400,
                background: active ? "var(--bg-inset)" : "transparent",
                borderLeft: active ? `2px solid ${v.accent}` : "2px solid transparent",
            }}
        >
            {label}
        </a>
    );
}
