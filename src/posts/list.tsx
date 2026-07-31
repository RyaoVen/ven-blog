/** 文章卡片列表（首页与列表页共用）；有封面时卡片顶部显示封面图（blur-up 由全局 load 监听解除） */

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
                <li key={p.id} className="ven-card ven-card-hover" style={{ overflow: "hidden" }}>
                    {p.coverUrl && (
                        <img
                            src={p.coverUrl}
                            alt={p.title}
                            loading="lazy"
                            style={{
                                display: "block",
                                width: "100%",
                                height: 150,
                                objectFit: "cover",
                                borderBottom: `1px solid ${v.border}`,
                            }}
                        />
                    )}
                    <div style={{ padding: "18px 22px" }}>
                        <a
                            href={`/posts/${p.id}`}
                            className="ven-hl"
                            style={{ fontSize: 17, fontWeight: 650, color: v.text }}
                        >
                            {p.title}
                        </a>
                        <p style={{ margin: "8px 0 12px", fontSize: 14, color: v.textSecondary }}>{excerptOf(p)}</p>
                        <div className="ven-meta" style={{ display: "flex", gap: 14, alignItems: "center" }}>
                            <span>{p.authorName}</span>
                            <span>{formatDateTime(p.createdAt)}</span>
                            {p.tags.map((t) => (
                                <a
                                    key={t}
                                    className="ven-chip"
                                    href={`/posts?tag=${encodeURIComponent(t)}`}
                                    style={{ textDecoration: "none" }}
                                >
                                    {t}
                                </a>
                            ))}
                        </div>
                    </div>
                </li>
            ))}
        </ul>
    );
}
