/** docs 共享组件：侧边目录树 + 文档正文 + 上一页/下一页（供 /docs 固定深度页面薄壳复用） */

import { useMemo } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { formatDateTime } from "../lib/format";
import { renderMarkdown } from "../lib/markdown";
import { markdownCss } from "../lib/markdownCss";
import { v } from "../lib/theme";
import type { DocPageState, DocsTreeNode } from "./types";

/** DocDetailPage 文档详情视图（三段固定深度页面共用） */
export function DocDetailPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { doc: null, children: [], tree: [], prev: null, next: null }) as DocPageState;
    const doc = state.doc;
    const rendered = useMemo(() => (doc?.content ? renderMarkdown(doc.content) : null), [doc]);

    if (!doc) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>文档不存在或未发布。</p>
            </Layout>
        );
    }

    const renderedHtml = rendered?.html ?? "";

    return (
        <Layout>
            <style>{markdownCss}</style>
            <div style={{ display: "flex", gap: 32, alignItems: "flex-start" }}>
                <aside style={{ width: 240, flexShrink: 0, position: "sticky", top: 80 }} className="ven-docs-sidebar">
                    <DocTree nodes={state.tree} currentPath={doc.path} depth={0} />
                </aside>
                <article style={{ flex: 1, minWidth: 0 }} className="ven-docs-article">
                    <p style={{ color: v.textSecondary, fontSize: 13, margin: "0 0 8px" }}>
                        {doc.path}
                    </p>
                    <h1 style={{ fontSize: 30, marginBottom: 12 }}>{doc.title}</h1>
                    {doc.summary && (
                        <p style={{ color: v.textSecondary, marginBottom: 20 }}>{doc.summary}</p>
                    )}
                    {renderedHtml ? (
                        <div className="ven-md" dangerouslySetInnerHTML={{ __html: renderedHtml }} />
                    ) : (
                        doc.kind === "section" && (
                            <p style={{ color: v.textSecondary }}>这是一个目录，从侧边栏选择子文档阅读。</p>
                        )
                    )}
                    <div
                        style={{
                            display: "flex",
                            justifyContent: "space-between",
                            gap: 16,
                            marginTop: 40,
                            paddingTop: 16,
                            borderTop: `1px solid ${v.border}`,
                        }}
                    >
                        {state.prev ? (
                            <a
                                href={`/docs/${state.prev.path}`}
                                onClick={(e) => {
                                    e.preventDefault();
                                    navigate(`/docs/${state.prev!.path}`);
                                }}
                                style={{ color: v.textPrimary, textDecoration: "none" }}
                            >
                                ← {state.prev.title}
                            </a>
                        ) : (
                            <span />
                        )}
                        {state.next ? (
                            <a
                                href={`/docs/${state.next.path}`}
                                onClick={(e) => {
                                    e.preventDefault();
                                    navigate(`/docs/${state.next!.path}`);
                                }}
                                style={{ color: v.textPrimary, textDecoration: "none", textAlign: "right" }}
                            >
                                {state.next.title} →
                            </a>
                        ) : (
                            <span />
                        )}
                    </div>
                    <p style={{ color: v.textSecondary, fontSize: 12, marginTop: 24 }}>
                        更新于 {formatDateTime(doc.updatedAt)}
                    </p>
                </article>
            </div>
        </Layout>
    );
}

/** DocTree 递归目录树（当前路径高亮；目录可折叠默认展开第一层） */
export function DocTree({ nodes, currentPath, depth }: { nodes: DocsTreeNode[]; currentPath: string; depth: number }) {
    if (nodes.length === 0) {
        return depth === 0 ? <p style={{ color: v.textSecondary }}>暂无文档。</p> : null;
    }
    return (
        <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
            {nodes.map((node) => {
                const active = node.doc.path === currentPath;
                return (
                    <li key={node.doc.id}>
                        <a
                            href={`/docs/${node.doc.path}`}
                            onClick={(e) => {
                                e.preventDefault();
                                navigate(`/docs/${node.doc.path}`);
                            }}
                            style={{
                                display: "block",
                                padding: "6px 8px",
                                borderRadius: 8,
                                fontSize: 14,
                                fontWeight: node.doc.kind === "section" ? 600 : 400,
                                color: active ? v.textPrimary : v.textSecondary,
                                background: active ? v.surfaceRaised ?? "rgba(128,128,128,0.12)" : "transparent",
                                textDecoration: "none",
                            }}
                        >
                            {node.doc.kind === "section" ? "▸ " : ""}
                            {node.doc.title}
                        </a>
                        {node.children.length > 0 && (
                            <div style={{ paddingLeft: 14 }}>
                                <DocTree nodes={node.children} currentPath={currentPath} depth={depth + 1} />
                            </div>
                        )}
                    </li>
                );
            })}
        </ul>
    );
}
