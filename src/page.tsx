/** 首页：hero 区 + 最近文章 */

import type { PageAppProps } from "./app/pageApp";
import { Layout } from "./lib/layout";
import { radius, v } from "./lib/theme";
import { PostList } from "./posts/list";
import type { PostListState } from "./posts/types";

export default function HomePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { posts: [] }) as PostListState;
    return (
        <Layout>
            <section
                style={{
                    padding: "52px 36px",
                    background: v.bgSubtle,
                    border: `1px solid ${v.border}`,
                    borderRadius: radius.lg,
                    marginBottom: 40,
                }}
            >
                <h1 style={{ fontSize: 34, marginBottom: 14 }}>RyaoVen 的博客</h1>
                <p style={{ fontSize: 16, color: v.textSecondary, maxWidth: 600, marginBottom: 24 }}>
                    记录技术与生活。本站由自研 VenHybird 框架驱动：SSR 直出、SPA 接管、ISR 物化、SSE 实时推送。
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
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 16 }}>
                    <h2 style={{ fontSize: 20, margin: 0 }}>最近文章</h2>
                    <a href="/posts" style={{ fontSize: 14 }}>
                        全部文章 →
                    </a>
                </div>
                <PostList posts={state.posts} />
            </section>
        </Layout>
    );
}
