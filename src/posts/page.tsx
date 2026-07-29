/** 文章列表页（ISR 静态页；数据变更由框架失效再生 + SSE 推送刷新，页面零接入） */

import type { PageAppProps } from "../app/pageApp";
import { Layout } from "../lib/layout";
import { PostList } from "./list";
import type { PostListState } from "./types";

export default function PostsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { posts: [] }) as PostListState;
    return (
        <Layout>
            <h1>文章</h1>
            <PostList posts={state.posts} />
        </Layout>
    );
}
