/** 文档首页：published 目录树总览 */

import type { PageAppProps } from "../app/pageApp";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";
import { DocTree } from "./DocView";
import type { DocsHomeState } from "./types";

export default function DocsHomePage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { tree: [] }) as DocsHomeState;
    const tree = state.tree ?? [];
    return (
        <Layout>
            <header style={{ marginBottom: 24 }}>
                <h1 style={{ fontSize: 30 }}>文档</h1>
                <p style={{ color: v.textSecondary }}>笔记与项目文档的目录总览。</p>
            </header>
            <DocTree nodes={tree} currentPath="" depth={0} />
        </Layout>
    );
}
