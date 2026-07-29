/** 首页：博客简介 + 最近文章 */

import type { PageAppProps } from "./app/pageApp";
import { Layout } from "./lib/layout";
import { PostList } from "./posts/list";
import type { PostListState } from "./posts/types";

export default function HomePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { posts: [] }) as PostListState;
    return (
        <Layout>
            <h1>RyaoVen 的博客</h1>
            <p>基于 VenHybird 框架（Go 网关 + Node SSR 混合渲染）搭建的个人博客。</p>
            <h2>最近文章</h2>
            <PostList posts={state.posts} />
            <p style={{ marginTop: 24 }}>
                <a href="/posts" style={{ color: "#0969da", textDecoration: "none" }}>
                    全部文章 →
                </a>
            </p>
        </Layout>
    );
}
