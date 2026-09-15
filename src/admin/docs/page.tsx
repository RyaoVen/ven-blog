/** docs 后台管理：树形列表（含 draft）+ 删除/排序/发布切换入口 */

import { useEffect, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import type { DocView, DocsTreeNode } from "../../docs/types";

interface AdminDocsState {
    docs: DocView[];
}

export default function AdminDocsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { docs: [] }) as AdminDocsState;
    const [docs, setDocs] = useState<DocView[]>(state.docs ?? []);
    void docs;
    const [error, setError] = useState("");

    async function reload() {
        const resp = await fetch("/admin/docs", { headers: { "X-Ven-Data-Only": "true" } });
        if (resp.ok) {
            setDocs(((await resp.json()) as AdminDocsState).docs ?? []);
        }
    }

    useEffect(() => {
        if (docs.length === 0 && state.docs?.length > 0) {
            setDocs(state.docs);
        }
    }, [state.docs]);

    async function remove(doc: DocView) {
        const hasKids = docs.some((d) => d.path.startsWith(doc.path + "/"));
        const msg = hasKids ? `「${doc.title}」存在子节点，级联删除？` : `删除「${doc.title}」？`;
        if (!confirm(msg)) {
            return;
        }
        const resp = await fetch(`/api/admin/docs/${doc.id}?recursive=${hasKids}`, { method: "DELETE" });
        if (resp.ok) {
            await reload();
        } else {
            setError(((await resp.json()) as { error?: string }).error ?? "删除失败");
        }
    }

    async function toggleStatus(doc: DocView) {
        const next = doc.status === "published" ? "draft" : "published";
        const resp = await fetch(`/admin/docs/${doc.id}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ status: next }),
        });
        if (resp.ok) {
            await reload();
        }
    }

    async function reorder(doc: DocView, delta: number) {
        const resp = await fetch(`/api/admin/docs/${doc.id}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ order: doc.sortOrder + delta }),
        });
        if (resp.ok) {
            await reload();
        }
    }

    const tree = buildTree(docs);
    const flat = docs;

    return (
        <AdminLayout route={bootstrap.route}>
            <header style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20 }}>
                <h1 style={{ fontSize: 26 }}>文档管理</h1>
                <a href="/admin/docs/new" className="ven-btn ven-btn-primary">
                    + 新建文档
                </a>
            </header>
            {error && <p style={{ color: "#c0392b" }}>{error}</p>}
            {flat.length === 0 ? (
                <p style={{ color: v.textSecondary }}>
                    还没有文档。用上方按钮新建，或通过 MCP <code>doc.create</code> 由 agent 创建。
                </p>
            ) : (
                <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
                    {tree.map((node) => (
                        <TreeRow key={node.doc.id} node={node} depth={0} docs={docs} onRemove={remove} onToggle={toggleStatus} onReorder={reorder} />
                    ))}
                </ul>
            )}
        </AdminLayout>
    );
}

function TreeRow({
    node,
    depth,
    docs,
    onRemove,
    onToggle,
    onReorder,
}: {
    node: DocsTreeNode;
    depth: number;
    docs: DocView[];
    onRemove: (d: DocView) => void;
    onToggle: (d: DocView) => void;
    onReorder: (d: DocView, delta: number) => void;
}) {
    const doc = docs.find((d) => d.id === node.doc.id) ?? toDocView(node);
    return (
        <li>
            <div
                style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    padding: "8px 10px",
                    paddingLeft: 10 + depth * 22,
                    borderBottom: `1px solid ${v.border}`,
                }}
            >
                <a
                    href={`/admin/docs/${doc.id}/edit`}
                    style={{ color: v.textPrimary, textDecoration: "none", flex: 1, fontWeight: doc.kind === "section" ? 600 : 400 }}
                >
                    {doc.kind === "section" ? "▸ " : ""}
                    {doc.title}
                    <span style={{ color: v.textSecondary, fontSize: 12, marginLeft: 8 }}>{doc.path}</span>
                </a>
                <button type="button" className="ven-meta" onClick={() => onReorder(doc, -1)} title="上移">
                    ↑
                </button>
                <button type="button" className="ven-meta" onClick={() => onReorder(doc, 1)} title="下移">
                    ↓
                </button>
                <button type="button" className="ven-meta" onClick={() => onToggle(doc)}>
                    {doc.status === "published" ? "撤稿" : "发布"}
                </button>
                <button type="button" className="ven-meta" onClick={() => onRemove(doc)}>
                    删除
                </button>
            </div>
            {node.children.map((child) => (
                <TreeRow key={child.doc.id} node={child} depth={depth + 1} docs={docs} onRemove={onRemove} onToggle={onToggle} onReorder={onReorder} />
            ))}
        </li>
    );
}

function toDocView(node: DocsTreeNode): DocView {
    return {
        id: node.doc.id,
        parentId: "0",
        slug: node.doc.slug,
        path: node.doc.path,
        kind: node.doc.kind,
        title: node.doc.title,
        summary: node.doc.summary,
        tags: node.doc.tags,
        sortOrder: node.doc.sortOrder,
        status: node.doc.status,
        createdAt: node.doc.updatedAt,
        updatedAt: node.doc.updatedAt,
    };
}

/** 前端构建树（按 parentId + sortOrder + slug，与 Go 侧树一致） */
function buildTree(docs: DocView[]): DocsTreeNode[] {
    const nodes = new Map<string, DocsTreeNode>();
    const roots: DocsTreeNode[] = [];
    for (const d of docs) {
        nodes.set(d.id, {
            doc: {
                id: d.id, path: d.path, slug: d.slug, kind: d.kind, title: d.title,
                summary: d.summary, sortOrder: d.sortOrder, status: d.status,
                tags: d.tags, updatedAt: d.updatedAt,
            },
            children: [],
        });
    }
    const sorted = [...docs].sort((a, b) => a.sortOrder - b.sortOrder || a.slug.localeCompare(b.slug));
    for (const d of sorted) {
        const node = nodes.get(d.id)!;
        const parent = d.parentId !== "0" ? nodes.get(d.parentId) : undefined;
        if (parent) {
            parent.children.push(node);
        } else {
            roots.push(node);
        }
    }
    return roots;
}
