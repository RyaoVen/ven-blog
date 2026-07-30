/** 文章卡片列表（首页与列表页共用） */

import { formatDateTime } from "../lib/format";
import { v } from "../lib/theme";
import type { Post } from "./types";

/** 摘要：优先用显式 summary，否则取正文前 90 字符（空白归一） */
function excerptOf(p: Post): string {
    const source = (p.summary || p.content).replace(/\s+/g, " ").trim();
    return source.length > 90 ? source.slice(0, 90) + "…" : source;
}

export function PostList({ posts }: { posts: Post[] }) {
    if (posts.length === 0) {
        return <p style={{ color: v.textSecondary }}>还没有文章。</p>;
    }
    return (
        <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: 16 }}>
            {posts.map((p) => (
                <li key={p.id} className="ven-card ven-card-hover" style={{ padding: "18px 22px" }}>
                    <a
                        href={`/posts/${p.id}`}
                        style={{ fontSize: 17, fontWeight: 650, color: v.text, textDecoration: "none" }}
                    >
                        {p.title}
                    </a>
                    <p style={{ margin: "8px 0 12px", fontSize: 14, color: v.textSecondary }}>{excerptOf(p)}</p>
                    <div style={{ display: "flex", gap: 12, alignItems: "center", fontSize: 13, color: v.textMuted }}>
                        <span>{p.authorName}</span>
                        <span>{formatDateTime(p.createdAt)}</span>
                        {p.tags.map((t) => (
                            <span key={t} className="ven-chip">
                                {t}
                            </span>
                        ))}
                    </div>
                </li>
            ))}
        </ul>
    );
}
