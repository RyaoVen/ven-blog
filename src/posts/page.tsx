/** 文章列表页（ISR 静态页；数据变更由框架失效再生 + SSE 推送刷新，页面零接入） */

import type { PageAppProps } from "../app/pageApp";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";
import { PostList } from "./list";
import type { PostListState } from "./types";

export default function PostsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { posts: [] }) as PostListState;
    return (
        <Layout>
            <header style={{ marginBottom: 24 }}>
                <h1 style={{ fontSize: 28 }}>文章</h1>
                <p style={{ color: v.textSecondary, margin: 0 }}>共 {state.posts.length} 篇</p>
            </header>
            <PostList posts={state.posts} />
        </Layout>
    );
}
