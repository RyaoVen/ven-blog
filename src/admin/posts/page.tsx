/** 后台-文章管理：列表 + 新建入口 + 编辑/删除 */

import { useMemo, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { formatDateTime } from "../../lib/format";
import { TrashIcon } from "../../lib/icons";
import { ConfirmModal } from "../../lib/modal";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import { FilterSelect, SearchBar } from "../searchBar";
import type { AdminPostsState } from "../types";
import type { Post } from "../../posts/types";

export default function AdminPostsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { posts: [] }) as AdminPostsState;
    const [list, setList] = useState(state.posts);
    const [deleting, setDeleting] = useState<Post | null>(null);
    const [keyword, setKeyword] = useState("");
    const [category, setCategory] = useState("");

    const categories = useMemo(() => [...new Set(list.map((p) => p.category).filter(Boolean))], [list]);
    const filtered = useMemo(
        () =>
            list.filter(
                (p) =>
                    (!keyword || p.title.toLowerCase().includes(keyword.toLowerCase())) &&
                    (!category || p.category === category),
            ),
        [list, keyword, category],
    );

    async function confirmDelete() {
        if (!deleting) {
            return;
        }
        const resp = await fetch(`/api/posts/${deleting.id}`, { method: "DELETE" });
        if (resp.ok) {
            setList((l) => l.filter((p) => p.id !== deleting.id));
        }
        setDeleting(null);
    }

    return (
        <AdminLayout route={bootstrap.route}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20 }}>
                <p className="ven-meta" style={{ margin: 0 }}>
                    共 {filtered.length} / {list.length} 篇
                </p>
                <a href="/admin/posts/new" className="ven-btn ven-btn-primary">
                    新建文章
                </a>
            </div>
            <SearchBar keyword={keyword} onKeyword={setKeyword} placeholder="搜索标题…">
                <FilterSelect
                    value={category}
                    onChange={setCategory}
                    options={[{ value: "", label: "全部分类" }, ...categories.map((c) => ({ value: c, label: c }))]}
                />
            </SearchBar>
            {filtered.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>没有匹配的文章。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {filtered.map((p) => (
                        <li
                            key={p.id}
                            style={{
                                display: "flex",
                                alignItems: "baseline",
                                gap: 14,
                                padding: "12px 0",
                                borderBottom: `1px solid ${v.border}`,
                            }}
                        >
                            <a
                                href={`/posts/${p.id}`}
                                style={{
                                    fontWeight: 600,
                                    fontSize: 15,
                                    color: v.text,
                                    textDecoration: "none",
                                    flex: 1,
                                    overflow: "hidden",
                                    textOverflow: "ellipsis",
                                    whiteSpace: "nowrap",
                                }}
                            >
                                {p.title}
                            </a>
                            <span className="ven-meta" style={{ flexShrink: 0 }}>
                                {formatDateTime(p.createdAt)}
                            </span>
                            <a href={`/admin/posts/${p.id}/edit`} className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }}>
                                编辑
                            </a>
                            <button
                                type="button"
                                className="ven-btn ven-btn-danger"
                                style={{ padding: "3px 12px", fontSize: 12 }}
                                onClick={() => setDeleting(p)}
                            >
                                <TrashIcon size={12} />
                                删除
                            </button>
                        </li>
                    ))}
                </ul>
            )}
            <ConfirmModal
                open={deleting !== null}
                title="删除文章"
                message={`确定删除《${deleting?.title}》吗？文章与其评论将一并删除，不可恢复。`}
                confirmText="删除"
                danger
                onCancel={() => setDeleting(null)}
                onConfirm={confirmDelete}
            />
        </AdminLayout>
    );
}
