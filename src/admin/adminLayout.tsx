/** 后台子布局：标题 + 管理 tabs（文章 tab 在编辑器页保持激活） */

import { ReactNode } from "react";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";

const TABS = [
    { href: "/admin", label: "数据面板", exact: true },
    { href: "/admin/posts", label: "文章", exact: false },
    { href: "/admin/comments", label: "评论", exact: false },
    { href: "/admin/moments", label: "动态", exact: false },
] as const;

export function AdminLayout({ route, children }: { route: string; children: ReactNode }) {
    return (
        <Layout>
            <div
                style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "baseline",
                    flexWrap: "wrap",
                    gap: 12,
                    paddingBottom: 16,
                    marginBottom: 24,
                    borderBottom: `1px solid ${v.text}`,
                }}
            >
                <h1 style={{ fontSize: 24, margin: 0 }}>后台管理</h1>
                <nav style={{ display: "flex", gap: 18 }}>
                    {TABS.map((t) => {
                        const active = t.exact ? route === t.href : route.startsWith(t.href);
                        return (
                            <a
                                key={t.href}
                                href={t.href}
                                className="ven-meta"
                                style={{
                                    textDecoration: "none",
                                    color: active ? v.text : v.textMuted,
                                    fontWeight: active ? 700 : 400,
                                    borderBottom: active ? `2px solid ${v.text}` : "2px solid transparent",
                                    paddingBottom: 2,
                                }}
                            >
                                {t.label}
                            </a>
                        );
                    })}
                </nav>
            </div>
            {children}
        </Layout>
    );
}
