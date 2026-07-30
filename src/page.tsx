/** 首页：hero 区（工业极简：发丝线 + 排版驱动）+ 最近文章 */

import type { PageAppProps } from "./app/pageApp";
import { Layout } from "./lib/layout";
import { v } from "./lib/theme";
import { PostList } from "./posts/list";
import type { PostListState } from "./posts/types";

export default function HomePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { posts: [] }) as PostListState;
    return (
        <Layout>
            <section
                style={{
                    borderTop: `1px solid ${v.text}`,
                    borderBottom: `1px solid ${v.border}`,
                    padding: "56px 0 48px",
                    marginBottom: 40,
                }}
            >
                <p className="ven-meta" style={{ margin: 0 }}>
                    PERSONAL SITE / VEN-BLOG
                </p>
                <h1 style={{ fontSize: 44, letterSpacing: "-0.03em", margin: "14px 0 16px" }}>
                    RyaoVen 的博客
                </h1>
                <p style={{ fontSize: 16, color: v.textSecondary, maxWidth: 620, marginBottom: 28 }}>
                    记录技术与生活。本站由自研 VenHybird 框架驱动——SSR 直出、SPA 接管、ISR 物化、SSE 实时推送。
                </p>
                <div style={{ display: "flex", gap: 12 }}>
                    <a href="/posts" className="ven-btn ven-btn-primary">
                        开始阅读
                    </a>
                    <a
                        href="https://github.com/RyaoVen/ven_hybird"
                        className="ven-btn"
                        target="_blank"
                        rel="noreferrer"
                    >
                        了解框架
                    </a>
                </div>
            </section>
            <section>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 20 }}>
                    <h2 style={{ fontSize: 20, margin: 0 }}>最近文章</h2>
                    <a href="/posts" className="ven-meta" style={{ textDecoration: "none" }}>
                        全部文章 →
                    </a>
                </div>
                <PostList posts={state.posts} />
            </section>
        </Layout>
    );
}
